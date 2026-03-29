package geo

import (
	"testing"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func TestIsIranCategoryGeositeCode(t *testing.T) {
	if !IsIranCategoryGeositeCode("CATEGORY-NEWS-IR") {
		t.Fatal()
	}
	if IsIranCategoryGeositeCode("category-news") {
		t.Fatal()
	}
	if IsIranCategoryGeositeCode("google") {
		t.Fatal()
	}
	if !IsIranCategoryGeositeCode("category-ir") {
		t.Fatal()
	}
}

func TestIranCategoryGeositeCodes(t *testing.T) {
	site := &routercommon.GeoSiteList{
		Entry: []*routercommon.GeoSite{
			{CountryCode: "category-b-ir", Domain: []*routercommon.Domain{{Type: routercommon.Domain_Full, Value: "b.ir"}}},
			{CountryCode: "category-a-ir", Domain: []*routercommon.Domain{{Type: routercommon.Domain_Full, Value: "a.ir"}}},
			{CountryCode: "google", Domain: []*routercommon.Domain{{Type: routercommon.Domain_Full, Value: "g.com"}}},
		},
	}
	got := IranCategoryGeositeCodes(site)
	if len(got) != 2 || got[0] != "category-a-ir" || got[1] != "category-b-ir" {
		t.Fatalf("%q", got)
	}
}
