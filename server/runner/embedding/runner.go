package embedding

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/store"
)

type Runner struct {
	embeddingService ai.EmbeddingService
	store            *store.Store
	model            string
	interval         time.Duration
	batchSize        int
}

// NewRunner creates a vector embedding runner.
// Parameters optimized for 2C2G: smaller batch size reduces memory peaks,
// longer interval reduces CPU contention.
func NewRunner(store *store.Store, embeddingService ai.EmbeddingService) *Runner {
	return &Runner{
		store:            store,
		embeddingService: embeddingService,
		interval:         2 * time.Minute,
		batchSize:        8,
		model:            "BAAI/bge-m3",
	}
}

// Run starts the background task.
func (r *Runner) Run(ctx context.Context) {
	// Process once on startup
	r.processNewMemos(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.processNewMemos(ctx)
		case <-ctx.Done():
			slog.Info("embedding runner stopped")
			return
		}
	}
}

// RunOnce processes memos once (for manual trigger).
func (r *Runner) RunOnce(ctx context.Context) {
	r.processNewMemos(ctx)
}

func (r *Runner) processNewMemos(ctx context.Context) {
	// Find memos without embeddings
	memos, err := r.findMemosWithoutEmbedding(ctx)
	if err != nil {
		slog.Error("failed to find memos without embedding", "error", err)
		return
	}

	if len(memos) == 0 {
		return
	}

	slog.Info("processing memos for embedding", "count", len(memos))

	// Process in batches
	for i := 0; i < len(memos); i += r.batchSize {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			slog.Info("embedding processing cancelled", "processed", i, "total", len(memos))
			return
		default:
			// Continue processing
		}

		end := i + r.batchSize
		if end > len(memos) {
			end = len(memos)
		}
		batch := memos[i:end]

		if err := r.processBatch(ctx, batch); err != nil {
			slog.Error("failed to process batch", "error", err)
			continue
		}
		slog.Info("batch processed", "count", len(batch), "progress", fmt.Sprintf("%d/%d", end, len(memos)))
	}
}

func (r *Runner) findMemosWithoutEmbedding(ctx context.Context) ([]*store.Memo, error) {
	return r.store.FindMemosWithoutEmbedding(ctx, &store.FindMemosWithoutEmbedding{
		Model: r.model,
		Limit: r.batchSize * 20, // Fetch more data, but process in small batches
	})
}

func (r *Runner) processBatch(ctx context.Context, memos []*store.Memo) error {
	// Check context before processing batch
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Extract content with attachment text
	texts := make([]string, len(memos))
	for i, m := range memos {
		texts[i] = r.buildMemoContentWithAttachments(ctx, m)
	}

	// Generate vectors in batch
	vectors, err := r.embeddingService.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}

	// Store vectors
	for i, m := range memos {
		_, err := r.store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
			MemoID:    m.ID,
			Embedding: vectors[i],
			Model:     r.model,
		})
		if err != nil {
			slog.Error("failed to upsert embedding", "memoID", m.ID, "error", err)
		}
	}

	return nil
}

// buildMemoContentWithAttachments builds the text content for embedding by combining
// memo content with OCR/extracted text from attachments.
func (r *Runner) buildMemoContentWithAttachments(ctx context.Context, m *store.Memo) string {
	content := m.Content

	// Fetch attachments for this memo
	attachments, err := r.store.ListAttachments(ctx, &store.FindAttachment{
		MemoID: &m.ID,
		Limit:  intPtr(50), // Max 50 attachments per memo
	})
	if err != nil {
		slog.Warn("failed to fetch attachments for memo", "memoID", m.ID, "error", err)
		return content
	}

	// Collect attachment text
	var attachmentTexts []string
	for _, att := range attachments {
		if att.RowStatus != "NORMAL" {
			continue
		}
		// Prioritize OCR text (for images) over extracted text (for documents)
		if att.OCRText != "" {
			attachmentTexts = append(attachmentTexts, att.OCRText)
		} else if att.ExtractedText != "" {
			attachmentTexts = append(attachmentTexts, att.ExtractedText)
		}
	}

	// Combine content with attachment text
	if len(attachmentTexts) > 0 {
		// Use a separator that won't confuse the embedding model
		combined := content + "\n\n[附件内容]\n" + joinNonEmpty(attachmentTexts, "\n---\n")
		// Truncate if too long (most models have limits, BAAI/bge-m3 supports up to 8192 tokens)
		if len(combined) > 8000 {
			// Keep memo content and truncate attachment text if needed
			if len(m.Content) < 8000 {
				combined = m.Content + "\n\n[附件内容]\n" + joinNonEmpty(attachmentTexts, "\n---\n")[:8000-len(m.Content)-20]
			} else {
				combined = m.Content[:8000]
			}
		}
		return combined
	}

	// Truncate content if too long (BAAI/bge-m3 supports up to 8192 tokens)
	// 8000 chars is a safe limit for most content (roughly 2000-4000 tokens depending on language)
	if len(content) > 8000 {
		return content[:8000]
	}
	return content
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// joinNonEmpty joins non-empty strings with a separator.
func joinNonEmpty(strs []string, sep string) string {
	var result []string
	for _, s := range strs {
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return ""
	}
	joined := ""
	for i, s := range result {
		if i > 0 {
			joined += sep
		}
		joined += s
	}
	return joined
}
