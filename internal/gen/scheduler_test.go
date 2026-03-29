package gen

import (
	"math/rand/v2"
	"testing"

	"github.com/arastu/irtrafficgen/internal/config"
	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func TestIranSchedulerDistribution(t *testing.T) {
	cfg := config.Default()
	cfg.Weights.GeoIP = 1
	cfg.Weights.Geosite = map[string]float64{"test-ir": 1}
	site := &routercommon.GeoSiteList{
		Entry: []*routercommon.GeoSite{
			{CountryCode: "test-ir", Domain: []*routercommon.Domain{{Type: routercommon.Domain_Full, Value: "example.ir"}}},
		},
	}
	cfg.GeositeLists = []string{"test-ir"}
	ipList := &routercommon.GeoIPList{
		Entry: []*routercommon.GeoIP{
			{CountryCode: geo.GeoIPCodeIR, Cidr: []*routercommon.CIDR{{Ip: []byte{8, 8, 8, 0}, Prefix: 24}}},
		},
	}
	rng := rand.New(rand.NewPCG(42, 43))
	s, err := NewIranScheduler(cfg, site, ipList, rng)
	if err != nil {
		t.Fatal(err)
	}
	var geoIP, geoSite int
	for range 2000 {
		tgt, err := s.NextTarget()
		if err != nil {
			t.Fatal(err)
		}
		switch tgt.Kind {
		case TargetGeoIP:
			geoIP++
		case TargetGeosite:
			geoSite++
		}
	}
	if geoIP < 700 || geoIP > 1300 {
		t.Fatalf("geoip count %d", geoIP)
	}
	if geoSite < 700 || geoSite > 1300 {
		t.Fatalf("geosite count %d", geoSite)
	}
}
