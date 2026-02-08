package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusExporter(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	t.Run("RecordChatRequest", func(t *testing.T) {
		exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
		exporter.RecordChatRequest("memo", "normal", 200*time.Millisecond, true)
		exporter.RecordChatRequest("schedule", "normal", 150*time.Millisecond, false)

		exporter.SetActiveChats(5)
	})

	t.Run("RecordToolCall", func(t *testing.T) {
		exporter.RecordToolCall("memo_search", 50*time.Millisecond, true, "")
		exporter.RecordToolCall("schedule_add", 100*time.Millisecond, false, "timeout")
	})

	t.Run("RecordCache", func(t *testing.T) {
		exporter.RecordCacheHit("intent")
		exporter.RecordCacheHit("intent")
		exporter.RecordCacheMiss("retrieval")
	})

	t.Run("RecordLLM", func(t *testing.T) {
		exporter.RecordLLMTokens("deepseek-chat", "prompt", 100)
		exporter.RecordLLMTokens("deepseek-chat", "completion", 50)
		exporter.RecordLLMCachedTokens("deepseek-chat", 80)
		exporter.RecordLLMLatency("deepseek-chat", "deepseek", 500*time.Millisecond)
	})

	t.Run("RecordAgentError", func(t *testing.T) {
		exporter.RecordAgentError("memo", "timeout")
		exporter.RecordAgentError("schedule", "invalid_input")
	})

	t.Run("SetAgentSuccessRate", func(t *testing.T) {
		exporter.SetAgentSuccessRate("memo", 0.95)
		exporter.SetAgentSuccessRate("schedule", 0.98)
	})
}

func TestPrometheusExporterHandler(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	// Record some metrics
	exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
	exporter.RecordToolCall("memo_search", 50*time.Millisecond, true, "")
	exporter.RecordCacheHit("intent")
	exporter.RecordLLMTokens("deepseek-chat", "prompt", 100)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	w := httptest.NewRecorder()

	exporter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "divinesense_ai_chat_requests_total") {
		t.Error("expected chat_requests_total metric in output")
	}
	if !strings.Contains(body, "divinesense_ai_tool_calls_total") {
		t.Error("expected tool_calls_total metric in output")
	}
	if !strings.Contains(body, "divinesense_ai_cache_hits_total") {
		t.Error("expected cache_hits_total metric in output")
	}
	if !strings.Contains(body, "divinesense_ai_llm_tokens_total") {
		t.Error("expected llm_tokens_total metric in output")
	}
}

func TestPrometheusExporterExportText(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	// Record some metrics
	exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
	exporter.RecordCacheHit("intent")
	exporter.RecordLLMTokens("deepseek-chat", "prompt", 100)

	output, err := exporter.ExportText()
	if err != nil {
		t.Fatalf("ExportText failed: %v", err)
	}

	if !strings.Contains(output, "# HELP") {
		t.Error("expected HELP comment in output")
	}
	if !strings.Contains(output, "# TYPE") {
		t.Error("expected TYPE comment in output")
	}
}

func TestPrometheusExporterCustomRegistry(t *testing.T) {
	customReg := NewPrometheusExporter(Config{})
	customReg.RecordChatRequest("test", "mode", 50*time.Millisecond, true)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	w := httptest.NewRecorder()

	customReg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func BenchmarkPrometheusExporter(b *testing.B) {
	exporter := NewPrometheusExporter(DefaultConfig())

	b.Run("RecordChatRequest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
		}
	})

	b.Run("RecordToolCall", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			exporter.RecordToolCall("memo_search", 50*time.Millisecond, true, "")
		}
	})

	b.Run("RecordCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			exporter.RecordCacheHit("intent")
		}
	})
}

// Additional tests

func TestPrometheusExporter_RecordCacheMiss(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.RecordCacheMiss("intent")
	exporter.RecordCacheMiss("retrieval")
	exporter.RecordCacheMiss("intent")

	// Verify metrics are recorded
	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "cache_misses_total")
}

func TestPrometheusExporter_RecordLLMCachedTokens(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.RecordLLMCachedTokens("deepseek-chat", 100)
	exporter.RecordLLMCachedTokens("deepseek-chat", 50)

	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "llm_tokens_cached_total")
}

func TestPrometheusExporter_RecordLLMLatency(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.RecordLLMLatency("deepseek-chat", "deepseek", 500*time.Millisecond)
	exporter.RecordLLMLatency("qwen", "siliconflow", 200*time.Millisecond)

	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "llm_latency_seconds")
}

func TestPrometheusExporter_SetActiveChats(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.SetActiveChats(5)
	exporter.SetActiveChats(10)

	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "chat_active")
}

func TestPrometheusExporter_GetHandler(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	handler := exporter.GetHandler()
	assert.NotNil(t, handler)
}

func TestPrometheusExporter_Handler(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	handler := exporter.Handler()
	assert.NotNil(t, handler)
}

func TestPrometheusExporter_RegisterHandler(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	exporter.RegisterHandler("/custom", customHandler)

	// Should not panic
}

func TestPrometheusExporter_GetRegistry(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	registry := exporter.GetRegistry()
	assert.NotNil(t, registry)
}

func TestPrometheusExporter_Snapshot(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
	exporter.RecordToolCall("memo_search", 50*time.Millisecond, true, "")

	snapshot := exporter.Snapshot()

	assert.NotNil(t, snapshot)
	assert.Contains(t, snapshot, "timestamp")
	assert.Contains(t, snapshot, "registry")
}

func TestPrometheusExporter_Close(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	err := exporter.Close()
	assert.NoError(t, err)
}

func TestPrometheusExporter_Config_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotEmpty(t, cfg.LatencyBuckets)
	assert.Nil(t, cfg.Registry)
}

func TestPrometheusExporter_NewWithCustomRegistry(t *testing.T) {
	customReg := NewPrometheusExporter(Config{
		Registry:       nil,
		LatencyBuckets: []float64{0.1, 0.5, 1.0},
	})

	assert.NotNil(t, customReg)
	assert.NotNil(t, customReg.GetRegistry())
}

func TestPrometheusExporter_RecordToolCallWithError(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.RecordToolCall("memo_search", 50*time.Millisecond, false, "timeout")
	exporter.RecordToolCall("schedule_add", 100*time.Millisecond, false, "validation")

	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "tool_errors_total")
}

func TestPrometheusExporter_RecordAgentError(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.RecordAgentError("memo", "timeout")
	exporter.RecordAgentError("schedule", "validation")

	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "agent_errors_total")
}

func TestPrometheusExporter_SetAgentSuccessRate(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	exporter.SetAgentSuccessRate("memo", 0.95)
	exporter.SetAgentSuccessRate("schedule", 0.98)
	exporter.SetAgentSuccessRate("amazing", 0.99)

	output, err := exporter.ExportText()
	require.NoError(t, err)
	assert.Contains(t, output, "agent_success_rate")
}

func TestPrometheusExporter_AllMetricTypes(t *testing.T) {
	exporter := NewPrometheusExporter(DefaultConfig())

	// Test all metric types
	exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
	exporter.RecordToolCall("memo_search", 50*time.Millisecond, true, "")
	exporter.RecordCacheHit("intent")
	exporter.RecordCacheMiss("retrieval")
	exporter.RecordLLMTokens("deepseek-chat", "prompt", 100)
	exporter.RecordLLMCachedTokens("deepseek-chat", 50)
	exporter.RecordLLMLatency("deepseek-chat", "deepseek", 500*time.Millisecond)
	exporter.SetActiveChats(5)
	exporter.RecordAgentError("memo", "timeout")
	exporter.SetAgentSuccessRate("memo", 0.95)

	output, err := exporter.ExportText()
	require.NoError(t, err)

	// Verify all metric types are present
	assert.Contains(t, output, "chat_requests_total")
	assert.Contains(t, output, "chat_latency_seconds")
	assert.Contains(t, output, "tool_calls_total")
	assert.Contains(t, output, "tool_latency_seconds")
	assert.Contains(t, output, "cache_hits_total")
	assert.Contains(t, output, "cache_misses_total")
	assert.Contains(t, output, "llm_tokens_total")
	assert.Contains(t, output, "llm_tokens_cached_total")
	assert.Contains(t, output, "llm_latency_seconds")
	assert.Contains(t, output, "chat_active")
	assert.Contains(t, output, "agent_errors_total")
	assert.Contains(t, output, "agent_success_rate")
}

func BenchmarkPrometheusExporter_ExportText(b *testing.B) {
	exporter := NewPrometheusExporter(DefaultConfig())

	// Record some metrics
	for i := 0; i < 100; i++ {
		exporter.RecordChatRequest("memo", "normal", time.Duration(i)*time.Millisecond, true)
		exporter.RecordToolCall("search", 50*time.Millisecond, true, "")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exporter.ExportText()
	}
}

func BenchmarkPrometheusExporter_Snapshot(b *testing.B) {
	exporter := NewPrometheusExporter(DefaultConfig())

	for i := 0; i < 100; i++ {
		exporter.RecordChatRequest("memo", "normal", 100*time.Millisecond, true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = exporter.Snapshot()
	}
}
