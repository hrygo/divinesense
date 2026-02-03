package tags

import (
	"context"
	"regexp"
	"strings"
)

// RulesLayer provides tag suggestions based on pattern matching.
// Layer 2: ~10ms latency, uses tech terms, emotion words, and date patterns.
type RulesLayer struct {
	techTerms    []string
	emotionTerms map[string]string
	datePatterns []*regexp.Regexp
}

// NewRulesLayer creates a new rules layer with predefined patterns.
func NewRulesLayer() *RulesLayer {
	return &RulesLayer{
		techTerms: []string{
			// Programming languages
			"Go", "Golang", "Python", "Java", "JavaScript", "TypeScript",
			"Rust", "C++", "Swift", "Kotlin", "Ruby", "PHP",
			// Frameworks
			"React", "Vue", "Angular", "Next.js", "Node.js", "Django",
			"Flask", "Spring", "Rails", "Gin", "Echo", "Fiber",
			// Databases
			"PostgreSQL", "MySQL", "MongoDB", "Redis", "SQLite",
			// Cloud & DevOps
			"Docker", "Kubernetes", "AWS", "GCP", "Azure",
			"CI/CD", "DevOps", "GitHub", "GitLab",
			// AI/ML
			"AI", "ML", "机器学习", "深度学习", "LLM", "GPT", "Claude",
			"TensorFlow", "PyTorch", "Transformer",
			// Others
			"API", "REST", "GraphQL", "gRPC", "WebSocket",
			"Linux", "macOS", "Windows",
		},
		emotionTerms: map[string]string{
			// Chinese emotion/status words
			"灵感": "灵感",
			"想法": "想法",
			"问题": "问题",
			"待办": "待办",
			"记录": "记录",
			"学习": "学习",
			"复习": "学习",
			"笔记": "笔记",
			"总结": "总结",
			"反思": "反思",
			"计划": "计划",
			"目标": "目标",
			"进度": "进度",
			"完成": "完成",
			"参考": "参考",
			"备忘": "备忘",
			"重要": "重要",
			"紧急": "紧急",
			// English equivalents
			"TODO":      "待办",
			"todo":      "待办",
			"FIXME":     "问题",
			"BUG":       "问题",
			"bug":       "问题",
			"idea":      "想法",
			"note":      "笔记",
			"meeting":   "会议",
			"review":    "复盘",
			"daily":     "日常",
			"weekly":    "周报",
			"monthly":   "月报",
			"project":   "项目",
			"research":  "调研",
			"reference": "参考",
		},
		datePatterns: []*regexp.Regexp{
			regexp.MustCompile(`20\d{2}[-/年]?\d{1,2}[-/月]?`), // 2024-01, 2024年1月
			regexp.MustCompile(`Q[1-4]`),                     // Q1, Q2, Q3, Q4
			regexp.MustCompile(`第[一二三四1234]季度`),              // 第一季度
			regexp.MustCompile(`(周一|周二|周三|周四|周五|周六|周日)`),     // Weekdays
			regexp.MustCompile(`(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)`),
		},
	}
}

// Name returns the layer name.
func (l *RulesLayer) Name() string {
	return "rules"
}

// Suggest returns tag suggestions based on pattern matching.
func (l *RulesLayer) Suggest(ctx context.Context, req *SuggestRequest) []Suggestion {
	var suggestions []Suggestion
	seen := make(map[string]bool)

	text := req.Title + " " + req.Content
	textLower := strings.ToLower(text)

	// 1. Tech terms recognition
	for _, term := range l.techTerms {
		termLower := strings.ToLower(term)
		if strings.Contains(textLower, termLower) {
			if !seen[termLower] {
				seen[termLower] = true
				suggestions = append(suggestions, Suggestion{
					Name:       term,
					Confidence: 0.9,
					Source:     "rules",
					Reason:     "tech term",
				})
			}
		}
	}

	// 2. Emotion/status word recognition
	for keyword, tag := range l.emotionTerms {
		if strings.Contains(text, keyword) {
			tagLower := strings.ToLower(tag)
			if !seen[tagLower] {
				seen[tagLower] = true
				suggestions = append(suggestions, Suggestion{
					Name:       tag,
					Confidence: 0.85,
					Source:     "rules",
					Reason:     "status/emotion",
				})
			}
		}
	}

	// 3. Date pattern extraction
	for _, pattern := range l.datePatterns {
		if matches := pattern.FindAllString(text, 3); len(matches) > 0 {
			for _, match := range matches {
				matchLower := strings.ToLower(match)
				if !seen[matchLower] {
					seen[matchLower] = true
					suggestions = append(suggestions, Suggestion{
						Name:       match,
						Confidence: 0.8,
						Source:     "rules",
						Reason:     "time marker",
					})
				}
			}
		}
	}

	return suggestions
}
