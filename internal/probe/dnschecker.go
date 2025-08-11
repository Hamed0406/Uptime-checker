package probe

import (
	"context"
	"net/url"
)

type DNSChecker struct{}

func NewDNSChecker() *DNSChecker {
	return &DNSChecker{}
}

func (d *DNSChecker) Check(ctx context.Context, target string) CheckResult {
	host := extractHost(target)
	dns := CheckDNS(host)

	return CheckResult{
		Name:    "DNS",
		Success: dns.Class == "RESOLVES",
		Message: dns.Class,
	}
}

func extractHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return raw
	}
	return u.Hostname()
}
