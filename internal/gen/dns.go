package gen

import (
	"context"
	"net"
)

func LookupHost(ctx context.Context, r *net.Resolver, host string) ([]string, error) {
	if r == nil {
		r = net.DefaultResolver
	}
	return r.LookupHost(ctx, host)
}
