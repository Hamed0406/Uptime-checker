package probe

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

type DNSStatus struct {
	Domain        string
	HasAOrAAAA    bool
	IPs           []net.IP
	CNAME         string
	HasNS         bool
	Nameservers   []string
	Class         string // "NXDOMAIN" | "NO_A_RECORD" | "RESOLVES" | "SERVFAIL_or_TIMEOUT" | "INVALID_NAME"
	ResolverError string
}

var dnsTimeout = 3 * time.Second

func CheckDNS(domain string) DNSStatus {
	s := DNSStatus{Domain: strings.TrimSpace(domain)}
	if s.Domain == "" || strings.Contains(s.Domain, "://") {
		s.Class = "INVALID_NAME"
		return s
	}

	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeout)
	defer cancel()
	r := &net.Resolver{} // OS resolver

	ips, err := r.LookupIP(ctx, "ip", s.Domain)
	if err == nil && len(ips) > 0 {
		s.HasAOrAAAA = true
		s.IPs = ips
		s.Class = "RESOLVES"
	} else if err != nil {
		var de *net.DNSError
		s.ResolverError = err.Error()
		if errors.As(err, &de) {
			if de.IsNotFound {
				s.Class = "NXDOMAIN"
			} else if de.IsTemporary || de.Timeout() {
				s.Class = "SERVFAIL_or_TIMEOUT"
			}
		}
	}

	if cname, err := r.LookupCNAME(ctx, s.Domain); err == nil && !strings.EqualFold(cname, s.Domain+".") {
		s.CNAME = strings.TrimSuffix(cname, ".")
	}

	if ns, err := r.LookupNS(ctx, s.Domain); err == nil && len(ns) > 0 {
		s.HasNS = true
		for _, n := range ns {
			s.Nameservers = append(s.Nameservers, strings.TrimSuffix(n.Host, "."))
		}
		if s.Class == "NXDOMAIN" {
			s.Class = "NO_A_RECORD"
		}
	}

	if s.Class == "" {
		if s.HasAOrAAAA {
			s.Class = "RESOLVES"
		} else if s.HasNS {
			s.Class = "NO_A_RECORD"
		} else if s.ResolverError != "" {
			s.Class = "SERVFAIL_or_TIMEOUT"
		} else {
			s.Class = "NXDOMAIN"
		}
	}
	return s
}
