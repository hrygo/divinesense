package summary

import (
	"testing"
	"unicode/utf8"
)

func TestFallbackSummarize(t *testing.T) {
	tests := []struct {
		name       string
		req        *SummarizeRequest
		wantSource string
		wantLenLE  int // 期望结果长度 <= 这个值，-1 表示不检查
	}{
		{
			name: "first paragraph extraction",
			req: &SummarizeRequest{
				Content: "这是第一段内容。\n这是第二段内容。",
				MaxLen:  100,
			},
			wantSource: "fallback_first_para",
			wantLenLE:  100,
		},
		{
			name: "first sentence extraction - shorter than maxLen",
			req: &SummarizeRequest{
				Content: "这是第一段内容。",
				MaxLen:  100,
			},
			wantSource: "fallback_first_para",
			wantLenLE:  100,
		},
		{
			name: "truncation when content exceeds maxLen",
			req: &SummarizeRequest{
				Content: "这是一段很长的内容需要被截断",
				MaxLen:  10,
			},
			wantSource: "fallback_first_para",
			wantLenLE:  10,
		},
		{
			name: "first sentence truncation",
			req: &SummarizeRequest{
				Content: "这是第一句。这是第二句。",
				MaxLen:  5,
			},
			wantSource: "fallback_first_para",
			wantLenLE:  5,
		},
		{
			name: "default maxLen when zero",
			req: &SummarizeRequest{
				Content: "这是一段测试内容，用于验证默认最大长度。",
				MaxLen:  0,
			},
			wantSource: "fallback_first_para",
			wantLenLE:  200,
		},
		{
			name: "empty content",
			req: &SummarizeRequest{
				Content: "",
				MaxLen:  100,
			},
			wantSource: "fallback_truncate",
			wantLenLE:  100,
		},
		{
			name: "Chinese and English mixed content",
			req: &SummarizeRequest{
				Content: "Hello world! 你好世界。\nSecond paragraph.",
				MaxLen:  50,
			},
			wantSource: "fallback_first_para",
			wantLenLE:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := FallbackSummarize(tt.req)
			if err != nil {
				t.Fatalf("FallbackSummarize() error = %v", err)
			}
			if resp.Source != tt.wantSource {
				t.Errorf("Source = %v, want %v", resp.Source, tt.wantSource)
			}
			if tt.wantLenLE >= 0 {
				summaryLen := utf8.RuneCountInString(resp.Summary)
				if summaryLen > tt.wantLenLE {
					t.Errorf("Summary length = %d, want <= %d", summaryLen, tt.wantLenLE)
				}
			}
		})
	}
}

func TestExtractFirstParagraph(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "normal content",
			content: "第一段\n第二段\n第三段",
			want:    "第一段",
		},
		{
			name:    "leading empty lines",
			content: "\n\n第一段\n第二段",
			want:    "第一段",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "only whitespace",
			content: "   \n   \n内容",
			want:    "内容",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstParagraph(tt.content)
			if got != tt.want {
				t.Errorf("extractFirstParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFirstSentence(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "English period",
			content: "This is the first sentence. This is the second.",
			want:    "This is the first sentence.",
		},
		{
			name:    "Chinese period",
			content: "这是第一句。这是第二句。",
			want:    "这是第一句。",
		},
		{
			name:    "Question mark followed by uppercase",
			content: "Are you ok? Yes, I am.",
			want:    "Are you ok?",
		},
		{
			name:    "Question mark followed by space",
			content: "Are you ok? Yes I am.",
			want:    "Are you ok?",
		},
		{
			name:    "Exclamation mark",
			content: "Wow! Amazing!",
			want:    "Wow!",
		},
		{
			name:    "No end marker",
			content: "This is a sentence without end marker",
			want:    "This is a sentence without end marker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstSentence(tt.content)
			if got != tt.want {
				t.Errorf("extractFirstSentence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "normal truncation",
			s:      "你好世界",
			maxLen: 2,
			want:   "你好",
		},
		{
			name:   "no truncation needed",
			s:      "你好",
			maxLen: 10,
			want:   "你好",
		},
		{
			name:   "zero maxLen",
			s:      "你好",
			maxLen: 0,
			want:   "你好",
		},
		{
			name:   "negative maxLen",
			s:      "你好",
			maxLen: -1,
			want:   "你好",
		},
		{
			name:   "English truncation",
			s:      "Hello World",
			maxLen: 5,
			want:   "Hello",
		},
		{
			name:   "mixed content truncation",
			s:      "你好 World",
			maxLen: 3,
			want:   "你好 ",
		},
		{
			name:   "mixed content truncation 4 chars",
			s:      "你好 World",
			maxLen: 4,
			want:   "你好 W",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateRunes(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateRunes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuneLen(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{
			name: "Chinese characters",
			s:    "你好",
			want: 2,
		},
		{
			name: "English characters",
			s:    "hello",
			want: 5,
		},
		{
			name: "mixed content",
			s:    "你好hello",
			want: 7,
		},
		{
			name: "empty string",
			s:    "",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runeLen(tt.s)
			if got != tt.want {
				t.Errorf("runeLen() = %d, want %d", got, tt.want)
			}
		})
	}
}
