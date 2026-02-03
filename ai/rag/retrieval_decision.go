// Package rag provides Self-RAG (Retrieval-Augmented Generation) optimization.
// It implements a lightweight, rule-driven approach to reduce unnecessary retrievals.
package rag

import (
	"strings"
)

// RetrievalDecision represents the decision on whether to retrieve.
type RetrievalDecision struct {
	Reason         string
	Confidence     float32
	ShouldRetrieve bool
}

// DecisionReason constants.
const (
	ReasonChitchat         = "chitchat_detected"
	ReasonSystemCommand    = "system_command"
	ReasonRetrievalTrigger = "retrieval_trigger"
	ReasonScheduleQuery    = "schedule_query"
	ReasonDefault          = "default"
)

// Retrieval trigger patterns.
var (
	// Patterns that indicate no retrieval is needed.
	chitchatPatterns = []string{
		"你好", "谢谢", "再见", "哈哈", "好的", "嗯", "ok", "hi", "hello",
		"好", "行", "可以", "没问题", "明白了", "知道了",
	}

	systemCommands = []string{
		"帮助", "设置", "退出", "清空", "重置", "取消",
	}

	// Patterns that indicate retrieval is needed.
	retrievalTriggers = []string{
		"搜索", "查找", "找到", "找", "查", "有什么", "哪些",
		"记录", "笔记", "memo", "之前", "写过", "提到",
	}

	scheduleTriggers = []string{
		"日程", "安排", "会议", "提醒", "约会", "预约",
		"今天", "明天", "后天", "下周", "本周",
	}

	// Question patterns that usually need retrieval.
	questionPatterns = []string{
		"什么时候", "在哪里", "怎么样", "多少", "是不是",
	}
)

// RetrievalDecider makes decisions about whether to retrieve.
type RetrievalDecider struct {
	chitchatPatterns  []string
	systemCommands    []string
	retrievalTriggers []string
	scheduleTriggers  []string
	questionPatterns  []string
}

// NewRetrievalDecider creates a new retrieval decider with default patterns.
func NewRetrievalDecider() *RetrievalDecider {
	return &RetrievalDecider{
		chitchatPatterns:  chitchatPatterns,
		systemCommands:    systemCommands,
		retrievalTriggers: retrievalTriggers,
		scheduleTriggers:  scheduleTriggers,
		questionPatterns:  questionPatterns,
	}
}

// Decide determines whether retrieval is needed for a query.
func (d *RetrievalDecider) Decide(query string) *RetrievalDecision {
	query = strings.TrimSpace(query)
	queryLower := strings.ToLower(query)

	// Rule 1: Very short queries - likely chitchat
	if len([]rune(query)) <= 2 {
		return &RetrievalDecision{
			ShouldRetrieve: false,
			Reason:         ReasonChitchat,
			Confidence:     0.9,
		}
	}

	// Rule 2: Chitchat patterns - no retrieval
	for _, pattern := range d.chitchatPatterns {
		if strings.HasPrefix(queryLower, pattern) || query == pattern {
			return &RetrievalDecision{
				ShouldRetrieve: false,
				Reason:         ReasonChitchat,
				Confidence:     0.95,
			}
		}
	}

	// Rule 3: System commands - no retrieval
	for _, cmd := range d.systemCommands {
		if strings.Contains(queryLower, cmd) && len([]rune(query)) < 10 {
			return &RetrievalDecision{
				ShouldRetrieve: false,
				Reason:         ReasonSystemCommand,
				Confidence:     0.9,
			}
		}
	}

	// Rule 4: Schedule triggers - retrieval needed
	for _, trigger := range d.scheduleTriggers {
		if strings.Contains(queryLower, trigger) {
			return &RetrievalDecision{
				ShouldRetrieve: true,
				Reason:         ReasonScheduleQuery,
				Confidence:     0.85,
			}
		}
	}

	// Rule 5: Explicit retrieval triggers - retrieval needed
	for _, trigger := range d.retrievalTriggers {
		if strings.Contains(queryLower, trigger) {
			return &RetrievalDecision{
				ShouldRetrieve: true,
				Reason:         ReasonRetrievalTrigger,
				Confidence:     0.9,
			}
		}
	}

	// Rule 6: Question patterns - usually need retrieval
	for _, pattern := range d.questionPatterns {
		if strings.Contains(queryLower, pattern) {
			return &RetrievalDecision{
				ShouldRetrieve: true,
				Reason:         ReasonDefault,
				Confidence:     0.7,
			}
		}
	}

	// Rule 7: Default - retrieve for longer queries
	if len([]rune(query)) > 5 {
		return &RetrievalDecision{
			ShouldRetrieve: true,
			Reason:         ReasonDefault,
			Confidence:     0.6,
		}
	}

	return &RetrievalDecision{
		ShouldRetrieve: false,
		Reason:         ReasonChitchat,
		Confidence:     0.5,
	}
}

// DecideRetrieval is a convenience function for quick decisions.
func DecideRetrieval(query string) *RetrievalDecision {
	return NewRetrievalDecider().Decide(query)
}
