package gen

import (
	"math/rand/v2"

	"github.com/arastu/irtrafficgen/internal/config"
)

type OpKind int

const (
	OpHead OpKind = iota
	OpGet
	OpPost
)

func (k OpKind) String() string {
	switch k {
	case OpHead:
		return "head"
	case OpGet:
		return "get"
	case OpPost:
		return "post"
	default:
		return "unknown"
	}
}

func PickOpKind(cfg *config.Config, rng *rand.Rand, rc *RatioController, rx, tx uint64) OpKind {
	if cfg == nil || !cfg.Asymmetric.Enabled {
		return OpHead
	}
	h, g, p := rc.Weights(cfg, rx, tx)
	t := (h + g + p) * rng.Float64()
	var acc float64
	acc += h
	if t < acc {
		return OpHead
	}
	acc += g
	if t < acc {
		return OpGet
	}
	return OpPost
}

func DryEstimateBytes(cfg *config.Config, op OpKind) (rx, tx int64) {
	if cfg == nil {
		return 4096, 4096
	}
	if !cfg.Asymmetric.Enabled {
		return 4096, 4096
	}
	a := cfg.Asymmetric
	switch op {
	case OpHead:
		return a.HeadEstimateRxBytes, a.HeadEstimateTxBytes
	case OpGet:
		return a.DownloadMaxBytes, 512
	case OpPost:
		return 512, a.UploadMaxBytes
	default:
		return a.HeadEstimateRxBytes, a.HeadEstimateTxBytes
	}
}
