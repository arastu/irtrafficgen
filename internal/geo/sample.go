package geo

import (
	"fmt"
	"math/rand/v2"
	"net"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func CIDRToIPNet(c *routercommon.CIDR) (*net.IPNet, error) {
	ip := net.IP(c.GetIp())
	n := len(ip)
	if n != net.IPv4len && n != net.IPv6len {
		return nil, net.InvalidAddrError("bad ip length")
	}
	ones := c.GetPrefix()
	if int(ones) < 0 || int(ones) > n*8 {
		return nil, fmt.Errorf("invalid prefix %d", ones)
	}
	mask := net.CIDRMask(int(ones), n*8)
	return &net.IPNet{IP: ip.Mask(mask), Mask: mask}, nil
}

func RandomIPInNet(n *net.IPNet, rng *rand.Rand) net.IP {
	ip := make(net.IP, len(n.IP))
	copy(ip, n.IP)
	for i := range ip {
		r := byte(rng.Uint32())
		ip[i] = n.IP[i] | (r & ^n.Mask[i])
	}
	return ip
}

func SamplePublicIPFromCIDRs(cidrs []*routercommon.CIDR, denyPrivate bool, rng *rand.Rand, maxAttempts int) (net.IP, error) {
	if len(cidrs) == 0 {
		return nil, fmt.Errorf("no cidrs")
	}
	for range maxAttempts {
		c := cidrs[rng.IntN(len(cidrs))]
		ipNet, err := CIDRToIPNet(c)
		if err != nil {
			continue
		}
		ip := RandomIPInNet(ipNet, rng)
		if denyPrivate && !isPublicIP(ip) {
			continue
		}
		return ip, nil
	}
	return nil, fmt.Errorf("exhausted attempts sampling public ip")
}

func isPublicIP(ip net.IP) bool {
	if len(ip) == 0 {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		return true
	}
	if len(ip) == net.IPv6len {
		if ip.IsUnspecified() || ip.Equal(net.IPv6loopback) {
			return false
		}
		if ip[0] == 0xfc || ip[0] == 0xfd {
			return false
		}
		return true
	}
	return false
}
