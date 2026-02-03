// Package duplicate - similarity calculation for P2-C002.
package duplicate

import (
	"math"
	"strings"
	"time"
)

// TimeDecayDays is the decay period for time proximity calculation.
const TimeDecayDays = 7

// CosineSimilarity calculates cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// TagCoOccurrence calculates Jaccard similarity between two tag sets.
func TagCoOccurrence(tags1, tags2 []string) float64 {
	if len(tags1) == 0 && len(tags2) == 0 {
		return 0
	}

	// Build sets (case-insensitive)
	set1 := make(map[string]bool)
	for _, tag := range tags1 {
		set1[strings.ToLower(strings.TrimSpace(tag))] = true
	}

	set2 := make(map[string]bool)
	for _, tag := range tags2 {
		set2[strings.ToLower(strings.TrimSpace(tag))] = true
	}

	// Calculate intersection
	var intersection int
	for tag := range set1 {
		if set2[tag] {
			intersection++
		}
	}

	// Jaccard similarity = intersection / union
	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// TimeProximity calculates time proximity using exponential decay.
// Returns 1.0 for same day, decaying exponentially over TimeDecayDays.
func TimeProximity(newTime, candidateTime time.Time) float64 {
	daysDiff := newTime.Sub(candidateTime).Hours() / 24
	if daysDiff < 0 {
		daysDiff = -daysDiff
	}

	// Exponential decay: e^(-days/7)
	return math.Exp(-daysDiff / TimeDecayDays)
}

// FindSharedTags returns tags that appear in both slices.
func FindSharedTags(tags1, tags2 []string) []string {
	set := make(map[string]bool)
	for _, tag := range tags1 {
		set[strings.ToLower(strings.TrimSpace(tag))] = true
	}

	var shared []string
	seen := make(map[string]bool)
	for _, tag := range tags2 {
		lower := strings.ToLower(strings.TrimSpace(tag))
		if set[lower] && !seen[lower] {
			shared = append(shared, tag)
			seen[lower] = true
		}
	}

	return shared
}

// CalculateWeightedSimilarity computes weighted similarity from breakdown.
func CalculateWeightedSimilarity(b *Breakdown, w Weights) float64 {
	return b.Vector*w.Vector + b.TagCoOccur*w.TagCoOccur + b.TimeProx*w.TimeProx
}

// Truncate truncates content to maxLen characters.
func Truncate(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// ExtractTitle extracts title from memo content (first line).
func ExtractTitle(content string) string {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	title := strings.TrimSpace(lines[0])
	// Remove markdown headers (handle multiple # symbols)
	for strings.HasPrefix(title, "#") {
		title = strings.TrimPrefix(title, "#")
	}
	title = strings.TrimSpace(title)
	return Truncate(title, 50)
}
