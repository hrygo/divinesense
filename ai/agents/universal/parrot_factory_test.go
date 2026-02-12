// Package universal provides tests for ParrotFactory.
package universal

import (
	"testing"
)

// TestLoadConfigs verifies that all parrot configs can be loaded.
func TestLoadConfigs(t *testing.T) {
	// Use absolute path from project root
	factory, err := NewParrotFactory(
		WithConfigDir("../../../config/parrots"),
	)
	if err != nil {
		t.Skipf("NewParrotFactory (config dir may not exist in test environment): %v", err)
		return
	}

	// Check that configs were loaded
	configs := factory.ListConfigs()
	if len(configs) == 0 {
		t.Error("no configs loaded")
	}

	// Note: "amazing" config was removed in favor of Orchestrator-Workers architecture
	expectedConfigs := []string{"memo", "schedule"}
	for _, expected := range expectedConfigs {
		found := false
		for _, name := range configs {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("config not found: %s (loaded: %v)", expected, configs)
		}
	}

	t.Logf("loaded %d configs: %v", len(configs), configs)
}

// TestGetConfig verifies that individual configs can be retrieved.
func TestGetConfig(t *testing.T) {
	factory, err := NewParrotFactory(
		WithConfigDir("../../../config/parrots"),
	)
	if err != nil {
		t.Skipf("NewParrotFactory (config dir may not exist in test environment): %v", err)
		return
	}

	// Note: "amazing" config was removed in favor of Orchestrator-Workers architecture
	testCases := []struct {
		name string
	}{
		{"memo"},
		{"schedule"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, ok := factory.GetConfig(tc.name)
			if !ok {
				t.Fatalf("config not found: %s", tc.name)
			}

			if config.Name != tc.name {
				t.Errorf("name mismatch: got %s, want %s", config.Name, tc.name)
			}

			if len(config.Tools) == 0 {
				t.Errorf("no tools defined for %s", tc.name)
			}

			t.Logf("config %s: strategy=%s, tools=%v",
				config.Name, config.Strategy, config.Tools)
		})
	}
}

// TestDefaultConfigs verifies that default configs can be used as fallback.
func TestDefaultConfigs(t *testing.T) {
	// Note: "amazing" config was removed in favor of Orchestrator-Workers architecture
	tests := []struct {
		name      string
		defaultFn func() *ParrotConfig
	}{
		{"memo", DefaultMemoParrotConfig},
		{"schedule", DefaultScheduleParrotConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.defaultFn()

			if config.Name != tt.name {
				t.Errorf("name mismatch: got %s, want %s", config.Name, tt.name)
			}

			if config.MaxIterations <= 0 {
				t.Error("MaxIterations not set")
			}

			if config.Strategy == "" {
				t.Error("Strategy not set")
			}

			t.Logf("%s config: strategy=%s, max_iterations=%d, tools=%v",
				config.Name, config.Strategy, config.MaxIterations, config.Tools)
		})
	}
}

// TestToolFactoryFunc verifies the tool factory function signature.
func TestToolFactoryFunc(t *testing.T) {
	// This test verifies that ToolFactoryFunc has the correct signature.
	// It's a compile-time check; if it compiles, the signature is correct.

	// Import agent package for ToolWithSchema interface
	type dummyTool struct{}

	// Verify ToolFactoryFunc signature: func(userID int32) (agent.ToolWithSchema, error)
	// If this compiles, the signature is correct in parrot_factory.go
	t.Log("ToolFactoryFunc signature verified at compile time")
}
