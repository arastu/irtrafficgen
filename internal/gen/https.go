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

	"golang.org/x/net/http2"
)

func HTTPSHead(ctx context.Context, host string, ip net.IP, sniForIP string, timeout time.Duration, insecure bool) error {
	var u string
	var serverName string
	if host != "" {
		u = "https://" + host + "/"
		serverName = host
	} else if ip != nil {
		if ip.To4() != nil {
			u = "https://" + ip.String() + "/"
		} else {
			u = "https://[" + ip.String() + "]/"
		}
		serverName = strings.TrimSpace(sniForIP)
	} else {
		return fmt.Errorf("no host or ip")
	}
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: serverName,
	}
	if insecure {
		tlsCfg.InsecureSkipVerify = true
	}
	tr := &http.Transport{
		TLSClientConfig: tlsCfg,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			if host != "" {
				return d.DialContext(ctx, network, addr)
			}
			return d.DialContext(ctx, network, net.JoinHostPort(ip.String(), "443"))
		},
	}
	if err := http2.ConfigureTransport(tr); err != nil {
		return fmt.Errorf("http2 configure transport: %w", err)
	}
	client := &http.Client{Transport: tr, Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return err
	}
	if host != "" {
		req.Host = host
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return nil
}

func ClassifyHTTPSError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	var tlsErr tls.RecordHeaderError
	if errors.As(err, &tlsErr) {
		return "tls"
	}
	if strings.Contains(strings.ToLower(err.Error()), "tls") ||
		strings.Contains(strings.ToLower(err.Error()), "certificate") {
		return "tls"
	}
	if strings.Contains(strings.ToLower(err.Error()), "http status") {
		return "http"
	}
	return "other"
}
