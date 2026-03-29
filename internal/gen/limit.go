package gen

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type PerHostLimiters struct {
	mu      sync.Mutex
	m       map[string]*rate.Limiter
	qps     float64
	maxKeys int
}

func NewPerHostLimiters(qps float64, maxKeys int) *PerHostLimiters {
	if maxKeys < 1 {
		maxKeys = 2048
	}
	return &PerHostLimiters{
		m:       make(map[string]*rate.Limiter),
		qps:     qps,
		maxKeys: maxKeys,
	}
}

func (p *PerHostLimiters) Wait(ctx context.Context, key string) error {
	if p == nil || p.qps <= 0 {
		return nil
	}
	p.mu.Lock()
	if len(p.m) >= p.maxKeys {
		p.evictHalfLocked()
	}
	lim, ok := p.m[key]
	if !ok {
		burst := int(p.qps) + 1
		if burst < 1 {
			burst = 1
		}
		lim = rate.NewLimiter(rate.Limit(p.qps), burst)
		p.m[key] = lim
	}
	p.mu.Unlock()
	return lim.Wait(ctx)
}

func (p *PerHostLimiters) evictHalfLocked() {
	n := 0
	for k := range p.m {
		delete(p.m, k)
		n++
		if n >= p.maxKeys/2 {
			return
		}
	}
}
