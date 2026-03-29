package geo

import (
	"testing"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

func TestLoadEmbeddedGeoSite(t *testing.T) {
	list, err := LoadEmbeddedGeoSite()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.GetEntry()) == 0 {
		t.Fatal("empty geosite")
	}
}

func TestLoadEmbeddedGeoIP(t *testing.T) {
	list, err := LoadEmbeddedGeoIP()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.GetEntry()) == 0 {
		t.Fatal("empty geoip")
	}
}

func TestGeoSiteByName(t *testing.T) {
	list := &routercommon.GeoSiteList{
		Entry: []*routercommon.GeoSite{
			{
				CountryCode: "testx",
				Domain: []*routercommon.Domain{
					{Type: routercommon.Domain_Full, Value: "a.example"},
				},
			},
		},
	}
	d, ok := GeoSiteByName(list, "TESTX")
	if !ok || len(d) != 1 {
		t.Fatalf("lookup: ok=%v len=%d", ok, len(d))
	}
}

func TestGeoIPByCode(t *testing.T) {
	list := &routercommon.GeoIPList{
		Entry: []*routercommon.GeoIP{
			{
				CountryCode: "ir",
				Cidr: []*routercommon.CIDR{
					{Ip: []byte{8, 8, 8, 0}, Prefix: 24},
				},
			},
		},
	}
	c, ok := GeoIPByCode(list, "IR")
	if !ok || len(c) != 1 {
		t.Fatalf("lookup: ok=%v len=%d", ok, len(c))
	}
}

func TestFixtureRoundTrip(t *testing.T) {
	list := &routercommon.GeoSiteList{
		Entry: []*routercommon.GeoSite{
			{CountryCode: "ir", Domain: []*routercommon.Domain{{Type: routercommon.Domain_Full, Value: "x.ir"}}},
		},
	}
	b, err := proto.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	var out routercommon.GeoSiteList
	if err := proto.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.GetEntry()) != 1 {
		t.Fatal("round trip")
	}
}

func BenchmarkLoadEmbeddedGeoSite(b *testing.B) {
	for b.Loop() {
		_, err := LoadEmbeddedGeoSite()
		if err != nil {
			b.Fatal(err)
		}
	}
}

