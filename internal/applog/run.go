package applog

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/arastu/irtrafficgen/internal/config"
	"github.com/arastu/irtrafficgen/internal/gen"
	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/arastu/irtrafficgen/internal/metrics"
	"go.uber.org/zap"
)

func RunStartup(_ *zap.Logger, cfg *config.Config, configPath string, once, verbose bool) {
	mode := "live"
	if cfg.DryRun {
		mode = "dry_run"
	}
	cfgLabel := "defaults"
	if strings.TrimSpace(configPath) != "" {
		cfgLabel = configPath
	}

	var geositeSummary string
	nList := len(cfg.GeositeLists)
	if nList == 0 {
		geositeSummary = "none (geoip:ir only)"
	} else {
		geositeSummary = fmt.Sprintf("%d lists", nList)
	}

	runMode := "continuous"
	if once {
		runMode = "once"
	}

	fmt.Fprintln(os.Stderr, "Run overview")
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "setting\tvalue")
	fmt.Fprintf(w, "mode\t%s\n", mode)
	fmt.Fprintf(w, "config\t%s\n", cfgLabel)
	fmt.Fprintf(w, "workers\t%d\n", cfg.Limits.MaxInFlight)
	fmt.Fprintf(w, "global_qps_cap\t%g\n", cfg.Limits.GlobalQPS)
	if cfg.Limits.PerHostQPS > 0 {
		fmt.Fprintf(w, "per_host_qps_cap\t%g\n", cfg.Limits.PerHostQPS)
	} else {
		fmt.Fprintf(w, "per_host_qps_cap\t(off)\n")
	}
	fmt.Fprintf(w, "geosite_lists\t%s\n", geositeSummary)
	fmt.Fprintf(w, "weight_geoip_ir\t%g\n", cfg.Weights.GeoIP)
	fmt.Fprintf(w, "dns_after_sample\t%v\n", cfg.DNSEnabled)
	fmt.Fprintf(w, "run_mode\t%s\n", runMode)
	fmt.Fprintf(w, "verbose\t%v\n", verbose)
	_ = w.Flush()

	if nList == 0 {
		return
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Geosite list weights")
	w = tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "list\tweight")
	for _, name := range cfg.GeositeLists {
		wei := cfg.Weights.Geosite[name]
		if wei <= 0 {
			wei = geo.DefaultGeositeListWeight
		}
		fmt.Fprintf(w, "%s\t%g\n", name, wei)
	}
	_ = w.Flush()
	fmt.Fprintln(os.Stderr)
}

func DryOp(log *zap.Logger, seq uint64, tgt gen.Target) {
	s := log.Sugar()
	if tgt.Kind == gen.TargetGeoIP {
		s.Infof("dry_run seq=%d kind=geoip list=%s ip=%s would=https:443", seq, tgt.ListName, tgt.IP.String())
		return
	}
	s.Infof("dry_run seq=%d kind=geosite list=%s host=%s would=https_head", seq, tgt.ListName, tgt.Host)
}

func LiveTarget(log *zap.Logger, seq uint64, tgt gen.Target) {
	s := log.Sugar()
	if tgt.Kind == gen.TargetGeoIP {
		s.Infof("target_selected seq=%d kind=geoip list=%s ip=%s", seq, tgt.ListName, tgt.IP.String())
		return
	}
	s.Infof("target_selected seq=%d kind=geosite list=%s host=%s", seq, tgt.ListName, tgt.Host)
}

func LiveDNS(log *zap.Logger, host string, err error) {
	s := log.Sugar()
	if err != nil {
		s.Warnf("dns_lookup host=%s error=%v", host, err)
		return
	}
	s.Infof("dns_lookup host=%s ok", host)
}

func LiveHTTPSHost(log *zap.Logger, host string, err error) {
	s := log.Sugar()
	url := "https://" + host + "/"
	if err != nil {
		s.Warnf("https_head url=%s error=%v", url, err)
		return
	}
	s.Infof("https_head url=%s ok", url)
}

func LiveHTTPSIP(log *zap.Logger, ip string, err error) {
	s := log.Sugar()
	url := "https://" + ip + "/"
	if err != nil {
		s.Warnf("https_head url=%s error=%v", url, err)
		return
	}
	s.Infof("https_head url=%s ok", url)
}

func MetricsSummary(_ *zap.Logger, mc *metrics.Counters) {
	fmt.Fprintln(os.Stderr, "Session metrics")
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "metric\tcount")
	fmt.Fprintf(w, "https_attempts\t%d\n", mc.Attempts.Load())
	fmt.Fprintf(w, "https_success\t%d\n", mc.Successes.Load())
	fmt.Fprintf(w, "err_timeout\t%d\n", mc.ErrTimeout.Load())
	fmt.Fprintf(w, "err_tls\t%d\n", mc.ErrTLS.Load())
	fmt.Fprintf(w, "err_http\t%d\n", mc.ErrHTTP.Load())
	fmt.Fprintf(w, "err_other\t%d\n", mc.ErrOther.Load())
	fmt.Fprintf(w, "dns_success\t%d\n", mc.DNSSuccesses.Load())
	fmt.Fprintf(w, "dns_errors\t%d\n", mc.DNSErrors.Load())
	_ = w.Flush()
	fmt.Fprintln(os.Stderr)
}

func RunFooter(log *zap.Logger, msg string) {
	log.Sugar().Infof("run_complete %s", msg)
}

func Version(log *zap.Logger, ver, commit string) {
	log.Sugar().Infof("build version=%s commit=%s", ver, commit)
}
