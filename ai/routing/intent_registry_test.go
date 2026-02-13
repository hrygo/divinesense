package routing

import (
	"regexp"
	"testing"
)

func TestIntentRegistry_Match(t *testing.T) {
	// Create a fresh registry with test configs
	registry := NewIntentRegistry()
	registry.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建", "添加", "安排", "预约"},
		Patterns:  []*regexp.Regexp{regexp.MustCompile(`(?i)(今天|明天|下周)\s*(下午|上午)?\d{1,2}点`)},
		Priority:  100,
		RouteType: "schedule",
	})
	registry.Register(IntentConfig{
		Intent:    IntentMemoCreate,
		AgentType: AgentTypeMemo,
		Keywords:  []string{"记录", "写笔记", "帮我记"},
		Priority:  90,
		RouteType: "memo",
	})
	registry.Register(IntentConfig{
		Intent:    IntentBatchSchedule,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"每天", "每周", "批量"},
		Priority:  110, // Higher priority
		RouteType: "schedule",
	})

	tests := []struct {
		name        string
		input       string
		wantIntent  Intent
		wantConfMin float32 // minimum expected confidence
		wantMatched bool
	}{
		// Regex pattern matches
		{name: "regex match - today afternoon", input: "今天下午3点开会", wantIntent: IntentScheduleCreate, wantConfMin: 0.8, wantMatched: true},
		{name: "regex match - tomorrow morning", input: "明天上午10点面试", wantIntent: IntentScheduleCreate, wantConfMin: 0.8, wantMatched: true},

		// Keyword matches
		{name: "keyword match - create schedule", input: "帮我创建一个日程", wantIntent: IntentScheduleCreate, wantConfMin: 0.5, wantMatched: true},
		{name: "keyword match - add memo", input: "帮我记一下这个想法", wantIntent: IntentMemoCreate, wantConfMin: 0.5, wantMatched: true},
		{name: "keyword match - batch schedule", input: "每天早上8点提醒我", wantIntent: IntentBatchSchedule, wantConfMin: 0.5, wantMatched: true},

		// Priority order - batch should win over create
		{name: "priority order - batch over create", input: "批量添加日程", wantIntent: IntentBatchSchedule, wantConfMin: 0.5, wantMatched: true},

		// Edge cases
		{name: "empty input", input: "", wantIntent: IntentUnknown, wantConfMin: 0, wantMatched: false},
		{name: "unknown input", input: "随便说说", wantIntent: IntentUnknown, wantConfMin: 0, wantMatched: false},
		{name: "whitespace only", input: "   ", wantIntent: IntentUnknown, wantConfMin: 0, wantMatched: false},

		// Case insensitivity - keywords are lowercased
		{name: "case insensitivity works", input: "帮我记一下", wantIntent: IntentMemoCreate, wantConfMin: 0.5, wantMatched: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIntent, gotConf, gotMatched := registry.Match(tt.input)

			if gotMatched != tt.wantMatched {
				t.Errorf("Match() matched = %v, want %v", gotMatched, tt.wantMatched)
			}

			if gotIntent != tt.wantIntent {
				t.Errorf("Match() intent = %v, want %v", gotIntent, tt.wantIntent)
			}

			if gotConf < tt.wantConfMin {
				t.Errorf("Match() confidence = %v, want at least %v", gotConf, tt.wantConfMin)
			}
		})
	}
}

func TestIntentRegistry_GetAgentType(t *testing.T) {
	registry := NewIntentRegistry()
	registry.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建"},
		Priority:  100,
		RouteType: "schedule",
	})
	registry.Register(IntentConfig{
		Intent:    IntentMemoSearch,
		AgentType: AgentTypeMemo,
		Keywords:  []string{"搜索"},
		Priority:  100,
		RouteType: "memo",
	})

	tests := []struct {
		name       string
		intent     Intent
		wantAgent  AgentType
		wantExists bool
	}{
		{name: "schedule intent", intent: IntentScheduleCreate, wantAgent: AgentTypeSchedule, wantExists: true},
		{name: "memo intent", intent: IntentMemoSearch, wantAgent: AgentTypeMemo, wantExists: true},
		// Note: IntentUnknown is not registered, so it returns empty string (zero value)
		{name: "unknown intent", intent: IntentUnknown, wantAgent: AgentType(""), wantExists: false},
		{name: "non-existent intent", intent: Intent("nonexistent"), wantAgent: AgentType(""), wantExists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAgent, gotExists := registry.GetAgentType(tt.intent)

			if gotExists != tt.wantExists {
				t.Errorf("GetAgentType() exists = %v, want %v", gotExists, tt.wantExists)
			}

			if gotAgent != tt.wantAgent {
				t.Errorf("GetAgentType() agent = %v, want %v", gotAgent, tt.wantAgent)
			}
		})
	}
}

func TestIntentRegistry_GetRouteType(t *testing.T) {
	registry := NewIntentRegistry()
	registry.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建"},
		Priority:  100,
		RouteType: "schedule",
	})

	tests := []struct {
		name       string
		intent     Intent
		wantRoute  string
		wantExists bool
	}{
		{name: "existing intent", intent: IntentScheduleCreate, wantRoute: "schedule", wantExists: true},
		{name: "non-existent intent", intent: Intent("nonexistent"), wantRoute: "", wantExists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoute, gotExists := registry.GetRouteType(tt.intent)

			if gotExists != tt.wantExists {
				t.Errorf("GetRouteType() exists = %v, want %v", gotExists, tt.wantExists)
			}

			if gotRoute != tt.wantRoute {
				t.Errorf("GetRouteType() route = %v, want %v", gotRoute, tt.wantRoute)
			}
		})
	}
}

func TestIntentRegistry_GetIntent(t *testing.T) {
	registry := NewIntentRegistry()
	registry.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建"},
		Priority:  100,
		RouteType: "schedule",
	})

	tests := []struct {
		name       string
		agentType  AgentType
		wantIntent Intent
		wantExists bool
	}{
		{name: "existing agent", agentType: AgentTypeSchedule, wantIntent: IntentScheduleCreate, wantExists: true},
		// Note: AgentTypeUnknown is not registered, returns empty string (zero value)
		{name: "unknown agent", agentType: AgentTypeUnknown, wantIntent: Intent(""), wantExists: false},
		{name: "non-existent agent", agentType: AgentType("nonexistent"), wantIntent: Intent(""), wantExists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIntent, gotExists := registry.GetIntent(tt.agentType)

			if gotExists != tt.wantExists {
				t.Errorf("GetIntent() exists = %v, want %v", gotExists, tt.wantExists)
			}

			if gotIntent != tt.wantIntent {
				t.Errorf("GetIntent() intent = %v, want %v", gotIntent, tt.wantIntent)
			}
		})
	}
}

func TestIntentRegistry_Register_UpdatesExisting(t *testing.T) {
	registry := NewIntentRegistry()

	// Register initial config
	registry.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建"},
		Priority:  100,
		RouteType: "schedule",
	})

	// Update with new keywords
	registry.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建", "添加", "新建"},
		Priority:  100,
		RouteType: "schedule_v2",
	})

	// Verify update
	intent, conf, matched := registry.Match("新建日程")
	if !matched || intent != IntentScheduleCreate {
		t.Errorf("Register() update failed: matched=%v, intent=%v", matched, intent)
	}
	if conf < 0.5 {
		t.Errorf("Register() confidence too low: %v", conf)
	}

	// Verify route type updated
	route, exists := registry.GetRouteType(IntentScheduleCreate)
	if !exists || route != "schedule_v2" {
		t.Errorf("Register() route type not updated: exists=%v, route=%v", exists, route)
	}
}

func TestIntentRegistry_DefaultRegistry(t *testing.T) {
	// Verify default registry is initialized
	registry := DefaultRegistry()
	if registry == nil {
		t.Fatal("DefaultRegistry() returned nil")
	}

	// Verify some default intents are registered
	tests := []struct {
		intent     Intent
		wantAgent  AgentType
		wantExists bool
	}{
		{IntentScheduleCreate, AgentTypeSchedule, true},
		{IntentMemoSearch, AgentTypeMemo, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.intent), func(t *testing.T) {
			gotAgent, gotExists := registry.GetAgentType(tt.intent)
			if gotExists != tt.wantExists {
				t.Errorf("GetAgentType() exists = %v, want %v", gotExists, tt.wantExists)
			}
			if gotAgent != tt.wantAgent {
				t.Errorf("GetAgentType() agent = %v, want %v", gotAgent, tt.wantAgent)
			}
		})
	}
}

func TestIntentRegistry_Concurrency(t *testing.T) {
	registry := NewIntentRegistry()

	// Simulate concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			registry.Register(IntentConfig{
				Intent:    Intent("test_intent"),
				AgentType: AgentType("test_agent"),
				Keywords:  []string{"test"},
				Priority:  i,
				RouteType: "test",
			})
		}
		done <- true
	}()

	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				registry.Match("test")
				registry.GetAgentType(Intent("test_intent"))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 6; i++ {
		<-done
	}
}
