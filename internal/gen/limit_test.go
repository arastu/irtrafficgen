package gen

import (
	"context"
	"testing"
	"time"
)

func TestPerHostLimitersTwoHosts(t *testing.T) {
	ctx := context.Background()
	p := NewPerHostLimiters(100, 256)
	start := time.Now()
	_ = p.Wait(ctx, "a")
	_ = p.Wait(ctx, "b")
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("first waits should be immediate")
	}
}
