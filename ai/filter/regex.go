// Package filter provides optimized regex patterns for sensitive data detection.
package filter

import (
	"regexp"
	"strings"
	"sync"
)

// Precompiled regex patterns for common sensitive data formats.
// These patterns are optimized for performance using atomic compilation
// and pattern caching.

var (
	// phonePattern matches Chinese mobile phone numbers: 1[3-9]xxxxxxxxx
	phonePattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b1[3-9]\d{9}\b`)
	})

	// idCardPattern matches 18-digit Chinese ID card numbers
	idCardPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]\b`)
	})

	// emailPattern matches standard email addresses
	emailPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`)
	})

	// bankCardPattern matches 12-19 digit bank card numbers
	bankCardPattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b\d{12,19}\b`)
	})

	// ipv4Pattern matches IPv4 addresses
	ipv4Pattern = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4]\d|1?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|1?\d\d?)\b`)
	})

	// urlPattern matches HTTP/HTTPS URLs (reserved for future use)
	//nolint:unused // Reserved for URL filtering feature
	_ = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\bhttps?://[a-zA-Z0-9.-]+(?:\.[a-zA-Z]{2,})(?:/\S*)?\b`)
	})

	// wechatPattern matches WeChat IDs (reserved for future use)
	//nolint:unused // Reserved for WeChat filtering feature
	_ = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b[a-zA-Z][-a-zA-Z0-9_]{5,19}\b`)
	})

	// qqPattern matches QQ numbers (reserved for future use)
	//nolint:unused // Reserved for QQ filtering feature
	_ = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`\b[1-9]\d{4,10}\b`)
	})
)

// GetPattern returns a precompiled regex pattern for the given filter type.
func GetPattern(ft FilterType) *regexp.Regexp {
	switch ft {
	case Phone:
		return phonePattern()
	case IDCard:
		return idCardPattern()
	case Email:
		return emailPattern()
	case BankCard:
		return bankCardPattern()
	case IP:
		return ipv4Pattern()
	default:
		return nil
	}
}

// FastMatch performs a fast match check using precompiled patterns.
// Returns true if any sensitive data pattern matches the input text.
func FastMatch(text string, types []FilterType) bool {
	for _, ft := range types {
		if re := GetPattern(ft); re != nil {
			if re.MatchString(text) {
				return true
			}
		}
	}
	return false
}

// FindAllMatches finds all matches for the given filter types.
func FindAllMatches(text string, types []FilterType) []Match {
	var matches []Match
	seen := make(map[[2]int]bool) // Deduplicate overlapping matches

	for _, ft := range types {
		re := GetPattern(ft)
		if re == nil {
			continue
		}

		idx := re.FindAllStringIndex(text, -1)
		for _, m := range idx {
			key := [2]int{m[0], m[1]}
			if seen[key] {
				continue
			}
			seen[key] = true

			matches = append(matches, Match{
				Type:     ft,
				Start:    m[0],
				End:      m[1],
				Original: text[m[0]:m[1]],
			})
		}
	}

	return matches
}

// CompositePattern creates a composite regex pattern from multiple filter types.
// This is more efficient for bulk scanning than running each pattern separately.
func CompositePattern(types []FilterType) (*regexp.Regexp, error) {
	var patterns []string
	for _, ft := range types {
		pattern := getRawPattern(ft)
		if pattern != "" {
			patterns = append(patterns, "("+pattern+")")
		}
	}

	if len(patterns) == 0 {
		return nil, nil
	}

	combined := "(?:" + strings.Join(patterns, "|") + ")"
	return regexp.Compile(combined)
}

// getRawPattern returns the raw pattern string for a filter type.
func getRawPattern(ft FilterType) string {
	switch ft {
	case Phone:
		return `1[3-9]\d{9}`
	case IDCard:
		return `[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]`
	case Email:
		return `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
	case BankCard:
		return `\d{12,19}`
	case IP:
		return `(?:(?:25[0-5]|2[0-4]\d|1?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|1?\d\d?)`
	default:
		return ""
	}
}

// PatternSet represents a set of compiled patterns for efficient matching.
type PatternSet struct {
	patterns map[FilterType]*regexp.Regexp
	mu       sync.RWMutex
}

// NewPatternSet creates a new pattern set with the given filter types.
func NewPatternSet(types []FilterType) *PatternSet {
	ps := &PatternSet{
		patterns: make(map[FilterType]*regexp.Regexp),
	}

	for _, ft := range types {
		if re := GetPattern(ft); re != nil {
			ps.patterns[ft] = re
		}
	}

	return ps
}

// Match checks if the text matches any pattern in the set.
func (ps *PatternSet) Match(text string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for _, re := range ps.patterns {
		if re.MatchString(text) {
			return true
		}
	}

	return false
}

// FindAll finds all matches for patterns in the set.
func (ps *PatternSet) FindAll(text string) []Match {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var matches []Match
	seen := make(map[[2]int]bool)

	for ft, re := range ps.patterns {
		idx := re.FindAllStringIndex(text, -1)
		for _, m := range idx {
			key := [2]int{m[0], m[1]}
			if seen[key] {
				continue
			}
			seen[key] = true

			matches = append(matches, Match{
				Type:     ft,
				Start:    m[0],
				End:      m[1],
				Original: text[m[0]:m[1]],
			})
		}
	}

	return matches
}

// Add adds a new pattern type to the set.
func (ps *PatternSet) Add(ft FilterType) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.patterns[ft]; exists {
		return
	}

	if re := GetPattern(ft); re != nil {
		ps.patterns[ft] = re
	}
}

// Remove removes a pattern type from the set.
func (ps *PatternSet) Remove(ft FilterType) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.patterns, ft)
}

// Has returns true if the pattern set contains the given type.
func (ps *PatternSet) Has(ft FilterType) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	_, exists := ps.patterns[ft]
	return exists
}

// Types returns all filter types in the set.
func (ps *PatternSet) Types() []FilterType {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	types := make([]FilterType, 0, len(ps.patterns))
	for ft := range ps.patterns {
		types = append(types, ft)
	}

	return types
}

// FastScanner provides high-performance scanning for sensitive data.
// It uses a composite pattern for single-pass scanning.
type FastScanner struct {
	composite *regexp.Regexp
	types     []FilterType
}

// NewFastScanner creates a new fast scanner for the given types.
func NewFastScanner(types []FilterType) (*FastScanner, error) {
	if len(types) == 0 {
		types = []FilterType{Phone, IDCard, Email, BankCard, IP}
	}

	composite, err := CompositePattern(types)
	if err != nil {
		return nil, err
	}

	return &FastScanner{
		composite: composite,
		types:     types,
	}, nil
}

// Scan scans text for sensitive data matches.
func (fs *FastScanner) Scan(text string) []Match {
	if fs.composite == nil {
		return nil
	}

	// First pass: find all matches using composite pattern
	idx := fs.composite.FindAllStringIndex(text, -1)

	// Second pass: identify which pattern matched each result
	var matches []Match
	seen := make(map[[2]int]bool)

	for _, m := range idx {
		key := [2]int{m[0], m[1]}
		if seen[key] {
			continue
		}
		seen[key] = true

		substr := text[m[0]:m[1]]

		// Identify which pattern matched
		for _, ft := range fs.types {
			if re := GetPattern(ft); re != nil && re.MatchString(substr) {
				matches = append(matches, Match{
					Type:     ft,
					Start:    m[0],
					End:      m[1],
					Original: substr,
				})
				break
			}
		}
	}

	return matches
}

// HasAny returns true if text contains any sensitive data.
func (fs *FastScanner) HasAny(text string) bool {
	if fs.composite == nil {
		return false
	}
	return fs.composite.MatchString(text)
}
