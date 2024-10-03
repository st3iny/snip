package router

import (
	"fmt"
	"strings"
)

type Frontend struct {
	Match   []DomainMatcher
	Backend *Backend
}

func (frontend *Frontend) MatchesDomain(domain string) bool {
	for _, matcher := range frontend.Match {
		if matcher.Match(domain) {
			return true
		}
	}

	return false
}

type DomainMatcher interface {
	Match(domain string) bool
}

func ParseMatcher(match string) (DomainMatcher, error) {
	match = strings.TrimSpace(match)

	if match == "" {
		return nil, fmt.Errorf("Given match is empty")
	} else if match == "*" {
		return &catchAllMatcher{}, nil
	} else if strings.HasPrefix(match, "*") {
		suffix := strings.TrimPrefix(match, "*")
		return &wildcardMatcher{suffix}, nil
	} else {
		return &fqdnMatcher{domain: match}, nil
	}
}

type fqdnMatcher struct {
	domain string
}

type wildcardMatcher struct {
	suffix string
}

type catchAllMatcher struct {
}

func (matcher *fqdnMatcher) Match(domain string) bool {
	return matcher.domain == domain
}

func (matcher *wildcardMatcher) Match(domain string) bool {
	return strings.HasSuffix(domain, matcher.suffix)
}

func (matcher *catchAllMatcher) Match(domain string) bool {
	return true
}
