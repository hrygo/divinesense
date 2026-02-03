package tags

import (
	"context"
	"testing"
)

func TestRulesLayer_TechTerms(t *testing.T) {
	layer := NewRulesLayer()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "React detection",
			content:  "今天学习了 React Hooks 的使用方法",
			expected: []string{"React"},
		},
		{
			name:     "Multiple tech terms",
			content:  "使用 Docker 部署 Go 应用到 Kubernetes",
			expected: []string{"Docker", "Go", "Kubernetes"},
		},
		{
			name:     "Python and AI",
			content:  "Python 机器学习入门教程",
			expected: []string{"Python", "机器学习"},
		},
		{
			name:     "No tech terms",
			content:  "今天天气很好",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := layer.Suggest(context.Background(), &SuggestRequest{
				Content: tt.content,
			})

			found := make(map[string]bool)
			for _, s := range suggestions {
				if s.Source == "rules" {
					found[s.Name] = true
				}
			}

			for _, exp := range tt.expected {
				if !found[exp] {
					t.Errorf("expected tag %q not found in suggestions", exp)
				}
			}
		})
	}
}

func TestRulesLayer_EmotionTerms(t *testing.T) {
	layer := NewRulesLayer()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "TODO detection",
			content:  "TODO: 完成报告",
			expected: "待办",
		},
		{
			name:     "灵感 detection",
			content:  "突然有个灵感",
			expected: "灵感",
		},
		{
			name:     "学习 detection",
			content:  "今天学习了新知识",
			expected: "学习",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := layer.Suggest(context.Background(), &SuggestRequest{
				Content: tt.content,
			})

			found := false
			for _, s := range suggestions {
				if s.Name == tt.expected {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected tag %q not found", tt.expected)
			}
		})
	}
}

func TestRulesLayer_DatePatterns(t *testing.T) {
	layer := NewRulesLayer()

	tests := []struct {
		name    string
		content string
		hasDate bool
	}{
		{
			name:    "Quarter detection",
			content: "Q1 季度总结",
			hasDate: true,
		},
		{
			name:    "Year-month detection",
			content: "2024-01 计划",
			hasDate: true,
		},
		{
			name:    "No date",
			content: "普通笔记内容",
			hasDate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := layer.Suggest(context.Background(), &SuggestRequest{
				Content: tt.content,
			})

			hasDateTag := false
			for _, s := range suggestions {
				if s.Reason == "time marker" {
					hasDateTag = true
					break
				}
			}

			if hasDateTag != tt.hasDate {
				t.Errorf("hasDate = %v, want %v", hasDateTag, tt.hasDate)
			}
		})
	}
}

func TestLLMLayer_ParseTagsFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected []string
	}{
		{
			name:     "Valid JSON array",
			response: `["技术", "学习", "Go"]`,
			expected: []string{"技术", "学习", "Go"},
		},
		{
			name:     "JSON with markdown",
			response: "```json\n[\"React\", \"前端\"]\n```",
			expected: []string{"React", "前端"},
		},
		{
			name:     "Line format fallback",
			response: "- 技术\n- 学习\n- 日常",
			expected: []string{"技术", "学习", "日常"},
		},
		{
			name:     "Empty response",
			response: "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := parseTagsFromJSON(tt.response)

			if len(tags) != len(tt.expected) {
				t.Errorf("got %d tags, want %d", len(tags), len(tt.expected))
				return
			}

			for i, exp := range tt.expected {
				if tags[i] != exp {
					t.Errorf("tag[%d] = %q, want %q", i, tags[i], exp)
				}
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "English words",
			text:     "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "Chinese characters",
			text:     "你好世界",
			expected: []string{"你好世界"},
		},
		{
			name:     "Mixed content",
			text:     "React 学习笔记",
			expected: []string{"react", "学习笔记"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tokenize(tt.text)

			if len(tokens) != len(tt.expected) {
				t.Errorf("got %d tokens, want %d: %v", len(tokens), len(tt.expected), tokens)
			}
		})
	}
}

func TestNormalizeFrequency(t *testing.T) {
	tests := []struct {
		count    int
		expected float64
	}{
		{count: 1, expected: 0.6},
		{count: 2, expected: 0.7},
		{count: 3, expected: 0.8},
		{count: 5, expected: 0.9},
		{count: 10, expected: 1.0},
		{count: 100, expected: 1.0},
	}

	for _, tt := range tests {
		result := normalizeFrequency(tt.count)
		if result != tt.expected {
			t.Errorf("normalizeFrequency(%d) = %v, want %v", tt.count, result, tt.expected)
		}
	}
}
