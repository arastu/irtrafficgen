package gen

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/arastu/irtrafficgen/internal/config"
)

func HTTPSHead(ctx context.Context, host string, ip net.IP, sniForIP string, timeout time.Duration, insecure bool) error {
	cli, err := newHTTPSClient(host, ip, sniForIP, timeout, insecure, nil)
	if err != nil {
		return err
	}
	u, _, _, err := buildHTTPSURL(host, ip, "/")
	if err != nil {
		return err
	}
	cfg := &config.Config{}
	_, err = doHead(ctx, cli, u, host, cfg)
	return err
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
