package applog

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"go.uber.org/zap"
)

func InspectPhaseLoad(log *zap.Logger, resource string) {
	log.Sugar().Infof("inspect_load resource=%s phase=started", resource)
}

func InspectPhaseReady(log *zap.Logger, resource, detail string) {
	log.Sugar().Infof("inspect_load resource=%s phase=ready detail=%s", resource, detail)
}

func GeositeLists(_ *zap.Logger, site *routercommon.GeoSiteList) {
	fmt.Fprintln(os.Stderr, "Embedded geosite lists")
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "list\tdomain_rules")
	for _, code := range geo.GeoSiteEntryCodes(site) {
		doms, _ := geo.GeoSiteByName(site, code)
		fmt.Fprintf(w, "%s\t%d\n", code, len(doms))
	}
	_ = w.Flush()
	fmt.Fprintln(os.Stderr)
}

func GeoIPIran(log *zap.Logger, ipList *routercommon.GeoIPList) int {
	cidrs, ok := geo.GeoIPByCode(ipList, geo.GeoIPCodeIR)
	if !ok {
		log.Sugar().Errorf("geoip_missing code=%s", geo.GeoIPCodeIR)
		return 1
	}
	v4, v6 := geo.CIDRStats(cidrs)
	log.Sugar().Infof("geoip_iran code=%s cidr_entries=%d ipv4_rule_blocks=%d ipv6_rule_blocks=%d",
		geo.GeoIPCodeIR, len(cidrs), v4, v6)
	return 0
}

func InspectDone(log *zap.Logger) {
	log.Sugar().Infof("inspect_complete geoip_ir=present")
}
