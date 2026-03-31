package gen

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type rxByteLimiter struct {
	ctx context.Context
	r   io.Reader
	lim *rate.Limiter
}

func (b *rxByteLimiter) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	if n > 0 && b.lim != nil {
		if werr := b.lim.WaitN(b.ctx, n); werr != nil {
			return n, werr
		}
	}
	return n, err
}

type txByteLimiter struct {
	ctx context.Context
	r   io.Reader
	lim *rate.Limiter
}

func (b *txByteLimiter) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return b.r.Read(p)
	}
	n, err := b.r.Read(p)
	if n > 0 && b.lim != nil {
		if werr := b.lim.WaitN(b.ctx, n); werr != nil {
			return n, werr
		}
	}
	return n, err
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
