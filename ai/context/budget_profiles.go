// Package context provides budget profiles for dynamic token allocation.
// Issue #93: 动态 Token 预算分配 - 按意图类型自适应调整
package context

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
)

// BudgetProfile defines token allocation ratios for different intents.
type BudgetProfile struct {
	Name        string
	Description string

	// Token allocation ratios for the remaining budget (after SystemPrompt + UserPrefs)
	// Must sum to 1.0
	ShortTermRatio float64
	LongTermRatio  float64
	RetrievalRatio float64

	// Fixed ratios
	UserPrefsRatio float64 // Default: 0.10
}

// DefaultBudgetProfiles returns the built-in profile registry.
var DefaultBudgetProfiles = map[string]*BudgetProfile{
	"memo_search": {
		Name:           "memo_search",
		Description:    "Memo search intent - prioritize retrieval results",
		ShortTermRatio: 0.30,
		LongTermRatio:  0.10,
		RetrievalRatio: 0.60,
		UserPrefsRatio: 0.10,
	},
	"schedule_create": {
		Name:           "schedule_create",
		Description:    "Schedule creation - moderate conversation, low retrieval",
		ShortTermRatio: 0.55,
		LongTermRatio:  0.25,
		RetrievalRatio: 0.20,
		UserPrefsRatio: 0.10,
	},
	"schedule_query": {
		Name:           "schedule_query",
		Description:    "Schedule query - moderate conversation, low retrieval",
		ShortTermRatio: 0.55,
		LongTermRatio:  0.25,
		RetrievalRatio: 0.20,
		UserPrefsRatio: 0.10,
	},
	"amazing": {
		Name:           "amazing",
		Description:    "Amazing parrot - balanced allocation",
		ShortTermRatio: 0.40,
		LongTermRatio:  0.15,
		RetrievalRatio: 0.45,
		UserPrefsRatio: 0.10,
	},
	"geek": {
		Name:           "geek",
		Description:    "Geek parrot - no LLM budget (Claude Code CLI manages context)",
		ShortTermRatio: 0.00,
		LongTermRatio:  0.00,
		RetrievalRatio: 0.00,
		UserPrefsRatio: 0.00,
	},
	"evolution": {
		Name:           "evolution",
		Description:    "Evolution parrot - no LLM budget (Claude Code CLI manages context)",
		ShortTermRatio: 0.00,
		LongTermRatio:  0.00,
		RetrievalRatio: 0.00,
		UserPrefsRatio: 0.00,
	},
	"default": {
		Name:           "default",
		Description:    "Fallback profile - balanced allocation",
		ShortTermRatio: 0.40,
		LongTermRatio:  0.15,
		RetrievalRatio: 0.45,
		UserPrefsRatio: 0.10,
	},
}

// ProfileRegistry manages budget profiles with override support.
type ProfileRegistry struct {
	mu       sync.RWMutex
	profiles map[string]*BudgetProfile
}

// NewProfileRegistry creates a new profile registry with defaults.
func NewProfileRegistry() *ProfileRegistry {
	return &ProfileRegistry{
		profiles: cloneProfiles(DefaultBudgetProfiles),
	}
}

// Get returns a profile by name (thread-safe).
func (r *ProfileRegistry) Get(name string) (*BudgetProfile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.profiles[name]
	if !ok {
		// Fallback to default
		profile, ok = r.profiles["default"]
		if !ok {
			// Create default if somehow missing
			profile = &BudgetProfile{
				Name:           "default",
				Description:    "Fallback profile",
				ShortTermRatio: 0.40,
				LongTermRatio:  0.15,
				RetrievalRatio: 0.45,
				UserPrefsRatio: 0.10,
			}
		}
	}
	return profile, true
}

// Set adds or overrides a profile (thread-safe).
func (r *ProfileRegistry) Set(name string, profile *BudgetProfile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[name] = profile
}

// LoadFromEnv overrides profiles from environment variables.
// Format: DIVINESENSE_BUDGET_<PROFILE>_<FIELD>=<value>
// Example: DIVINESENSE_BUDGET_MEMO_SEARCH_SHORT_TERM=0.35
func (r *ProfileRegistry) LoadFromEnv() {
	const prefix = "DIVINESENSE_BUDGET_"

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		// Parse: DIVINESENSE_BUDGET_MEMO_SEARCH_SHORT_TERM=0.35
		parts := strings.SplitN(strings.TrimPrefix(env, prefix), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Split key into profile and field
		// "MEMO_SEARCH_SHORT_TERM" -> ["MEMO_SEARCH", "SHORT_TERM"]
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 3 {
			continue
		}

		// Find the field name (last 1 or 2 parts)
		var fieldName string
		var profileName string

		// Check for 2-part field names like SHORT_TERM
		if len(keyParts) >= 3 && (keyParts[len(keyParts)-1] == "TERM" || keyParts[len(keyParts)-1] == "RATIO") {
			// Last 2 parts form the field name
			fieldIdx := len(keyParts) - 2
			fieldName = strings.Join(keyParts[fieldIdx:], "_")
			profileName = strings.Join(keyParts[:fieldIdx], "_")
		} else {
			// Single part field name
			fieldName = keyParts[len(keyParts)-1]
			profileName = strings.Join(keyParts[:len(keyParts)-1], "_")
		}

		profileName = strings.ToLower(profileName)

		// Get or create profile
		profile, _ := r.Get(profileName)
		if profile.Name == "default" {
			// Create a new profile if we're using default
			profile = &BudgetProfile{
				Name:           profileName,
				Description:    "Custom profile from env",
				ShortTermRatio: 0.40,
				LongTermRatio:  0.15,
				RetrievalRatio: 0.45,
				UserPrefsRatio: 0.10,
			}
		}

		// Parse float value
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			slog.Warn("Invalid budget config value", "key", key, "value", value, "error", err)
			continue
		}

		// Update field
		switch strings.ToUpper(fieldName) {
		case "SHORT_TERM", "SHORT_TERM_RATIO":
			profile.ShortTermRatio = floatVal
		case "LONG_TERM", "LONG_TERM_RATIO":
			profile.LongTermRatio = floatVal
		case "RETRIEVAL", "RETRIEVAL_RATIO":
			profile.RetrievalRatio = floatVal
		case "USER_PREFS", "USER_PREFS_RATIO":
			profile.UserPrefsRatio = floatVal
		default:
			slog.Warn("Unknown budget config field", "field", fieldName)
			continue
		}

		r.Set(profileName, profile)
		slog.Info("Budget profile updated from env", "profile", profileName, "field", fieldName, "value", floatVal)
	}
}

// cloneProfiles creates a deep copy of the profiles map.
func cloneProfiles(src map[string]*BudgetProfile) map[string]*BudgetProfile {
	dst := make(map[string]*BudgetProfile, len(src))
	for k, v := range src {
		dst[k] = &BudgetProfile{
			Name:           v.Name,
			Description:    v.Description,
			ShortTermRatio: v.ShortTermRatio,
			LongTermRatio:  v.LongTermRatio,
			RetrievalRatio: v.RetrievalRatio,
			UserPrefsRatio: v.UserPrefsRatio,
		}
	}
	return dst
}

// IntentResolver infers task intent from AgentType.
type IntentResolver struct {
	// Future: can be enhanced with ML-based classification
}

// NewIntentResolver creates a new intent resolver.
func NewIntentResolver() *IntentResolver {
	return &IntentResolver{}
}

// Resolve infers intent from AgentType (thread-safe).
// Returns intent name (e.g., "memo_search", "schedule_create").
func (r *IntentResolver) Resolve(agentType string) string {
	// Normalize input
	agentType = strings.ToLower(strings.TrimSpace(agentType))

	switch agentType {
	case "memo":
		return "memo_search"
	case "schedule":
		return "schedule_create" // Default to create (can be refined with LLM)
	case "amazing":
		return "amazing"
	case "geek":
		return "geek" // Special case: no LLM budget needed
	case "evolution":
		return "evolution" // Special case: maximum priority
	case "auto":
		return "amazing" // AUTO defaults to amazing (most versatile)
	default:
		return "default"
	}
}
