package cmd

import (
	"context"
	"fmt"

	"github.com/arastu/irtrafficgen/internal/applog"
	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/urfave/cli/v3"
)

func newInspectCommand() *cli.Command {
	return &cli.Command{
		Name:        "inspect",
		Usage:       "Print embedded geosite/geoip summary",
		Description: "Lists geosite list codes and Iran (geoip:ir) CIDR stats. Exits 1 if geoip:ir is missing.",
		Action:      inspectAction,
	}
}

func inspectAction(_ context.Context, _ *cli.Command) error {
	log, err := applog.New()
	if err != nil {
		return cli.Exit(err, 2)
	}
	defer func() { _ = log.Sync() }()

	applog.InspectPhaseLoad(log, "geosite.dat")
	site, err := geo.LoadEmbeddedGeoSite()
	if err != nil {
		return cli.Exit(err, 2)
	}
	applog.InspectPhaseReady(log, "geosite.dat", fmt.Sprintf("%d lists", len(site.GetEntry())))

	applog.InspectPhaseLoad(log, "geoip.dat")
	ipList, err := geo.LoadEmbeddedGeoIP()
	if err != nil {
		return cli.Exit(err, 2)
	}
	applog.InspectPhaseReady(log, "geoip.dat", fmt.Sprintf("%d country entries", len(ipList.GetEntry())))

	applog.GeositeLists(log, site)
	code := applog.GeoIPIran(log, ipList)
	if code != 0 {
		return cli.Exit("geoip:ir missing from embedded geoip.dat", 1)
	}
	applog.InspectDone(log)
	return nil
}
