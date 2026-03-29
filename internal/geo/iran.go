package geo

import (
	"sort"
	"strings"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

const GeoIPCodeIR = "ir"

const DefaultGeositeListWeight = 2.0

func IsIranCategoryGeositeCode(code string) bool {
	c := strings.ToLower(strings.TrimSpace(code))
	return strings.HasPrefix(c, "category-") && strings.HasSuffix(c, "-ir")
}

func IranCategoryGeositeCodes(site *routercommon.GeoSiteList) []string {
	seen := make(map[string]struct{})
	for _, e := range site.GetEntry() {
		code := e.GetCountryCode()
		if !IsIranCategoryGeositeCode(code) {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for code := range seen {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}
