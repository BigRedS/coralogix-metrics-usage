package region

import (
	"fmt"
	"sort"
	"strings"
)

// Short region code -> API hostname (no scheme).
var regionAPIHosts = map[string]string{
	"eu1": "api.coralogix.com",
	"us1": "api.coralogix.us",
	"us2": "api.cx498.coralogix.com",
	"us3": "api.us3.coralogix.com",
	"eu2": "api.eu2.coralogix.com",
	"ap1": "api.coralogix.in",
	"ap2": "api.coralogixsg.com",
	"ap3": "api.ap3.coralogix.com",
}

// RegionChoice is a short region code and its metrics/management API hostname (no scheme).
type RegionChoice struct {
	Code string `json:"code"`
	Host string `json:"host"`
}

// SortedRegionChoices returns stable UI dropdown entries (code ascending).
func SortedRegionChoices() []RegionChoice {
	out := make([]RegionChoice, 0, len(regionAPIHosts))
	for code, host := range regionAPIHosts {
		out = append(out, RegionChoice{Code: code, Host: host})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Team login domain -> API hostname.
var loginDomainToAPIHost = map[string]string{
	"eu1.coralogix.com": "api.coralogix.com",
	"us1.coralogix.com": "api.coralogix.us",
	"us2.coralogix.com": "api.cx498.coralogix.com",
	"us3.coralogix.com": "api.us3.coralogix.com",
	"eu2.coralogix.com": "api.eu2.coralogix.com",
	"ap1.coralogix.com": "api.coralogix.in",
	"ap2.coralogix.com": "api.coralogixsg.com",
	"ap3.coralogix.com": "api.ap3.coralogix.com",
}

// ResolveAPIHost maps a region code, login domain, or API hostname to the management/metrics API host.
func ResolveAPIHost(regionOrDomain string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(regionOrDomain))
	if h, ok := regionAPIHosts[key]; ok {
		return h, nil
	}
	if h, ok := loginDomainToAPIHost[key]; ok {
		return h, nil
	}
	for _, h := range regionAPIHosts {
		if key == h {
			return h, nil
		}
	}
	if strings.HasPrefix(key, "api.") && isHostname(key) {
		return key, nil
	}
	codes := make([]string, 0, len(regionAPIHosts))
	for c := range regionAPIHosts {
		codes = append(codes, c)
	}
	return "", fmt.Errorf(
		"unknown region or domain %q: expected one of %v, a login domain (e.g. eu2.coralogix.com), or an API host (e.g. api.eu2.coralogix.com)",
		regionOrDomain, codes,
	)
}

func isHostname(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func MgmtOpenAPIV5Base(apiHost string) string {
	return "https://" + apiHost + "/mgmt/openapi/5"
}

func MetricsBase(apiHost string) string {
	return "https://" + apiHost + "/metrics"
}
