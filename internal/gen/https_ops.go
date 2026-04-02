package gen

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/arastu/irtrafficgen/internal/config"
	"github.com/arastu/irtrafficgen/internal/version"
	"golang.org/x/net/http2"
	"golang.org/x/time/rate"
)

type TrafficResult struct {
	Rx int64
	Tx int64
}

type OpLimiter struct {
	Receive *rate.Limiter
	Send    *rate.Limiter
}

func buildHTTPSURL(host string, ip net.IP, path string) (u string, serverName string, dialIP net.IP, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if host != "" {
		return "https://" + host + path, host, nil, nil
	}
	if ip == nil {
		return "", "", nil, fmt.Errorf("no host or ip")
	}
	if ip.To4() != nil {
		return "https://" + ip.String() + path, "", ip, nil
	}
	return "https://[" + ip.String() + "]" + path, "", ip, nil
}

func newHTTPSClient(host string, ip net.IP, sniForIP string, timeout time.Duration, insecure bool, ac *config.Asymmetric) (*http.Client, error) {
	var serverName string
	if host != "" {
		serverName = host
	} else if ip != nil {
		serverName = strings.TrimSpace(sniForIP)
	}
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: serverName,
	}
	if insecure {
		tlsCfg.InsecureSkipVerify = true
	}
	maxIdle := 32
	idleSec := 90
	maxRedir := 10
	if ac != nil && ac.Enabled {
		maxIdle = ac.TransportMaxIdleConnsPerHost
		idleSec = ac.TransportIdleConnTimeoutSec
		maxRedir = ac.MaxRedirects
	}
	tr := &http.Transport{
		TLSClientConfig:     tlsCfg,
		MaxIdleConnsPerHost: maxIdle,
		IdleConnTimeout:     time.Duration(idleSec) * time.Second,
	}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{Timeout: timeout}
		if host != "" {
			return d.DialContext(ctx, network, addr)
		}
		return d.DialContext(ctx, network, net.JoinHostPort(ip.String(), "443"))
	}
	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, fmt.Errorf("http2 configure transport: %w", err)
	}
	cli := &http.Client{
		Transport: tr,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedir {
				return fmt.Errorf("stopped after %d redirects", maxRedir)
			}
			return nil
		},
	}
	return cli, nil
}

func HTTPSOperation(ctx context.Context, cfg *config.Config, host string, ip net.IP, op OpKind, lim *OpLimiter) (TrafficResult, error) {
	if cfg == nil {
		return TrafficResult{}, fmt.Errorf("nil config")
	}
	to := time.Duration(cfg.Limits.HTTPSTimeout) * time.Second
	var ac *config.Asymmetric
	if cfg.Asymmetric.Enabled {
		ac = &cfg.Asymmetric
	}
	cli, err := newHTTPSClient(host, ip, cfg.SNIForIP, to, cfg.InsecureTLS, ac)
	if err != nil {
		return TrafficResult{}, err
	}
	if !cfg.Asymmetric.Enabled {
		op = OpHead
	}
	getPath := "/"
	postPath := "/"
	if ac != nil {
		getPath = ac.GetPath
		postPath = ac.PostPath
	}
	u, _, _, err := buildHTTPSURL(host, ip, pathForOp(op, getPath, postPath))
	if err != nil {
		return TrafficResult{}, err
	}
	switch op {
	case OpHead:
		return doHead(ctx, cli, u, host, cfg)
	case OpGet:
		return doGet(ctx, cli, u, host, cfg, lim)
	case OpPost:
		return doPost(ctx, cli, u, host, cfg, lim)
	default:
		return doHead(ctx, cli, u, host, cfg)
	}
}

func pathForOp(op OpKind, getPath, postPath string) string {
	if op == OpPost {
		return postPath
	}
	if op == OpGet {
		return getPath
	}
	return "/"
}

func HTTPSURLForLog(host string, ip net.IP, op OpKind, cfg *config.Config) string {
	gp, pp := "/", "/"
	if cfg != nil && cfg.Asymmetric.Enabled {
		gp = cfg.Asymmetric.GetPath
		pp = cfg.Asymmetric.PostPath
	}
	u, _, _, err := buildHTTPSURL(host, ip, pathForOp(op, gp, pp))
	if err != nil {
		return ""
	}
	return u
}

func userAgentForRequest(cfg *config.Config) string {
	if cfg != nil {
		if s := strings.TrimSpace(cfg.UserAgent); s != "" {
			return s
		}
	}
	return "irtrafficgen/" + version.Version
}

func setRequestUserAgent(req *http.Request, cfg *config.Config) {
	req.Header.Set("User-Agent", userAgentForRequest(cfg))
}

func doHead(ctx context.Context, cli *http.Client, u, host string, cfg *config.Config) (TrafficResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return TrafficResult{}, err
	}
	if host != "" {
		req.Host = host
	}
	setRequestUserAgent(req, cfg)
	resp, err := cli.Do(req)
	if err != nil {
		return TrafficResult{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return TrafficResult{}, fmt.Errorf("http status %d", resp.StatusCode)
	}
	var rx, tx int64
	if cfg.Asymmetric.Enabled {
		rx = cfg.Asymmetric.HeadEstimateRxBytes
		tx = cfg.Asymmetric.HeadEstimateTxBytes
	}
	return TrafficResult{Rx: rx, Tx: tx}, nil
}

func doGet(ctx context.Context, cli *http.Client, u, host string, cfg *config.Config, lim *OpLimiter) (TrafficResult, error) {
	a := cfg.Asymmetric
	max := a.DownloadMaxBytes
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return TrafficResult{}, err
	}
	if host != "" {
		req.Host = host
	}
	setRequestUserAgent(req, cfg)
	resp, err := cli.Do(req)
	if err != nil {
		return TrafficResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return TrafficResult{}, fmt.Errorf("http status %d", resp.StatusCode)
	}
	body := resp.Body
	if lim != nil && lim.Receive != nil {
		body = io.NopCloser(&rxByteLimiter{ctx: ctx, r: body, lim: lim.Receive})
	}
	lr := io.LimitReader(body, max+1)
	n, err := io.Copy(io.Discard, lr)
	if err != nil && !errors.Is(err, io.EOF) {
		return TrafficResult{}, err
	}
	if n > max {
		return TrafficResult{}, fmt.Errorf("response body exceeds download_max_bytes")
	}
	tx := int64(512)
	return TrafficResult{Rx: n, Tx: tx}, nil
}

func doPost(ctx context.Context, cli *http.Client, u, host string, cfg *config.Config, lim *OpLimiter) (TrafficResult, error) {
	a := cfg.Asymmetric
	up := a.UploadMaxBytes
	zr := io.LimitReader(zeroReader{}, up)
	var body io.Reader = zr
	if lim != nil && lim.Send != nil {
		body = &txByteLimiter{ctx: ctx, r: zr, lim: lim.Send}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return TrafficResult{}, err
	}
	if host != "" {
		req.Host = host
	}
	setRequestUserAgent(req, cfg)
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := cli.Do(req)
	if err != nil {
		return TrafficResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return TrafficResult{}, fmt.Errorf("http status %d", resp.StatusCode)
	}
	limR := io.LimitReader(resp.Body, 65537)
	n, err := io.Copy(io.Discard, limR)
	if err != nil && !errors.Is(err, io.EOF) {
		return TrafficResult{}, err
	}
	if n > 65536 {
		return TrafficResult{}, fmt.Errorf("post response body too large")
	}
	return TrafficResult{Rx: n, Tx: up}, nil
}
