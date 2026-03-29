package target

import (
	"testing"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func TestHostForDomain(t *testing.T) {
	cases := []struct {
		d      *routercommon.Domain
		www    bool
		want   string
		wantErr bool
	}{
		{&routercommon.Domain{Type: routercommon.Domain_Full, Value: "X.COM"}, false, "x.com", false},
		{&routercommon.Domain{Type: routercommon.Domain_RootDomain, Value: "ex.ir"}, false, "ex.ir", false},
		{&routercommon.Domain{Type: routercommon.Domain_RootDomain, Value: "ex.ir"}, true, "www.ex.ir", false},
		{&routercommon.Domain{Type: routercommon.Domain_Plain, Value: "ok.co.uk"}, false, "ok.co.uk", false},
		{&routercommon.Domain{Type: routercommon.Domain_Plain, Value: "bad"}, false, "", true},
		{&routercommon.Domain{Type: routercommon.Domain_Regex, Value: ".*"}, false, "", true},
	}
	for _, tc := range cases {
		got, err := HostForDomain(tc.d, tc.www)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("want err for %+v", tc.d)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("%+v: got %q err=%v want %q", tc.d, got, err, tc.want)
		}
	}
}

func TestHostAllowed(t *testing.T) {
	if !HostAllowed("a.b.ir", []string{"ir"}) {
		t.Fatal()
	}
	if HostAllowed("a.com", []string{"ir"}) {
		t.Fatal()
	}
	if !HostAllowed("a.com", nil) {
		t.Fatal()
	}
}
