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
	var sessionDL, sessionUL atomic.Uint64
	rc := gen.NewRatioController()
	opLim, largeLim, largeSem := asymmetricRuntime(cfg)

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
		onceRng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano())^uint64(time.Now().Unix()), uint64(os.Getpid())))
		code := doOne(runCtx, cfg, sched, &mc, perHost, log, &opSeq, onceRng, rc, opLim, largeLim, largeSem, &sessionDL, &sessionUL)
		if verbose {
			applog.MetricsSummary(log, &mc, verbose)
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
			rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano())^uint64(time.Now().Unix()), uint64(os.Getpid())))
			workerLoop(runCtx, cfg, sched, globalLim, perHost, &mc, log, &opSeq, rng, rc, opLim, largeLim, largeSem, &sessionDL, &sessionUL)
		}()
	}
	wg.Wait()
	if verbose {
		applog.MetricsSummary(log, &mc, verbose)
	}
	if runCtx.Err() != nil {
		applog.RunFooter(log, "stopped — signal received, workers exited")
	} else {
		applog.RunFooter(log, "run finished — all workers stopped")
	}
	return nil
}

func asymmetricRuntime(cfg *config.Config) (opLim *gen.OpLimiter, largeLim *rate.Limiter, largeSem chan struct{}) {
	a := cfg.Asymmetric
	if !a.Enabled {
		return nil, nil, nil
	}
	var rxLim, sendLim *rate.Limiter
	if a.ReceiveBytesPerSecond > 0 {
		burst := int(a.DownloadMaxBytes)
		if burst < 65536 {
			burst = 65536
		}
		rxLim = rate.NewLimiter(rate.Limit(a.ReceiveBytesPerSecond), burst)
	}
	if a.SendBytesPerSecond > 0 {
		burst := int(a.UploadMaxBytes)
		if burst < 65536 {
			burst = 65536
		}
		sendLim = rate.NewLimiter(rate.Limit(a.SendBytesPerSecond), burst)
	}
	if rxLim != nil || sendLim != nil {
		opLim = &gen.OpLimiter{Receive: rxLim, Send: sendLim}
	}
	if a.GlobalQPSLarge > 0 {
		largeLim = rate.NewLimiter(rate.Limit(a.GlobalQPSLarge), int(a.GlobalQPSLarge)+1)
	}
	if a.OperationWeights.Get > 0 && a.MaxConcurrentLargeDownloads > 0 {
		largeSem = make(chan struct{}, a.MaxConcurrentLargeDownloads)
	}
	return opLim, largeLim, largeSem
}

func workerLoop(ctx context.Context, cfg *config.Config, sched *gen.IranScheduler, globalLim *rate.Limiter, perHost *gen.PerHostLimiters, mc *metrics.Counters, log *zap.Logger, opSeq *atomic.Uint64, rng *rand.Rand, rc *gen.RatioController, opLim *gen.OpLimiter, largeLim *rate.Limiter, largeSem chan struct{}, sessionDL, sessionUL *atomic.Uint64) {
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
		_ = doOne(ctx, cfg, sched, mc, perHost, log, opSeq, rng, rc, opLim, largeLim, largeSem, sessionDL, sessionUL)
	}
}

func bumpOpAttempt(mc *metrics.Counters, op gen.OpKind) {
	mc.Attempts.Add(1)
	switch op {
	case gen.OpHead:
		mc.HeadAttempts.Add(1)
	case gen.OpGet:
		mc.GetAttempts.Add(1)
	case gen.OpPost:
		mc.PostAttempts.Add(1)
	}
}

func bumpOpSuccess(mc *metrics.Counters, op gen.OpKind) {
	mc.Successes.Add(1)
	switch op {
	case gen.OpHead:
		mc.HeadSuccesses.Add(1)
	case gen.OpGet:
		mc.GetSuccesses.Add(1)
	case gen.OpPost:
		mc.PostSuccesses.Add(1)
	}
}

func doOne(ctx context.Context, cfg *config.Config, sched *gen.IranScheduler, mc *metrics.Counters, perHost *gen.PerHostLimiters, log *zap.Logger, opSeq *atomic.Uint64, rng *rand.Rand, rc *gen.RatioController, opLim *gen.OpLimiter, largeLim *rate.Limiter, largeSem chan struct{}, sessionDL, sessionUL *atomic.Uint64) int {
	tgt, err := sched.NextTarget()
	if err != nil {
		log.Sugar().Errorf("scheduler_next_target error=%v", err)
		return 2
	}
	op := gen.PickOpKind(cfg, rng, rc, mc.BytesReceived.Load(), mc.BytesSent.Load())
	if cfg.Asymmetric.Enabled {
		a := cfg.Asymmetric
		if op == gen.OpGet && a.TotalDownloadCapBytes > 0 && sessionDL.Load() >= a.TotalDownloadCapBytes {
			op = gen.OpHead
		}
		if op == gen.OpPost && a.TotalUploadCapBytes > 0 && sessionUL.Load() >= a.TotalUploadCapBytes {
			op = gen.OpHead
		}
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

	host := tgt.Host
	var ip net.IP
	if tgt.IP != nil {
		ip = tgt.IP
	}
	opName := op.String()
	logURL := gen.HTTPSURLForLog(host, ip, op, cfg)

	if cfg.DryRun {
		erx, etx := gen.DryEstimateBytes(cfg, op)
		mc.BytesReceived.Add(uint64(erx))
		mc.BytesSent.Add(uint64(etx))
		applog.DryOp(log, seq, tgt, opName)
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

	if op == gen.OpGet && largeLim != nil {
		if err := largeLim.Wait(ctx); err != nil {
			return 0
		}
	}
	if op == gen.OpGet && largeSem != nil {
		select {
		case largeSem <- struct{}{}:
		case <-ctx.Done():
			return 0
		}
		defer func() { <-largeSem }()
	}

	bumpOpAttempt(mc, op)
	res, hErr := gen.HTTPSOperation(ctx, cfg, host, ip, op, opLim)
	applog.LiveHTTPS(log, opName, logURL, hErr)
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
	bumpOpSuccess(mc, op)
	if res.Rx > 0 {
		mc.BytesReceived.Add(uint64(res.Rx))
	}
	if res.Tx > 0 {
		mc.BytesSent.Add(uint64(res.Tx))
	}
	if cfg.Asymmetric.Enabled {
		if res.Rx > 0 {
			sessionDL.Add(uint64(res.Rx))
		}
		if res.Tx > 0 {
			sessionUL.Add(uint64(res.Tx))
		}
	}
	return 0
}
