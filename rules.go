package trauth

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

type Exclude struct {
	Path  string `yaml:"path"`
	IPNet string `yaml:"ipnet"`

	// "computed" values from configuration parsing
	regexPath *regexp.Regexp
	ipNet     *net.IPNet
}

// Rule defines a trauth rule to exclude authentication
type Rule struct {
	Domain   string    `yaml:"domain"`
	Excludes []Exclude `yaml:"excludes"`
}

func skipViaRule(rules []Rule, req *http.Request) bool {
	for _, rule := range rules {

		// skip processing rules for domains that dont match
		if req.Host != rule.Domain {
			continue
		}

		source := net.ParseIP(strings.Split(req.RemoteAddr, ":")[0])

		for _, exclude := range rule.Excludes {

			// check source ip rules
			if source != nil && exclude.ipNet != nil {
				if exclude.ipNet.Contains(source) {
					return true
				}
			}

			// check path rules
			if exclude.regexPath != nil {
				if exclude.regexPath.MatchString(req.URL.Path) {
					return true
				}
			}
		}
	}

	return false
}
