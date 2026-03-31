package gen

import (
	"sync"
	"time"

	"github.com/arastu/irtrafficgen/internal/config"
)

type RatioController struct {
	mu       sync.Mutex
	last     time.Time
	mH, mG, mP float64
}

func NewRatioController() *RatioController {
	return &RatioController{mH: 1, mG: 1, mP: 1}
}

func clampMul(v *float64, lo, hi float64) {
	if *v < lo {
		*v = lo
	}
	if *v > hi {
		*v = hi
	}
}

func (r *RatioController) Weights(cfg *config.Config, rx64, tx64 uint64) (h, g, p float64) {
	if cfg == nil || !cfg.Asymmetric.Enabled {
		return 1, 0, 0
	}
	a := cfg.Asymmetric
	bh, bg, bp := a.OperationWeights.Head, a.OperationWeights.Get, a.OperationWeights.Post
	if a.TargetRxTxRatio <= 0 {
		return bh, bg, bp
	}
	interval := time.Duration(a.RatioAdjustIntervalSec) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	now := time.Now()
	r.mu.Lock()
	if r.last.IsZero() {
		r.last = now
	} else if now.Sub(r.last) >= interval {
		r.last = now
		var ratio float64
		tx := float64(tx64)
		rx := float64(rx64)
		if tx > 0 {
			ratio = rx / tx
		} else if rx > 0 {
			ratio = 1e9
		} else {
			ratio = a.TargetRxTxRatio
		}
		tgt := a.TargetRxTxRatio
		if tgt > 0 && ratio < tgt*0.9 {
			r.mG *= 1.08
			r.mH *= 0.96
		} else if tgt > 0 && ratio > tgt*1.1 && tx > 0 {
			r.mG *= 0.96
			r.mH *= 1.04
		}
		clampMul(&r.mH, 0.25, 4)
		clampMul(&r.mG, 0.25, 4)
		clampMul(&r.mP, 0.25, 4)
	}
	h = bh * r.mH
	g = bg * r.mG
	p = bp * r.mP
	r.mu.Unlock()
	if h+g+p <= 0 {
		return 1, 0, 0
	}
	return h, g, p
}
