package gen

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/arastu/irtrafficgen/internal/config"
	"github.com/arastu/irtrafficgen/internal/geo"
	"github.com/arastu/irtrafficgen/internal/target"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

type TargetKind int

const (
	TargetGeosite TargetKind = iota
	TargetGeoIP
)

type Target struct {
	Kind     TargetKind
	ListName string
	Host     string
	IP       net.IP
}

type IranScheduler struct {
	mu          sync.Mutex
	rng         *rand.Rand
	geoWeight   float64
	listNames   []string
	listWeights []float64
	total       float64
	domains     map[string][]*routercommon.Domain
	cidrs       []*routercommon.CIDR
	cfg         *config.Config
}

func NewIranScheduler(cfg *config.Config, site *routercommon.GeoSiteList, ipList *routercommon.GeoIPList, rng *rand.Rand) (*IranScheduler, error) {
	cidrs, ok := geo.GeoIPByCode(ipList, geo.GeoIPCodeIR)
	if !ok || len(cidrs) == 0 {
		return nil, fmt.Errorf("geoip:%s missing", geo.GeoIPCodeIR)
	}
	domains := make(map[string][]*routercommon.Domain)
	var names []string
	var weights []float64
	sum := cfg.Weights.GeoIP
	if sum < 0 {
		sum = 0
	}
	for _, name := range cfg.GeositeLists {
		dlist, found := geo.GeoSiteByName(site, name)
		if !found || len(dlist) == 0 {
			return nil, fmt.Errorf("geosite %q", name)
		}
		domains[name] = dlist
		names = append(names, name)
		w := cfg.Weights.Geosite[name]
		if w <= 0 {
			w = geo.DefaultGeositeListWeight
		}
		weights = append(weights, w)
		sum += w
	}
	if sum <= 0 {
		return nil, fmt.Errorf("zero total weight")
	}
	return &IranScheduler{
		rng:         rng,
		geoWeight:   cfg.Weights.GeoIP,
		listNames:   names,
		listWeights: weights,
		total:       sum,
		domains:     domains,
		cidrs:       cidrs,
		cfg:         cfg,
	}, nil
}

func (s *IranScheduler) NextTarget() (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.total <= 0 {
		return Target{}, fmt.Errorf("scheduler closed")
	}
	r := s.rng.Float64() * s.total
	if r < s.geoWeight {
		ip, err := geo.SamplePublicIPFromCIDRs(s.cidrs, s.cfg.Safety.DenyPrivateIPs, s.rng, 64)
		if err != nil {
			return Target{}, err
		}
		return Target{Kind: TargetGeoIP, ListName: geo.GeoIPCodeIR, IP: ip}, nil
	}
	r -= s.geoWeight
	for i, w := range s.listWeights {
		r -= w
		if r < 0 {
			return s.sampleGeosite(s.listNames[i])
		}
	}
	return s.sampleGeosite(s.listNames[len(s.listNames)-1])
}

func (s *IranScheduler) sampleGeosite(listName string) (Target, error) {
	dlist := s.domains[listName]
	const maxTry = 80
	for range maxTry {
		d := dlist[s.rng.IntN(len(dlist))]
		host, err := target.HostForDomain(d, s.cfg.WWWRootDomain)
		if err != nil {
			continue
		}
		if !target.HostAllowed(host, s.cfg.Safety.AllowedDomainSuffixes) {
			continue
		}
		return Target{Kind: TargetGeosite, ListName: listName, Host: host}, nil
	}
	return Target{}, fmt.Errorf("geosite sample failed")
}

func JitterSleep(ctx context.Context, rng *rand.Rand, minMS, maxMS int) error {
	if maxMS < minMS {
		minMS, maxMS = maxMS, minMS
	}
	if maxMS <= 0 {
		return nil
	}
	d := time.Duration(rng.IntN(maxMS-minMS+1)+minMS) * time.Millisecond
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
