package target

import (
	"fmt"
	"strings"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
)

func HostForDomain(d *routercommon.Domain, wwwRoot bool) (string, error) {
	switch d.GetType() {
	case routercommon.Domain_Full:
		return strings.ToLower(d.GetValue()), nil
	case routercommon.Domain_RootDomain:
		v := strings.ToLower(d.GetValue())
		if wwwRoot {
			return "www." + v, nil
		}
		return v, nil
	case routercommon.Domain_Plain:
		v := d.GetValue()
		if strings.Contains(v, ".") && !strings.ContainsAny(v, "*?[]") {
			return strings.ToLower(v), nil
		}
		return "", fmt.Errorf("skip plain rule")
	case routercommon.Domain_Regex:
		return "", fmt.Errorf("skip regex rule")
	default:
		return "", fmt.Errorf("unknown domain type")
	}
}

func HostAllowed(host string, allowedSuffixes []string) bool {
	if len(allowedSuffixes) == 0 {
		return true
	}
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, suf := range allowedSuffixes {
		suf = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(suf), "."))
		if suf == "" {
			continue
		}
		if h == suf || strings.HasSuffix(h, "."+suf) {
			return true
		}
	}
	return false
}
