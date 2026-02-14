package store

// Metadata key constants for AIBlock.Metadata field.
// These keys follow the context-engineering.md architecture for
// state management and sticky routing.
const (
	// MetadataKeyLastAgent stores the last agent type for sticky routing.
	// Values: "memo", "schedule", "amazing", "geek", "evolution"
	MetadataKeyLastAgent = "last_agent"

	// MetadataKeyIntent stores the classified intent.
	// Values: "search", "create", "query", "edit", "delete", "unknown"
	MetadataKeyIntent = "intent"

	// MetadataKeyIntentConfidence stores the intent classification confidence (0-1).
	MetadataKeyIntentConfidence = "intent_confidence"

	// MetadataKeyRouteMethod stores how the routing decision was made.
	// Values: "cache", "rule", "llm", "sticky", "metadata_sticky"
	MetadataKeyRouteMethod = "route_method"

	// MetadataKeyStickyUntil stores the sticky window expiration timestamp (Unix seconds).
	MetadataKeyStickyUntil = "sticky_until"

	// MetadataKeyStickyCount stores the number of times sticky routing has been extended.
	MetadataKeyStickyCount = "sticky_count"

	// MetadataKeyTopics stores the conversation topics extracted from the block.
	// Values: []string{"schedule", "memo", "general"}
	MetadataKeyTopics = "topics"

	// MetadataKeyEntities stores extracted entities from the conversation.
	// Values: map[string]string{"date": "2024-01-15", "location": "Beijing"}
	MetadataKeyEntities = "entities"
)

// GetMetadataLastAgent retrieves the last agent from block metadata.
func (b *AIBlock) GetMetadataLastAgent() (string, bool) {
	if b.Metadata == nil {
		return "", false
	}
	val, ok := b.Metadata[MetadataKeyLastAgent].(string)
	return val, ok
}

// GetMetadataIntent retrieves the intent from block metadata.
func (b *AIBlock) GetMetadataIntent() (string, bool) {
	if b.Metadata == nil {
		return "", false
	}
	val, ok := b.Metadata[MetadataKeyIntent].(string)
	return val, ok
}

// GetMetadataIntentConfidence retrieves the intent confidence from block metadata.
func (b *AIBlock) GetMetadataIntentConfidence() (float32, bool) {
	if b.Metadata == nil {
		return 0, false
	}
	// Handle both float64 (from JSON) and float32
	switch v := b.Metadata[MetadataKeyIntentConfidence].(type) {
	case float64:
		return float32(v), true
	case float32:
		return v, true
	default:
		return 0, false
	}
}

// GetMetadataStickyUntil retrieves the sticky expiration timestamp.
func (b *AIBlock) GetMetadataStickyUntil() (int64, bool) {
	if b.Metadata == nil {
		return 0, false
	}
	// Handle both float64 (from JSON) and int64
	switch v := b.Metadata[MetadataKeyStickyUntil].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

// SetMetadataLastAgent sets the last agent in update metadata.
func (u *UpdateAIBlock) SetMetadataLastAgent(agent string) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]any)
	}
	u.Metadata[MetadataKeyLastAgent] = agent
}

// SetMetadataIntent sets the intent in update metadata.
func (u *UpdateAIBlock) SetMetadataIntent(intent string) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]any)
	}
	u.Metadata[MetadataKeyIntent] = intent
}

// SetMetadataIntentConfidence sets the intent confidence in update metadata.
func (u *UpdateAIBlock) SetMetadataIntentConfidence(confidence float32) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]any)
	}
	u.Metadata[MetadataKeyIntentConfidence] = confidence
}

// SetMetadataStickyUntil sets the sticky expiration timestamp.
func (u *UpdateAIBlock) SetMetadataStickyUntil(unixSeconds int64) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]any)
	}
	u.Metadata[MetadataKeyStickyUntil] = unixSeconds
}

// SetMetadataRouteMethod sets the routing method.
func (u *UpdateAIBlock) SetMetadataRouteMethod(method string) {
	if u.Metadata == nil {
		u.Metadata = make(map[string]any)
	}
	u.Metadata[MetadataKeyRouteMethod] = method
}
