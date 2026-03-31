package config

import (
	"testing"

	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func siteIR() *routercommon.GeoSiteList {
	return &routercommon.GeoSiteList{
		Entry: []*routercommon.GeoSite{
			{CountryCode: "ir", Domain: []*routercommon.Domain{{Type: routercommon.Domain_Full, Value: "x.ir"}}},
		},
	}
}

func ipIR() *routercommon.GeoIPList {
	return &routercommon.GeoIPList{
		Entry: []*routercommon.GeoIP{
			{CountryCode: geo.GeoIPCodeIR, Cidr: []*routercommon.CIDR{{Ip: []byte{8, 8, 8, 0}, Prefix: 24}}},
		},
	}
}

func TestValidateOK(t *testing.T) {
	c := Default()
	if err := Validate(c, siteIR(), ipIR()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateExplicitEmptyGeositeGeoIPOnly(t *testing.T) {
	c := Default()
	c.GeositeLists = []string{}
	c.Weights.GeoIP = 3
	if err := Validate(c, siteIR(), ipIR()); err != nil {
		t.Fatal(err)
	}
	if len(c.GeositeLists) != 0 {
		t.Fatalf("expected no geosite expansion, got %v", c.GeositeLists)
	}
}

func TestDefaultValidateEmbedded(t *testing.T) {
	site, err := geo.LoadEmbeddedGeoSite()
	if err != nil {
		t.Fatal(err)
	}
	ipList, err := geo.LoadEmbeddedGeoIP()
	if err != nil {
		t.Fatal(err)
	}
	c := Default()
	if err := Validate(c, site, ipList); err != nil {
		t.Fatal(err)
	}
}

func TestValidateMissingGeositeList(t *testing.T) {
	c := Default()
	c.GeositeLists = []string{"does-not-exist"}
	if err := Validate(c, siteIR(), ipIR()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateMissingGeoIP(t *testing.T) {
	c := Default()
	empty := &routercommon.GeoIPList{}
	if err := Validate(c, siteIR(), empty); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNoSources(t *testing.T) {
	c := Default()
	c.GeositeLists = nil
	c.Weights.GeoIP = 0
	if err := Validate(c, siteIR(), ipIR()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateDiscoverCategoriesEmbedded(t *testing.T) {
	site, err := geo.LoadEmbeddedGeoSite()
	if err != nil {
		t.Fatal(err)
	}
	ipList, err := geo.LoadEmbeddedGeoIP()
	if err != nil {
		t.Fatal(err)
	}
	c := Default()
	if err := Validate(c, site, ipList); err != nil {
		t.Fatal(err)
	}
	if len(c.GeositeLists) == 0 {
		t.Fatal("expected discovered category-*-ir lists")
	}
	for _, name := range c.GeositeLists {
		if !geo.IsIranCategoryGeositeCode(name) {
			t.Fatalf("unexpected list %q", name)
		}
	}
}

func TestValidateBadGlobalQPS(t *testing.T) {
	c := Default()
	c.Limits.GlobalQPS = 0
	if err := Validate(c, siteIR(), ipIR()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateJitter(t *testing.T) {
	c := Default()
	c.Limits.JitterMinMS = 100
	c.Limits.JitterMaxMS = 50
	if err := Validate(c, siteIR(), ipIR()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAsymmetricDisabledNoExtra(t *testing.T) {
	c := Default()
	if err := Validate(c, siteIR(), ipIR()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAsymmetricEnabledOK(t *testing.T) {
	c := Default()
	c.Asymmetric.Enabled = true
	c.Asymmetric.DownloadMaxBytes = 1024
	c.Asymmetric.UploadMaxBytes = 512
	c.Asymmetric.OperationWeights = OpWeights{Head: 1, Get: 2, Post: 0}
	c.Asymmetric.MaxConcurrentLargeDownloads = 2
	if err := Validate(c, siteIR(), ipIR()); err != nil {
		t.Fatal(err)
	}
	if c.Asymmetric.GetPath != "/" {
		t.Fatalf("get path %q", c.Asymmetric.GetPath)
	}
}

func TestValidateAsymmetricZeroWeights(t *testing.T) {
	c := Default()
	c.Asymmetric.Enabled = true
	c.Asymmetric.DownloadMaxBytes = 1
	c.Asymmetric.UploadMaxBytes = 1
	c.Asymmetric.OperationWeights = OpWeights{}
	if err := Validate(c, siteIR(), ipIR()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAsymmetricGetDefaultsConcurrency(t *testing.T) {
	c := Default()
	c.Asymmetric.Enabled = true
	c.Asymmetric.DownloadMaxBytes = 1
	c.Asymmetric.UploadMaxBytes = 1
	c.Asymmetric.OperationWeights = OpWeights{Get: 1}
	c.Asymmetric.MaxConcurrentLargeDownloads = 0
	if err := Validate(c, siteIR(), ipIR()); err != nil {
		t.Fatal(err)
	}
	if c.Asymmetric.MaxConcurrentLargeDownloads != 2 {
		t.Fatalf("got %d", c.Asymmetric.MaxConcurrentLargeDownloads)
	}
}
