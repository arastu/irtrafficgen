package gen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
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
