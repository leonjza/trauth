package trauth

import (
	"net/http"
	"regexp"
)

type Exclude struct {
	Exclude string `yaml:"exclude"`
	Regex   *regexp.Regexp
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

		// check paths in rules to see if a regexp matches the url path
		for _, res := range rule.Excludes {
			if res.Regex.MatchString(req.URL.Path) {
				return true
			}
		}
	}

	return false
}
