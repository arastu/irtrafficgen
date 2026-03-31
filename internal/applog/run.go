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
	fmt.Fprintf(w, "asymmetric_enabled\t%v\n", cfg.Asymmetric.Enabled)
	if cfg.Asymmetric.Enabled {
		aw := cfg.Asymmetric.OperationWeights
		fmt.Fprintf(w, "asymmetric_weights\thead=%g get=%g post=%g\n", aw.Head, aw.Get, aw.Post)
		fmt.Fprintf(w, "asymmetric_download_max_bytes\t%d\n", cfg.Asymmetric.DownloadMaxBytes)
		fmt.Fprintf(w, "asymmetric_upload_max_bytes\t%d\n", cfg.Asymmetric.UploadMaxBytes)
		if cfg.Asymmetric.TargetRxTxRatio > 0 {
			fmt.Fprintf(w, "asymmetric_target_rx_tx_ratio\t%g\n", cfg.Asymmetric.TargetRxTxRatio)
		}
		if cfg.Asymmetric.GlobalQPSLarge > 0 {
			fmt.Fprintf(w, "asymmetric_global_qps_large\t%g\n", cfg.Asymmetric.GlobalQPSLarge)
		}
		if cfg.Asymmetric.ReceiveBytesPerSecond > 0 {
			fmt.Fprintf(w, "asymmetric_receive_bps\t%g\n", cfg.Asymmetric.ReceiveBytesPerSecond)
		}
		if cfg.Asymmetric.SendBytesPerSecond > 0 {
			fmt.Fprintf(w, "asymmetric_send_bps\t%g\n", cfg.Asymmetric.SendBytesPerSecond)
		}
	}
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

func DryOp(log *zap.Logger, seq uint64, tgt gen.Target, op string) {
	s := log.Sugar()
	if tgt.Kind == gen.TargetGeoIP {
		s.Infof("dry_run seq=%d kind=geoip list=%s ip=%s op=%s would=https:443", seq, tgt.ListName, tgt.IP.String(), op)
		return
	}
	s.Infof("dry_run seq=%d kind=geosite list=%s host=%s op=%s would=https", seq, tgt.ListName, tgt.Host, op)
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

func LiveHTTPS(log *zap.Logger, op string, url string, err error) {
	s := log.Sugar()
	if err != nil {
		s.Warnf("https op=%s url=%s error=%v", op, url, err)
		return
	}
	s.Infof("https op=%s url=%s ok", op, url)
}

func MetricsSummary(_ *zap.Logger, mc *metrics.Counters, verbose bool) {
	fmt.Fprintln(os.Stderr, "Session metrics")
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "metric\tcount")
	fmt.Fprintf(w, "https_attempts\t%d\n", mc.Attempts.Load())
	fmt.Fprintf(w, "https_success\t%d\n", mc.Successes.Load())
	fmt.Fprintf(w, "bytes_sent\t%d\n", mc.BytesSent.Load())
	fmt.Fprintf(w, "bytes_received\t%d\n", mc.BytesReceived.Load())
	fmt.Fprintf(w, "head_attempts\t%d\n", mc.HeadAttempts.Load())
	fmt.Fprintf(w, "head_success\t%d\n", mc.HeadSuccesses.Load())
	fmt.Fprintf(w, "get_attempts\t%d\n", mc.GetAttempts.Load())
	fmt.Fprintf(w, "get_success\t%d\n", mc.GetSuccesses.Load())
	fmt.Fprintf(w, "post_attempts\t%d\n", mc.PostAttempts.Load())
	fmt.Fprintf(w, "post_success\t%d\n", mc.PostSuccesses.Load())
	fmt.Fprintf(w, "err_timeout\t%d\n", mc.ErrTimeout.Load())
	fmt.Fprintf(w, "err_tls\t%d\n", mc.ErrTLS.Load())
	fmt.Fprintf(w, "err_http\t%d\n", mc.ErrHTTP.Load())
	fmt.Fprintf(w, "err_other\t%d\n", mc.ErrOther.Load())
	fmt.Fprintf(w, "dns_success\t%d\n", mc.DNSSuccesses.Load())
	fmt.Fprintf(w, "dns_errors\t%d\n", mc.DNSErrors.Load())
	_ = w.Flush()
	if verbose {
		tx := float64(mc.BytesSent.Load())
		rx := float64(mc.BytesReceived.Load())
		if tx > 0 {
			fmt.Fprintf(os.Stderr, "rx_tx_ratio\t%g\n", rx/tx)
		} else if rx > 0 {
			fmt.Fprintln(os.Stderr, "rx_tx_ratio\tinf (tx=0)")
		}
	}
	fmt.Fprintln(os.Stderr)
}

func RunFooter(log *zap.Logger, msg string) {
	log.Sugar().Infof("run_complete %s", msg)
}

func Version(log *zap.Logger, ver, commit string) {
	log.Sugar().Infof("build version=%s commit=%s", ver, commit)
}
