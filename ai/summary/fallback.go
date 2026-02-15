package summary

import (
	"strings"
	"unicode/utf8"
)

// FallbackSummarize 提供三级降级摘要
func FallbackSummarize(req *SummarizeRequest) (*SummarizeResponse, error) {
	maxLen := req.MaxLen
	if maxLen <= 0 {
		maxLen = 200
	}

	// Level 1: 首段提取（最优降级）
	if firstPara := extractFirstParagraph(req.Content); firstPara != "" {
		if utf8.RuneCountInString(firstPara) <= maxLen {
			return &SummarizeResponse{
				Summary: firstPara,
				Source:  "fallback_first_para",
			}, nil
		}
		return &SummarizeResponse{
			Summary: truncateRunes(firstPara, maxLen),
			Source:  "fallback_first_para",
		}, nil
	}

	// Level 2: 首句提取
	if firstSentence := extractFirstSentence(req.Content); firstSentence != "" {
		if utf8.RuneCountInString(firstSentence) <= maxLen {
			return &SummarizeResponse{
				Summary: firstSentence,
				Source:  "fallback_first_sentence",
			}, nil
		}
		return &SummarizeResponse{
			Summary: truncateRunes(firstSentence, maxLen),
			Source:  "fallback_first_sentence",
		}, nil
	}

	// Level 3: Rune 安全截断（保底）
	return &SummarizeResponse{
		Summary: truncateRunes(req.Content, maxLen),
		Source:  "fallback_truncate",
	}, nil
}

// extractFirstParagraph 提取第一段
func extractFirstParagraph(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// extractFirstSentence 提取第一句
func extractFirstSentence(content string) string {
	firstLine := extractFirstParagraph(content)
	if firstLine == "" {
		return ""
	}

	// 优先检查英文标点，然后是中文标点
	endMarkers := []string{"?", "!", ".", "？", "！", "。"}
	for _, marker := range endMarkers {
		idx := strings.Index(firstLine, marker)
		if idx < 0 {
			continue
		}
		// 检查标记后是否是句末（空格、换行、字符串结尾或非大写字母）
		nextIdx := idx + len(marker)
		if nextIdx >= len(firstLine) {
			return firstLine[:nextIdx]
		}
		nextChar := firstLine[nextIdx]
		// 如果后面是空格或非大写字母，则认为是句末
		if nextChar == ' ' || nextChar == '\n' || nextChar < 'A' || nextChar > 'Z' {
			return firstLine[:nextIdx]
		}
		// 问号感叹号后面如果是句末标点也算
		if (marker == "?" || marker == "!") && nextIdx < len(firstLine) {
			// 检查后面是否有其他句末标点
			remaining := firstLine[nextIdx:]
			for _, endMark := range []string{".", "。", "?", "？", "!", "！"} {
				if strings.HasPrefix(strings.TrimLeft(remaining, " "), endMark) {
					return firstLine[:nextIdx]
				}
			}
		}
	}
	return firstLine
}

// truncateRunes 安全截断字符串（按 rune 而非 byte）
func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// runeLen 获取字符串的 rune 长度
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
