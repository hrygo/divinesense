package embedding

import (
	"strings"
	"unicode"
)

const (
	// ChunkSize is the maximum character count per chunk.
	ChunkSize = 500
	// ChunkOverlap is the character count overlap between chunks.
	ChunkOverlap = 50
)

// ChunkDocument splits a long document into multiple chunks for embedding.
// It preserves paragraph boundaries when possible.
func ChunkDocument(content string) []string {
	if len(content) <= ChunkSize {
		return []string{content}
	}

	// Split by paragraphs first
	paragraphs := splitParagraphs(content)

	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If adding this paragraph exceeds chunk size
		if currentChunk.Len()+len(para) > ChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, currentChunk.String())

			// Start new chunk with overlap
			currentChunk.Reset()
			overlapText := getOverlapText(chunks, ChunkOverlap)
			if overlapText != "" {
				currentChunk.WriteString(overlapText)
				currentChunk.WriteString("\n\n")
			}
		}

		// Add paragraph to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)

		// Handle very long paragraphs by force-splitting
		for currentChunk.Len() > ChunkSize {
			text := currentChunk.String()
			// Find a good break point (sentence or word)
			breakPoint := findBreakPoint(text[:ChunkSize])
			chunks = append(chunks, text[:breakPoint])

			// Start new chunk with remaining text
			remaining := text[breakPoint:]
			currentChunk.Reset()
			currentChunk.WriteString(remaining)
		}
	}

	// Add final chunk if not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// splitParagraphs splits content into paragraphs.
func splitParagraphs(content string) []string {
	// Split by common paragraph delimiters
	paragraphs := strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == '\r'
	})

	// Also split by double newlines (empty paragraphs)
	var result []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// getOverlapText returns the last N characters from the previous chunk for overlap.
func getOverlapText(chunks []string, overlapSize int) string {
	if len(chunks) == 0 {
		return ""
	}

	lastChunk := chunks[len(chunks)-1]
	if len(lastChunk) <= overlapSize {
		return lastChunk
	}

	// Try to break at word boundary
	overlapText := lastChunk[len(lastChunk)-overlapSize:]
	if idx := strings.IndexAny(overlapText, " \t"); idx > 0 {
		return overlapText[idx+1:]
	}

	return overlapText
}

// findBreakPoint finds a good position to split text (sentence or word boundary).
func findBreakPoint(text string) int {
	// Try to find sentence end
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '.' || text[i] == '!' || text[i] == '?' {
			// Check if it's really a sentence end (followed by space or end)
			if i == len(text)-1 || unicode.IsSpace(rune(text[i+1])) {
				return i + 1
			}
		}
	}

	// Try to find word boundary
	for i := len(text) - 1; i >= len(text)/2; i-- {
		if unicode.IsSpace(rune(text[i])) {
			return i
		}
	}

	// Force split at max length
	return len(text)
}
