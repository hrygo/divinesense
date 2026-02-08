package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
