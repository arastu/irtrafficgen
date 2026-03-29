package cmd

import (
	"context"
	"math/rand/v2"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arastu/irtrafficgen/internal/applog"
	"github.com/arastu/irtrafficgen/internal/config"
	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/arastu/irtrafficgen/internal/gen"
	"github.com/arastu/irtrafficgen/internal/metrics"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func newRunCommand() *cli.Command {
	return &cli.Command{
		Name:        "run",
		Usage:       "Generate traffic (dry-run unless -live or config)",
		Description: "Worker pool with rate limits. Default when no subcommand is given.",
		Action:      runAction,
	}
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	cfgPath := cmd.String("config")
	live := cmd.Bool("live")
	dryRunStr := cmd.String("dry-run")
	once := cmd.Bool("once")
	verbose := cmd.Bool("verbose")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return cli.Exit(err, 1)
	}
	switch strings.ToLower(strings.TrimSpace(dryRunStr)) {
	case "true", "1", "yes":
		cfg.DryRun = true
	case "false", "0", "no":
		cfg.DryRun = false
	case "":
	default:
		return cli.Exit("invalid --dry-run value, want true or false", 1)
	}
	if live {
		cfg.DryRun = false
	}

	site, err := geo.LoadEmbeddedGeoSite()
	if err != nil {
		return cli.Exit(err, 2)
	}
	ipList, err := geo.LoadEmbeddedGeoIP()
	if err != nil {
		return cli.Exit(err, 2)
	}
	if err := config.Validate(cfg, site, ipList); err != nil {
		return cli.Exit(err, 1)
	}

	sched, err := gen.NewIranScheduler(cfg, site, ipList, rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()>>32))))
	if err != nil {
		return cli.Exit(err, 1)
	}

	log, err := applog.New()
	if err != nil {
		return cli.Exit(err, 2)
	}
	defer func() { _ = log.Sync() }()

	applog.RunStartup(log, cfg, cfgPath, once, verbose)

	runCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var mc metrics.Counters
	var opSeq atomic.Uint64
	globalLim := rate.NewLimiter(rate.Limit(cfg.Limits.GlobalQPS), int(cfg.Limits.GlobalQPS)+1)
	var perHost *gen.PerHostLimiters
	if cfg.Limits.PerHostQPS > 0 {
		perHost = gen.NewPerHostLimiters(cfg.Limits.PerHostQPS, cfg.PerHostMapMax)
	}

	if once {
		if err := globalLim.Wait(runCtx); err != nil {
			applog.RunFooter(log, "stopped before first operation (signal)")
			return nil
		}
		code := doOne(runCtx, cfg, sched, &mc, perHost, log, &opSeq)
		if verbose {
			applog.MetricsSummary(log, &mc)
		}
		if code != 0 {
			applog.RunFooter(log, "run finished — one operation reported an error")
			return cli.Exit("HTTPS or scheduler error", code)
		}
		applog.RunFooter(log, "run finished — single operation complete")
		return nil
	}

	var wg sync.WaitGroup
	for range cfg.Limits.MaxInFlight {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerLoop(runCtx, cfg, sched, globalLim, perHost, &mc, log, &opSeq)
		}()
	}
	wg.Wait()
	if verbose {
		applog.MetricsSummary(log, &mc)
	}
	if runCtx.Err() != nil {
		applog.RunFooter(log, "stopped — signal received, workers exited")
	} else {
		applog.RunFooter(log, "run finished — all workers stopped")
	}
	return nil
}

func workerLoop(ctx context.Context, cfg *config.Config, sched *gen.IranScheduler, globalLim *rate.Limiter, perHost *gen.PerHostLimiters, mc *metrics.Counters, log *zap.Logger, opSeq *atomic.Uint64) {
	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano())^uint64(time.Now().Unix()), uint64(os.Getpid())))
	for {
		if err := globalLim.Wait(ctx); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := gen.JitterSleep(ctx, rng, cfg.Limits.JitterMinMS, cfg.Limits.JitterMaxMS); err != nil {
			return
		}
		_ = doOne(ctx, cfg, sched, mc, perHost, log, opSeq)
	}
}

func doOne(ctx context.Context, cfg *config.Config, sched *gen.IranScheduler, mc *metrics.Counters, perHost *gen.PerHostLimiters, log *zap.Logger, opSeq *atomic.Uint64) int {
	tgt, err := sched.NextTarget()
	if err != nil {
		log.Sugar().Errorf("scheduler_next_target error=%v", err)
		return 2
	}
	seq := opSeq.Add(1)
	key := tgt.Host
	if key == "" && tgt.IP != nil {
		key = tgt.IP.String()
	}
	if perHost != nil && key != "" {
		if err := perHost.Wait(ctx, key); err != nil {
			return 0
		}
	}

	if cfg.DryRun {
		applog.DryOp(log, seq, tgt)
		return 0
	}

	applog.LiveTarget(log, seq, tgt)

	if cfg.DNSEnabled && tgt.Host != "" {
		mc.DNSAttempts.Add(1)
		_, lerr := gen.LookupHost(ctx, net.DefaultResolver, tgt.Host)
		applog.LiveDNS(log, tgt.Host, lerr)
		if lerr != nil {
			mc.DNSErrors.Add(1)
		} else {
			mc.DNSSuccesses.Add(1)
		}
	}

	mc.Attempts.Add(1)
	to := time.Duration(cfg.Limits.HTTPSTimeout) * time.Second
	var hErr error
	if tgt.Host != "" {
		hErr = gen.HTTPSHead(ctx, tgt.Host, nil, "", to, cfg.InsecureTLS)
		applog.LiveHTTPSHost(log, tgt.Host, hErr)
	} else {
		hErr = gen.HTTPSHead(ctx, "", tgt.IP, cfg.SNIForIP, to, cfg.InsecureTLS)
		applog.LiveHTTPSIP(log, tgt.IP.String(), hErr)
	}
	if hErr != nil {
		cls := gen.ClassifyHTTPSError(hErr)
		switch cls {
		case "timeout":
			mc.ErrTimeout.Add(1)
		case "tls":
			mc.ErrTLS.Add(1)
		case "http":
			mc.ErrHTTP.Add(1)
		default:
			mc.ErrOther.Add(1)
		}
		return 2
	}
	mc.Successes.Add(1)
	return 0
}
