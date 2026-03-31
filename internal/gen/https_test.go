package gen

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/arastu/irtrafficgen/internal/config"
)

func TestHTTPSHeadLocal(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	parsed, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	host := parsed.Host
	ctx := context.Background()
	if hErr := HTTPSHead(ctx, host, nil, "", 5*time.Second, true); hErr != nil {
		t.Fatal(hErr)
	}
}

func asymBase() config.Asymmetric {
	return config.Asymmetric{
		Enabled:                      true,
		DownloadMaxBytes:             1024,
		UploadMaxBytes:               1024,
		OperationWeights:             config.OpWeights{Head: 1, Get: 1, Post: 1},
		MaxConcurrentLargeDownloads:  2,
		MaxRedirects:                 3,
		TransportMaxIdleConnsPerHost: 32,
		TransportIdleConnTimeoutSec:  90,
		GetPath:                      "/",
		PostPath:                     "/",
		HeadEstimateRxBytes:          100,
		HeadEstimateTxBytes:          100,
	}
}

func TestHTTPSGetBodyExceedsCap(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 200))
	}))
	defer ts.Close()
	parsed, _ := url.Parse(ts.URL)
	a := asymBase()
	a.DownloadMaxBytes = 50
	cfg := &config.Config{
		Limits:     config.Limits{HTTPSTimeout: 10},
		InsecureTLS: true,
		Asymmetric: a,
	}
	ctx := context.Background()
	_, err := HTTPSOperation(ctx, cfg, parsed.Host, nil, OpGet, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHTTPSGetBodyUnderCap(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 30))
	}))
	defer ts.Close()
	parsed, _ := url.Parse(ts.URL)
	a := asymBase()
	a.DownloadMaxBytes = 100
	cfg := &config.Config{
		Limits:     config.Limits{HTTPSTimeout: 10},
		InsecureTLS: true,
		Asymmetric: a,
	}
	ctx := context.Background()
	res, err := HTTPSOperation(ctx, cfg, parsed.Host, nil, OpGet, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Rx != 30 {
		t.Fatalf("rx=%d", res.Rx)
	}
}

func TestHTTPSPostUploadSize(t *testing.T) {
	var got int64
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		n, _ := io.Copy(io.Discard, r.Body)
		got = n
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	parsed, _ := url.Parse(ts.URL)
	a := asymBase()
	a.UploadMaxBytes = 400
	cfg := &config.Config{
		Limits:     config.Limits{HTTPSTimeout: 10},
		InsecureTLS: true,
		Asymmetric: a,
	}
	ctx := context.Background()
	res, err := HTTPSOperation(ctx, cfg, parsed.Host, nil, OpPost, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != 400 {
		t.Fatalf("server read %d bytes", got)
	}
	if res.Tx != 400 {
		t.Fatalf("tx=%d", res.Tx)
	}
}

func TestHTTPSRedirectCap(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Location", "/a")
			w.WriteHeader(http.StatusFound)
		case "/a":
			w.Header().Set("Location", "/b")
			w.WriteHeader(http.StatusFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()
	parsed, _ := url.Parse(ts.URL)
	a := asymBase()
	a.MaxRedirects = 2
	cfg := &config.Config{
		Limits:     config.Limits{HTTPSTimeout: 10},
		InsecureTLS: true,
		Asymmetric: a,
	}
	ctx := context.Background()
	_, err := HTTPSOperation(ctx, cfg, parsed.Host, nil, OpGet, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
