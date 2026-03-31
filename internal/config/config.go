package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"gopkg.in/yaml.v3"
)

type Limits struct {
	GlobalQPS     float64 `yaml:"global_qps"`
	MaxInFlight   int     `yaml:"max_in_flight"`
	PerHostQPS    float64 `yaml:"per_host_qps"`
	HTTPSTimeout  int     `yaml:"https_timeout_seconds"`
	JitterMinMS   int     `yaml:"jitter_min_ms"`
	JitterMaxMS   int     `yaml:"jitter_max_ms"`
}

type Weights struct {
	GeoIP   float64            `yaml:"geoip_ir"`
	Geosite map[string]float64 `yaml:"geosite"`
}

type Safety struct {
	DenyPrivateIPs        bool     `yaml:"deny_private_ips"`
	AllowedDomainSuffixes []string `yaml:"allowed_domain_suffixes"`
}

type Config struct {
	DryRun         bool       `yaml:"dry_run"`
	Limits         Limits     `yaml:"limits"`
	GeositeLists   []string   `yaml:"geosite_lists"`
	Weights        Weights    `yaml:"weights"`
	Safety         Safety     `yaml:"safety"`
	Asymmetric     Asymmetric `yaml:"asymmetric"`
	WWWRootDomain  bool       `yaml:"www_root_domain"`
	InsecureTLS    bool       `yaml:"insecure_tls"`
	DNSEnabled     bool       `yaml:"dns_enabled"`
	SNIForIP       string     `yaml:"sni_for_ip"`
	PerHostMapMax  int        `yaml:"per_host_limiter_map_max"`
}

func Default() *Config {
	return &Config{
		DryRun: true,
		Limits: Limits{
			GlobalQPS:     5,
			MaxInFlight:   20,
			PerHostQPS:    0.2,
			HTTPSTimeout:  15,
			JitterMinMS:   50,
			JitterMaxMS:   500,
		},
		GeositeLists: nil,
		Weights: Weights{
			GeoIP:   2,
			Geosite: nil,
		},
		Safety: Safety{
			DenyPrivateIPs: true,
		},
		WWWRootDomain: false,
		InsecureTLS:   true,
		DNSEnabled:    false,
		PerHostMapMax: 2048,
	}
}

func Load(path string) (*Config, error) {
	c := Default()
	if strings.TrimSpace(path) == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

func Validate(c *Config, site *routercommon.GeoSiteList, ip *routercommon.GeoIPList) error {
	cidrs, ok := geo.GeoIPByCode(ip, geo.GeoIPCodeIR)
	if !ok || len(cidrs) == 0 {
		return fmt.Errorf("embedded geoip missing or empty list %q", geo.GeoIPCodeIR)
	}
	if c.GeositeLists == nil {
		c.GeositeLists = geo.IranCategoryGeositeCodes(site)
	}
	if len(c.GeositeLists) == 0 && c.Weights.GeoIP <= 0 {
		return fmt.Errorf("need geosite_lists or positive geoip_ir weight (omit geosite_lists for all category-*-ir, or use geosite_lists: [] for geoip-only)")
	}
	if c.Weights.Geosite == nil {
		c.Weights.Geosite = make(map[string]float64)
	}
	sum := c.Weights.GeoIP
	for _, name := range c.GeositeLists {
		doms, found := geo.GeoSiteByName(site, name)
		if !found {
			return fmt.Errorf("geosite list %q not found in embedded geosite.dat", name)
		}
		if len(doms) == 0 {
			return fmt.Errorf("geosite list %q is empty", name)
		}
		w := c.Weights.Geosite[name]
		if w <= 0 {
			w = geo.DefaultGeositeListWeight
			c.Weights.Geosite[name] = w
		}
		sum += w
	}
	if sum <= 0 {
		return fmt.Errorf("sum of weights must be positive")
	}
	if c.Limits.GlobalQPS <= 0 {
		return fmt.Errorf("limits.global_qps must be positive")
	}
	if c.Limits.MaxInFlight < 1 {
		return fmt.Errorf("limits.max_in_flight must be at least 1")
	}
	if c.Limits.PerHostQPS < 0 {
		return fmt.Errorf("limits.per_host_qps must be non-negative")
	}
	if c.Limits.HTTPSTimeout < 1 {
		return fmt.Errorf("limits.https_timeout_seconds must be at least 1")
	}
	if c.Limits.JitterMaxMS < c.Limits.JitterMinMS {
		return fmt.Errorf("jitter_max_ms must be >= jitter_min_ms")
	}
	if c.PerHostMapMax < 1 {
		c.PerHostMapMax = 2048
	}
	if err := validateAsymmetric(c); err != nil {
		return err
	}
	return nil
}

func validateAsymmetric(c *Config) error {
	a := &c.Asymmetric
	if !a.Enabled {
		return nil
	}
	if a.DownloadMaxBytes < 1 {
		return fmt.Errorf("asymmetric.download_max_bytes must be at least 1 when asymmetric.enabled")
	}
	if a.UploadMaxBytes < 1 {
		return fmt.Errorf("asymmetric.upload_max_bytes must be at least 1 when asymmetric.enabled")
	}
	w := a.OperationWeights
	sum := w.Head
	if w.Get < 0 || w.Head < 0 || w.Post < 0 {
		return fmt.Errorf("asymmetric.operation_weights must be non-negative")
	}
	sum += w.Get + w.Post
	if sum <= 0 {
		return fmt.Errorf("asymmetric.operation_weights must have positive sum")
	}
	if a.TargetRxTxRatio < 0 {
		return fmt.Errorf("asymmetric.target_rx_tx_ratio must be non-negative")
	}
	if a.GlobalQPSLarge < 0 {
		return fmt.Errorf("asymmetric.global_qps_large must be non-negative")
	}
	if a.ReceiveBytesPerSecond < 0 || a.SendBytesPerSecond < 0 {
		return fmt.Errorf("asymmetric receive/send bytes_per_second must be non-negative")
	}
	if a.MaxRedirects < 0 {
		return fmt.Errorf("asymmetric.max_redirects must be non-negative")
	}
	if a.TransportMaxIdleConnsPerHost < 0 || a.TransportIdleConnTimeoutSec < 0 {
		return fmt.Errorf("asymmetric transport limits must be non-negative")
	}
	if a.HeadEstimateRxBytes < 0 || a.HeadEstimateTxBytes < 0 {
		return fmt.Errorf("asymmetric head estimate bytes must be non-negative")
	}
	if a.RatioAdjustIntervalSec < 0 {
		return fmt.Errorf("asymmetric.ratio_adjust_interval_seconds must be non-negative")
	}
	normalizePath := func(p *string) {
		s := strings.TrimSpace(*p)
		if s == "" {
			s = "/"
		}
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		*p = s
	}
	normalizePath(&a.GetPath)
	normalizePath(&a.PostPath)
	if a.MaxRedirects == 0 {
		a.MaxRedirects = 3
	}
	if a.TransportMaxIdleConnsPerHost == 0 {
		a.TransportMaxIdleConnsPerHost = 32
	}
	if a.TransportIdleConnTimeoutSec == 0 {
		a.TransportIdleConnTimeoutSec = 90
	}
	if w.Get > 0 && a.MaxConcurrentLargeDownloads < 1 {
		a.MaxConcurrentLargeDownloads = 2
	}
	if a.HeadEstimateRxBytes == 0 {
		a.HeadEstimateRxBytes = 4096
	}
	if a.HeadEstimateTxBytes == 0 {
		a.HeadEstimateTxBytes = 4096
	}
	if a.RatioAdjustIntervalSec == 0 && a.TargetRxTxRatio > 0 {
		a.RatioAdjustIntervalSec = 30
	}
	return nil
}
