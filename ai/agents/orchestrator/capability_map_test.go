package orchestrator

import (
	"testing"

	agents "github.com/hrygo/divinesense/ai/agents"
)

func TestCapabilityMap_NewCapabilityMap(t *testing.T) {
	cm := NewCapabilityMap()
	if cm == nil {
		t.Fatal("NewCapabilityMap should not return nil")
	}
	if cm.capabilityToExperts == nil {
		t.Error("capabilityToExperts should be initialized")
	}
	if cm.experts == nil {
		t.Error("experts should be initialized")
	}
}

func TestCapabilityMap_BuildFromConfigs(t *testing.T) {
	tests := []struct {
		name         string
		configs      []*agents.ParrotSelfCognition
		expectedCaps int
		expectedExps int
	}{
		{
			name: "single expert with multiple capabilities",
			configs: []*agents.ParrotSelfCognition{
				{
					Name:  "memo",
					Emoji: "📝",
					Title: "Note Expert",
					Capabilities: []string{
						"search_notes",
						"create_note",
						"edit_note",
					},
				},
			},
			expectedCaps: 3,
			expectedExps: 1,
		},
		{
			name: "multiple experts with overlapping capabilities",
			configs: []*agents.ParrotSelfCognition{
				{
					Name:         "memo",
					Emoji:        "📝",
					Title:        "Note Expert",
					Capabilities: []string{"search_notes", "create_note"},
				},
				{
					Name:         "schedule",
					Emoji:        "📅",
					Title:        "Schedule Expert",
					Capabilities: []string{"search_notes", "create_event"},
				},
			},
			expectedCaps: 3, // search_notes, create_note, create_event
			expectedExps: 2,
		},
		{
			name:         "nil configs",
			configs:      nil,
			expectedCaps: 0,
			expectedExps: 0,
		},
		{
			name: "empty configs",
			configs: []*agents.ParrotSelfCognition{
				nil,
			},
			expectedCaps: 0,
			expectedExps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewCapabilityMap()
			cm.BuildFromConfigs(tt.configs)

			// Check experts count
			experts := cm.GetAllExperts()
			if len(experts) != tt.expectedExps {
				t.Errorf("expected %d experts, got %d", tt.expectedExps, len(experts))
			}

			// Check capability count (capabilityToExperts map)
			if tt.expectedCaps > 0 && len(cm.capabilityToExperts) != tt.expectedCaps {
				t.Errorf("expected %d capabilities, got %d", tt.expectedCaps, len(cm.capabilityToExperts))
			}
		})
	}
}

func TestCapabilityMap_FindExpertsByCapability(t *testing.T) {
	cm := NewCapabilityMap()
	cm.BuildFromConfigs([]*agents.ParrotSelfCognition{
		{
			Name:         "memo",
			Emoji:        "📝",
			Title:        "Note Expert",
			Capabilities: []string{"search_notes", "create_note"},
		},
		{
			Name:         "schedule",
			Emoji:        "📅",
			Title:        "Schedule Expert",
			Capabilities: []string{"search_notes", "create_event"},
		},
		{
			Name:         "geek",
			Emoji:        "🤖",
			Title:        "Tech Expert",
			Capabilities: []string{"execute_code"},
		},
	})

	tests := []struct {
		name          string
		capability    string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "search_notes - should find memo and schedule",
			capability:    "search_notes",
			expectedCount: 2,
			expectedNames: []string{"memo", "schedule"},
		},
		{
			name:          "create_note - should find only memo",
			capability:    "create_note",
			expectedCount: 1,
			expectedNames: []string{"memo"},
		},
		{
			name:          "execute_code - should find only geek",
			capability:    "execute_code",
			expectedCount: 1,
			expectedNames: []string{"geek"},
		},
		{
			name:          "non_existent - should return nil",
			capability:    "non_existent",
			expectedCount: 0,
			expectedNames: nil,
		},
		{
			name:          "case insensitive - SEARCH_NOTES",
			capability:    "SEARCH_NOTES",
			expectedCount: 2,
			expectedNames: []string{"memo", "schedule"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			experts := cm.FindExpertsByCapability(tt.capability)
			if len(experts) != tt.expectedCount {
				t.Errorf("expected %d experts, got %d", tt.expectedCount, len(experts))
			}

			if tt.expectedNames != nil {
				for i, exp := range experts {
					if i >= len(tt.expectedNames) {
						break
					}
					if exp.Name != tt.expectedNames[i] {
						t.Errorf("expected expert %s at index %d, got %s", tt.expectedNames[i], i, exp.Name)
					}
				}
			}
		})
	}
}

func TestCapabilityMap_FindAlternativeExperts(t *testing.T) {
	cm := NewCapabilityMap()
	cm.BuildFromConfigs([]*agents.ParrotSelfCognition{
		{
			Name:         "memo",
			Emoji:        "📝",
			Title:        "Note Expert",
			Capabilities: []string{"search_notes", "create_note"},
		},
		{
			Name:         "schedule",
			Emoji:        "📅",
			Title:        "Schedule Expert",
			Capabilities: []string{"search_notes", "create_event"},
		},
	})

	tests := []struct {
		name          string
		capability    string
		excludeExpert string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "exclude memo from search_notes",
			capability:    "search_notes",
			excludeExpert: "memo",
			expectedCount: 1,
			expectedNames: []string{"schedule"},
		},
		{
			name:          "exclude schedule from search_notes",
			capability:    "search_notes",
			excludeExpert: "schedule",
			expectedCount: 1,
			expectedNames: []string{"memo"},
		},
		{
			name:          "exclude non_existent expert",
			capability:    "search_notes",
			excludeExpert: "non_existent",
			expectedCount: 2,
			expectedNames: []string{"memo", "schedule"},
		},
		{
			name:          "capability not found",
			capability:    "non_existent",
			excludeExpert: "memo",
			expectedCount: 0,
			expectedNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			experts := cm.FindAlternativeExperts(tt.capability, tt.excludeExpert)
			if len(experts) != tt.expectedCount {
				t.Errorf("expected %d experts, got %d", tt.expectedCount, len(experts))
			}

			if tt.expectedNames != nil {
				for i, exp := range experts {
					if i >= len(tt.expectedNames) {
						break
					}
					if exp.Name != tt.expectedNames[i] {
						t.Errorf("expected expert %s at index %d, got %s", tt.expectedNames[i], i, exp.Name)
					}
				}
			}
		})
	}
}

func TestCapabilityMap_GetAllExperts(t *testing.T) {
	cm := NewCapabilityMap()
	cm.BuildFromConfigs([]*agents.ParrotSelfCognition{
		{
			Name:         "memo",
			Emoji:        "📝",
			Title:        "Note Expert",
			Capabilities: []string{"search_notes"},
		},
		{
			Name:         "schedule",
			Emoji:        "📅",
			Title:        "Schedule Expert",
			Capabilities: []string{"create_event"},
		},
	})

	experts := cm.GetAllExperts()
	if len(experts) != 2 {
		t.Errorf("expected 2 experts, got %d", len(experts))
	}

	// Verify it returns a new slice each time (not the same slice reference)
	experts2 := cm.GetAllExperts()
	if &experts[0] == &experts2[0] {
		t.Error("GetAllExperts should return a new slice, not the same slice reference")
	}
}

func TestCapabilityMap_normalizeCapability(t *testing.T) {
	cm := NewCapabilityMap()

	tests := []struct {
		input    string
		expected string
	}{
		{"SEARCH_NOTES", "search_notes"},
		{"  search_notes  ", "search_notes"},
		{"Search_Notes", "search_notes"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		result := cm.normalizeCapability(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeCapability(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
