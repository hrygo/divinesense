// Package filter provides sensitive information filtering for AI outputs.
// It implements Issue #101: Sensitive information filtering with <1ms performance.
package filter

import (
	"regexp"
	"sync"
	"time"
)

// FilterType defines the type of sensitive information to filter.
type FilterType int

const (
	// Phone filters mobile phone numbers.
	Phone FilterType = iota

	// IDCard filters Chinese ID card numbers.
	IDCard

	// Email filters email addresses.
	Email

	// BankCard filters bank card numbers.
	BankCard

	// IP filters IP addresses.
	IP

	// All filters all known sensitive types.
	All
)

// FilterConfig configures the sensitive information filter.
type FilterConfig struct {
	// Enabled filter types.
	Enabled []FilterType

	// MaskChar is the character used for masking.
	MaskChar rune

	// PreserveLength determines whether to preserve original length.
	PreserveLength bool

	// KeepFirstN keeps first N characters unmasked.
	KeepFirstN int

	// KeepLastN keeps last N characters unmasked.
	KeepLastN int
}

// DefaultConfig returns default filter configuration.
func DefaultConfig() FilterConfig {
	return FilterConfig{
		Enabled:        []FilterType{Phone, IDCard, Email, BankCard, IP},
		MaskChar:       '*',
		PreserveLength: true,
		KeepFirstN:     3,
		KeepLastN:      4,
	}
}

// Filter filters sensitive information from text.
type Filter struct {
	config    FilterConfig
	regexes   map[FilterType]*regexp.Regexp
	mu        sync.RWMutex
	matchPool *sync.Pool
	stats     *filterStats
}

type filterStats struct {
	totalFiltered   int64
	totalMatches    int64
	phoneMatches    int64
	idCardMatches   int64
	emailMatches    int64
	bankCardMatches int64
	ipMatches       int64
	totalNs         int64
}

// NewFilter creates a new sensitive information filter.
func NewFilter(cfg FilterConfig) *Filter {
	if len(cfg.Enabled) == 0 {
		cfg.Enabled = []FilterType{Phone, IDCard, Email, BankCard, IP}
	}

	f := &Filter{
		config:  cfg,
		regexes: make(map[FilterType]*regexp.Regexp),
		matchPool: &sync.Pool{
			New: func() interface{} {
				return make([]Match, 0, 16)
			},
		},
		stats: &filterStats{},
	}

	// Compile regexes
	f.compileRegexes()

	return f
}

// DefaultFilter creates a filter with default configuration.
func DefaultFilter() *Filter {
	return NewFilter(DefaultConfig())
}

// compileRegexes compiles regex patterns for enabled filter types.
func (f *Filter) compileRegexes() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, ft := range f.config.Enabled {
		if ft == All {
			continue
		}
		if _, exists := f.regexes[ft]; exists {
			continue
		}

		pattern := getPattern(ft)
		if pattern != "" {
			f.regexes[ft] = regexp.MustCompile(pattern)
		}
	}
}

// getPattern returns the regex pattern for a filter type.
func getPattern(ft FilterType) string {
	switch ft {
	case Phone:
		// Chinese mobile: 1[3-9]\d{9}
		return `1[3-9]\d{9}`
	case IDCard:
		// Chinese ID: 18 digits, possibly with X at end
		return `\b[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]\b`
	case Email:
		// Standard email pattern
		return `\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`
	case BankCard:
		// Bank card: 12-19 digits
		return `\b\d{12,19}\b`
	case IP:
		// IPv4 address
		return `\b(?:(?:25[0-5]|2[0-4]\d|1?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|1?\d\d?)\b`
	default:
		return ""
	}
}

// Match represents a single match found in text.
type Match struct {
	Type     FilterType
	Start    int
	End      int
	Original string
	Replaced string
}

// FilterText filters sensitive information from text.
func (f *Filter) FilterText(text string) string {
	matches := f.FindMatches(text)
	if len(matches) == 0 {
		return text
	}

	// Sort matches by start position in reverse order
	// This allows us to replace from end to start without offset issues
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].Start < matches[j].Start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Build result by replacing matches from end to start
	result := text
	for _, match := range matches {
		// Use the pre-computed masked string
		prefix := result[:match.Start]
		suffix := result[match.End:]
		result = prefix + match.Replaced + suffix
	}

	return result
}

// FindMatches finds all sensitive information matches in text.
func (f *Filter) FindMatches(text string) []Match {
	start := time.Now()
	defer func() {
		f.recordStats(len(f.regexes), time.Since(start))
	}()

	v := f.matchPool.Get()
	matches, ok := v.([]Match)
	if !ok {
		matches = make([]Match, 0, 16)
	} else {
		matches = matches[:0] // Reset length while keeping capacity
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	for ft, re := range f.regexes {
		foundMatches := re.FindAllStringIndex(text, -1)
		for _, match := range foundMatches {
			original := text[match[0]:match[1]]
			matches = append(matches, Match{
				Type:     ft,
				Start:    match[0],
				End:      match[1],
				Original: original,
				Replaced: f.maskString(original, ft),
			})

			// Update stats
			f.recordMatch(ft)
		}
	}

	// Return a copy before returning
	result := make([]Match, len(matches))
	copy(result, matches)

	// Note: We don't return matches to pool as sync.Pool prefers pointer types
	// The slice will be garbage collected normally

	return result
}

// maskString masks a sensitive string according to configuration.
func (f *Filter) maskString(s string, ft FilterType) string {
	// Special handling for email - preserve @ and domain structure
	if ft == Email {
		return maskEmail(s, f.config.KeepFirstN, f.config.KeepLastN, f.config.MaskChar)
	}

	runes := []rune(s)
	length := len(runes)

	if length <= f.config.KeepFirstN+f.config.KeepLastN {
		return s // Too short to meaningfully mask
	}

	for i := f.config.KeepFirstN; i < length-f.config.KeepLastN; i++ {
		runes[i] = f.config.MaskChar
	}

	return string(runes)
}

// maskEmail masks an email address while preserving @ and domain structure.
func maskEmail(email string, keepFirst, keepLast int, maskChar rune) string {
	runes := []rune(email)
	length := len(runes)

	// Find @ position
	atPos := -1
	for i, r := range runes {
		if r == '@' {
			atPos = i
			break
		}
	}

	if atPos == -1 {
		// Not a valid email, fall back to default masking
		for i := keepFirst; i < length-keepLast; i++ {
			if i >= 0 && i < len(runes) {
				runes[i] = maskChar
			}
		}
		return string(runes)
	}

	// Mask username part (before @)
	for i := keepFirst; i < atPos; i++ {
		if i >= 0 && i < len(runes) {
			runes[i] = maskChar
		}
	}

	// Find domain dot position for domain masking
	dotPos := -1
	for i := length - 1; i > atPos; i-- {
		if runes[i] == '.' {
			dotPos = i
			break
		}
	}

	// Mask domain part (between @ and last dot)
	if dotPos != -1 {
		for i := atPos + 1; i < dotPos; i++ {
			if i >= 0 && i < len(runes) {
				runes[i] = maskChar
			}
		}
	} else {
		// No dot found, mask everything after @ except last keepLast chars
		for i := atPos + 1; i < length-keepLast; i++ {
			if i >= 0 && i < len(runes) {
				runes[i] = maskChar
			}
		}
	}

	return string(runes)
}

// FilterWithOptions filters text with custom options.
func (f *Filter) FilterWithOptions(text string, keepFirst, keepLast int, maskChar rune) string {
	oldFirst := f.config.KeepFirstN
	oldLast := f.config.KeepLastN
	oldChar := f.config.MaskChar

	f.config.KeepFirstN = keepFirst
	f.config.KeepLastN = keepLast
	f.config.MaskChar = maskChar

	result := f.FilterText(text)

	f.config.KeepFirstN = oldFirst
	f.config.KeepLastN = oldLast
	f.config.MaskChar = oldChar

	return result
}

// Validate checks if text contains any unfiltered sensitive information.
func (f *Filter) Validate(text string) bool {
	matches := f.FindMatches(text)
	return len(matches) == 0
}

// GetStats returns filter statistics.
func (f *Filter) GetStats() *FilterStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	total := f.stats.totalMatches
	if total == 0 {
		return &FilterStats{}
	}

	avgNs := f.stats.totalNs / total

	return &FilterStats{
		TotalFiltered:   f.stats.totalFiltered,
		TotalMatches:    total,
		PhoneMatches:    f.stats.phoneMatches,
		IDCardMatches:   f.stats.idCardMatches,
		EmailMatches:    f.stats.emailMatches,
		BankCardMatches: f.stats.bankCardMatches,
		IPMatches:       f.stats.ipMatches,
		AverageLatency:  time.Duration(avgNs),
	}
}

// FilterStats contains filter statistics.
type FilterStats struct {
	TotalFiltered   int64
	TotalMatches    int64
	PhoneMatches    int64
	IDCardMatches   int64
	EmailMatches    int64
	BankCardMatches int64
	IPMatches       int64
	AverageLatency  time.Duration
}

func (f *Filter) recordMatch(ft FilterType) {
	switch ft {
	case Phone:
		f.stats.phoneMatches++
	case IDCard:
		f.stats.idCardMatches++
	case Email:
		f.stats.emailMatches++
	case BankCard:
		f.stats.bankCardMatches++
	case IP:
		f.stats.ipMatches++
	}
	f.stats.totalMatches++
}

func (f *Filter) recordStats(matchCount int, duration time.Duration) {
	f.stats.totalFiltered++
	f.stats.totalNs += duration.Nanoseconds()
}

// ValidatePhone checks if a string is a valid Chinese mobile phone number.
func ValidatePhone(s string) bool {
	re := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return re.MatchString(s)
}

// ValidateIDCard checks if a string is a valid Chinese ID card number.
func ValidateIDCard(s string) bool {
	re := regexp.MustCompile(`^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)
	return re.MatchString(s)
}

// ValidateEmail checks if a string is a valid email address.
func ValidateEmail(s string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(s)
}

// ValidateBankCard checks if a string is a valid bank card number.
func ValidateBankCard(s string) bool {
	re := regexp.MustCompile(`^\d{12,19}$`)
	return re.MatchString(s)
}

// ValidateIP checks if a string is a valid IPv4 address.
func ValidateIP(s string) bool {
	re := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4]\d|1?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|1?\d\d?)$`)
	return re.MatchString(s)
}
