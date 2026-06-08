package main

import (
	"regexp"
	"strings"
	"sync"
)

type RewriteResult struct {
	Matched bool
	Rule    Rule
	NewURL  string
}

var (
	regexCache   = map[string]*regexp.Regexp{}
	regexCacheMu sync.RWMutex
)

// getRegex returns a cached compiled regex for the given pattern.
func getRegex(pattern string) *regexp.Regexp {
	regexCacheMu.RLock()
	re, ok := regexCache[pattern]
	regexCacheMu.RUnlock()
	if ok {
		return re
	}

	parts := strings.Split(pattern, "*")
	var sb strings.Builder
	sb.WriteString("^")
	for i, part := range parts {
		sb.WriteString(regexp.QuoteMeta(part))
		if i < len(parts)-1 {
			sb.WriteString("(.*)")
		}
	}
	sb.WriteString("$")
	re = regexp.MustCompile(sb.String())

	regexCacheMu.Lock()
	regexCache[pattern] = re
	regexCacheMu.Unlock()
	return re
}

func matchAndCapture(pattern, url string) ([]string, bool) {
	re := getRegex(pattern)
	matches := re.FindStringSubmatch(url)
	if matches == nil {
		return nil, false
	}
	return matches[1:], true
}

func rewriteURL(target string, patternCaptures []string, pattern string) string {
	targetParts := strings.Split(target, "*")
	targetStarCount := len(targetParts) - 1
	patternStarCount := len(strings.Split(pattern, "*")) - 1

	if targetStarCount > len(patternCaptures) {
		return target
	}

	// When pattern has more * than target, align from the right.
	// e.g. pattern "http://*.*:8080/api/*" has 3 captures,
	//      target  "http://127.0.0.1:8080/api/*" has 1 wildcard.
	//      Use the last capture, not the first.
	offset := patternStarCount - targetStarCount

	var sb strings.Builder
	for i, part := range targetParts {
		sb.WriteString(part)
		if i < len(targetParts)-1 {
			sb.WriteString(patternCaptures[offset+i])
		}
	}
	return sb.String()
}

func applyRewrite(rules []Rule, url string) RewriteResult {
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		captures, ok := matchAndCapture(rule.MatchPattern, url)
		if !ok {
			continue
		}
		newURL := rewriteURL(rule.TargetURL, captures, rule.MatchPattern)
		return RewriteResult{
			Matched: true,
			Rule:    rule,
			NewURL:  newURL,
		}
	}
	return RewriteResult{Matched: false}
}
