package geo

import (
	"fmt"
	"strings"

	"github.com/arastu/irtrafficgen/internal/assets"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

func LoadEmbeddedGeoSite() (*routercommon.GeoSiteList, error) {
	var list routercommon.GeoSiteList
	if err := proto.Unmarshal(assets.GeoSiteDat, &list); err != nil {
		return nil, fmt.Errorf("geosite: %w", err)
	}
	return &list, nil
}

func LoadEmbeddedGeoIP() (*routercommon.GeoIPList, error) {
	var list routercommon.GeoIPList
	if err := proto.Unmarshal(assets.GeoIPDat, &list); err != nil {
		return nil, fmt.Errorf("geoip: %w", err)
	}
	return &list, nil
}

func GeoSiteByName(list *routercommon.GeoSiteList, name string) ([]*routercommon.Domain, bool) {
	for _, e := range list.GetEntry() {
		if strings.EqualFold(e.GetCountryCode(), name) {
			return e.GetDomain(), true
		}
	}
	return nil, false
}

func GeoIPByCode(list *routercommon.GeoIPList, code string) ([]*routercommon.CIDR, bool) {
	for _, e := range list.GetEntry() {
		if strings.EqualFold(e.GetCountryCode(), code) {
			return e.GetCidr(), true
		}
	}
	return nil, false
}

func GeoSiteEntryCodes(list *routercommon.GeoSiteList) []string {
	entries := list.GetEntry()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.GetCountryCode())
	}
	return out
}

func CIDRStats(cidrs []*routercommon.CIDR) (ipv4, ipv6 int) {
	for _, c := range cidrs {
		ipLen := len(c.GetIp())
		switch ipLen {
		case 4:
			ipv4++
		case 16:
			ipv6++
		}
	}
	return ipv4, ipv6
}
