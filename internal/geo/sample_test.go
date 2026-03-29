package geo

import (
	"math/rand/v2"
	"testing"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func TestSamplePublicIPFromCIDRs(t *testing.T) {
	cidrs := []*routercommon.CIDR{
		{Ip: []byte{10, 0, 0, 0}, Prefix: 8},
		{Ip: []byte{8, 8, 8, 0}, Prefix: 24},
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		ip, err := SamplePublicIPFromCIDRs(cidrs, true, rng, 32)
		if err != nil {
			t.Fatal(err)
		}
		if ip.To4() == nil {
			t.Fatal("expected v4")
		}
		if !isPublicIP(ip) {
			t.Fatalf("not public %v", ip)
		}
		n, err := CIDRToIPNet(cidrs[1])
		if err != nil {
			t.Fatal(err)
		}
		if !n.Contains(ip) {
			t.Fatalf("outside net %v", ip)
		}
	}
}

func TestCIDRToIPNetInvalid(t *testing.T) {
	_, err := CIDRToIPNet(&routercommon.CIDR{Ip: []byte{1, 2}, Prefix: 24})
	if err == nil {
		t.Fatal("expected error")
	}
}
