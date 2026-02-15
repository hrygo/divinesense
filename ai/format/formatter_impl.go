package format

import (
	"context"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
)

type llmFormatter struct {
	llm     llm.Service
	timeout time.Duration
}

func NewFormatter(llmSvc llm.Service) Formatter {
	return &llmFormatter{
		llm:     llmSvc,
		timeout: 10 * time.Second,
	}
}

func (f *llmFormatter) Format(ctx context.Context, req *FormatRequest) (*FormatResponse, error) {
	start := time.Now()

	// 1. 已经是合格 Markdown 的短文本，直接跳过
	if isWellFormatted(req.Content) {
		return &FormatResponse{
			Formatted: req.Content,
			Changed:   false,
			Source:    "passthrough",
			Latency:   time.Since(start),
		}, nil
	}

	// 2. LLM 不可用时直接放行
	if f.llm == nil {
		return &FormatResponse{
			Formatted: req.Content,
			Changed:   false,
			Source:    "passthrough",
			Latency:   time.Since(start),
		}, nil
	}

	// 3. 调用 LLM 格式化
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	userPrompt := "请将以下内容整理为标准 Markdown 格式：\n\n" + req.Content

	messages := []llm.Message{
		llm.SystemPrompt(formatSystemPrompt),
		llm.UserMessage(userPrompt),
	}

	content, _, err := f.llm.Chat(ctx, messages)
	if err != nil {
		return &FormatResponse{
			Formatted: req.Content,
			Changed:   false,
			Source:    "passthrough",
			Latency:   time.Since(start),
		}, nil
	}

	formatted := parseFormattedContent(content)
	return &FormatResponse{
		Formatted: formatted,
		Changed:   formatted != req.Content,
		Source:    "llm",
		Latency:   time.Since(start),
	}, nil
}

func isWellFormatted(content string) bool {
	if len(content) < 50 {
		return false
	}
	lines := strings.Split(content, "\n")
	mdMarkers := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, "1. ") {
			mdMarkers++
		}
	}
	return mdMarkers >= 2
}

func parseFormattedContent(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```markdown")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	return content
}

const formatSystemPrompt = `你是一个笔记格式化助手。将用户随意输入的内容整理为结构清晰的 Markdown 格式。

规则：
1. 保持原文含义完全不变，不添加、不删除任何信息
2. 合理使用 Markdown 标记：标题(#)、列表(-)、加粗(**)、代码块(''')
3. 如果内容包含多个主题，使用标题分隔
4. 如果内容是清单/列表形式，转为 Markdown 列表
5. 如果内容已经格式良好，原样返回
6. 不要添加额外的标题或总结
7. 直接返回格式化后的 Markdown，不要包裹在 JSON 或代码块中
8. 如果原文是英文，使用英文标点；如果是中文，使用中文标点`
