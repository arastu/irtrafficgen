package metrics

import (
	"sync/atomic"
)

type Counters struct {
	Attempts      atomic.Uint64
	Successes     atomic.Uint64
	ErrTimeout    atomic.Uint64
	ErrTLS        atomic.Uint64
	ErrHTTP       atomic.Uint64
	ErrOther      atomic.Uint64
	DNSAttempts   atomic.Uint64
	DNSSuccesses  atomic.Uint64
	DNSErrors     atomic.Uint64
}

func (c *Counters) Reset() {
	c.Attempts.Store(0)
	c.Successes.Store(0)
	c.ErrTimeout.Store(0)
	c.ErrTLS.Store(0)
	c.ErrHTTP.Store(0)
	c.ErrOther.Store(0)
	c.DNSAttempts.Store(0)
	c.DNSSuccesses.Store(0)
	c.DNSErrors.Store(0)
}
