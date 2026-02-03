package duplicate

import (
	"testing"
	"time"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 4},
			expected: 0.9914,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "different length",
			a:        []float32{1, 2},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if diff := result - tt.expected; diff > 0.01 || diff < -0.01 {
				t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagCoOccurrence(t *testing.T) {
	tests := []struct {
		name     string
		tags1    []string
		tags2    []string
		expected float64
	}{
		{
			name:     "identical tags",
			tags1:    []string{"go", "react"},
			tags2:    []string{"go", "react"},
			expected: 1.0,
		},
		{
			name:     "no overlap",
			tags1:    []string{"go", "python"},
			tags2:    []string{"react", "vue"},
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			tags1:    []string{"go", "react"},
			tags2:    []string{"react", "vue"},
			expected: 0.333, // 1 / 3
		},
		{
			name:     "case insensitive",
			tags1:    []string{"Go", "React"},
			tags2:    []string{"go", "react"},
			expected: 1.0,
		},
		{
			name:     "both empty",
			tags1:    []string{},
			tags2:    []string{},
			expected: 0.0,
		},
		{
			name:     "one empty",
			tags1:    []string{"go"},
			tags2:    []string{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TagCoOccurrence(tt.tags1, tt.tags2)
			if diff := result - tt.expected; diff > 0.01 || diff < -0.01 {
				t.Errorf("TagCoOccurrence() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTimeProximity(t *testing.T) {
	now := time.Now()
	tests := []struct {
		candidateTime time.Time
		name          string
		minExpected   float64
		maxExpected   float64
	}{
		{
			name:          "same day",
			candidateTime: now,
			minExpected:   0.99,
			maxExpected:   1.01,
		},
		{
			name:          "1 day ago",
			candidateTime: now.Add(-24 * time.Hour),
			minExpected:   0.86,
			maxExpected:   0.88,
		},
		{
			name:          "7 days ago",
			candidateTime: now.Add(-7 * 24 * time.Hour),
			minExpected:   0.36,
			maxExpected:   0.38,
		},
		{
			name:          "14 days ago",
			candidateTime: now.Add(-14 * 24 * time.Hour),
			minExpected:   0.13,
			maxExpected:   0.15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeProximity(now, tt.candidateTime)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("TimeProximity() = %v, want between %v and %v", result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestFindSharedTags(t *testing.T) {
	tests := []struct {
		name     string
		tags1    []string
		tags2    []string
		expected int
	}{
		{
			name:     "all shared",
			tags1:    []string{"go", "react"},
			tags2:    []string{"go", "react"},
			expected: 2,
		},
		{
			name:     "partial shared",
			tags1:    []string{"go", "react"},
			tags2:    []string{"react", "vue"},
			expected: 1,
		},
		{
			name:     "none shared",
			tags1:    []string{"go", "python"},
			tags2:    []string{"react", "vue"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindSharedTags(tt.tags1, tt.tags2)
			if len(result) != tt.expected {
				t.Errorf("FindSharedTags() returned %d tags, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestCalculateWeightedSimilarity(t *testing.T) {
	tests := []struct {
		breakdown *Breakdown
		name      string
		weights   Weights
		expected  float64
	}{
		{
			name: "default weights",
			breakdown: &Breakdown{
				Vector:     0.9,
				TagCoOccur: 0.8,
				TimeProx:   0.7,
			},
			weights:  DefaultWeights,
			expected: 0.9*0.5 + 0.8*0.3 + 0.7*0.2, // 0.45 + 0.24 + 0.14 = 0.83
		},
		{
			name: "all zeros",
			breakdown: &Breakdown{
				Vector:     0,
				TagCoOccur: 0,
				TimeProx:   0,
			},
			weights:  DefaultWeights,
			expected: 0,
		},
		{
			name: "all ones",
			breakdown: &Breakdown{
				Vector:     1.0,
				TagCoOccur: 1.0,
				TimeProx:   1.0,
			},
			weights:  DefaultWeights,
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateWeightedSimilarity(tt.breakdown, tt.weights)
			if diff := result - tt.expected; diff > 0.001 || diff < -0.001 {
				t.Errorf("CalculateWeightedSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		maxLen   int
	}{
		{
			name:     "short content",
			content:  "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			content:  "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncated",
			content:  "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "unicode content",
			content:  "你好世界",
			maxLen:   2,
			expected: "你好...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.content, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Truncate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple title",
			content:  "Hello World\nThis is content",
			expected: "Hello World",
		},
		{
			name:     "markdown header",
			content:  "# Hello World\nThis is content",
			expected: "Hello World",
		},
		{
			name:     "multiple hash markdown header",
			content:  "### Hello World\nThis is content",
			expected: "Hello World",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "long title truncated",
			content:  "This is a very long title that should be truncated to fifty characters maximum\nContent here",
			expected: "This is a very long title that should be truncated...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTitle(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractTitle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestThresholds(t *testing.T) {
	// Verify threshold values
	if DuplicateThreshold != 0.9 {
		t.Errorf("DuplicateThreshold = %v, want 0.9", DuplicateThreshold)
	}
	if RelatedThreshold != 0.7 {
		t.Errorf("RelatedThreshold = %v, want 0.7", RelatedThreshold)
	}
	if DefaultTopK != 5 {
		t.Errorf("DefaultTopK = %v, want 5", DefaultTopK)
	}
}

func TestDefaultWeights(t *testing.T) {
	total := DefaultWeights.Vector + DefaultWeights.TagCoOccur + DefaultWeights.TimeProx
	if diff := total - 1.0; diff > 0.001 || diff < -0.001 {
		t.Errorf("DefaultWeights sum = %v, want 1.0", total)
	}
}
