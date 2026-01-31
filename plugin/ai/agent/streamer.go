package agent

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"time"
)

// StreamEvent represents a standardized event for the Web UI.
type StreamEvent struct {
	Type      string         `json:"type"`           // thinking, tool_use, tool_result, answer, error
	Content   string         `json:"content"`        // The actual text content
	Meta      map[string]any `json:"meta,omitempty"` // Extra metadata (tool name, file path, etc)
	Timestamp int64          `json:"timestamp"`
}

// BiDirectionalStreamer handles the IO loop for a session.
type BiDirectionalStreamer struct {
	logger *slog.Logger
}

// NewBiDirectionalStreamer creates a streamer.
func NewBiDirectionalStreamer(logger *slog.Logger) *BiDirectionalStreamer {
	return &BiDirectionalStreamer{
		logger: logger,
	}
}

// StreamOutput reads from stdout and sends events to the callback channel.
// It runs until stdout is closed or context cancelled.
func (s *BiDirectionalStreamer) StreamOutput(stdout io.Reader, eventChan chan<- StreamEvent) error {
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg StreamMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			// Non-JSON output (e.g. plain text or debris)
			// Treat as raw log or partial answer?
			// For now, treat as system log or ignore if strict.
			// Let's send as "sys.log" or generic answer part.
			s.logger.Debug("Streamer: non-json line", "line", string(line))
			eventChan <- StreamEvent{
				Type:      "sys.log",
				Content:   string(line),
				Timestamp: time.Now().UnixMilli(),
			}
			continue
		}

		// Transform to StreamEvent
		events := s.transformMessageToEvents(msg)
		for _, e := range events {
			eventChan <- e
		}

		// Check if we should stop? Persistent session doesn't stop on "result".
		// Actually, standard `cc_runner` stops on result.
		// In async mode, we keep reading.
	}

	return scanner.Err()
}

// transformMessageToEvents converts internal CLI message to UI events.
func (s *BiDirectionalStreamer) transformMessageToEvents(msg StreamMessage) []StreamEvent {
	var events []StreamEvent
	ts := time.Now().UnixMilli()

	switch msg.Type {
	case "thinking", "status":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				events = append(events, StreamEvent{
					Type:      "ai.thinking",
					Content:   block.Text,
					Timestamp: ts,
				})
			}
		}

	case "tool_use":
		events = append(events, StreamEvent{
			Type:      "ai.tool.call",
			Content:   msg.Name, // Using content for Name for simplicity, or empty
			Meta:      map[string]any{"name": msg.Name, "input": msg.Input},
			Timestamp: ts,
		})

	case "tool_result":
		// Tool result usually contains output.
		// msg.Output or msg.Content?
		content := msg.Output
		if content == "" {
			// fallback
			if len(msg.Content) > 0 {
				content = "Has content blocks"
			}
		}

		isError := false
		if msg.Error != "" {
			isError = true
			content = msg.Error
		} else if msg.Status == "error" {
			isError = true
		}

		events = append(events, StreamEvent{
			Type:      "ai.tool.result",
			Content:   content,
			Meta:      map[string]any{"is_error": isError},
			Timestamp: ts,
		})

	case "message", "assistant", "text":
		// Standard text response
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				events = append(events, StreamEvent{
					Type:      "ai.answer",
					Content:   block.Text,
					Timestamp: ts,
				})
			} else if block.Type == "tool_use" {
				events = append(events, StreamEvent{
					Type:      "ai.tool.call",
					Content:   block.Name,
					Meta:      map[string]any{"name": block.Name, "input": block.Input, "id": block.ID},
					Timestamp: ts,
				})
			}
		}

	case "user":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "tool_result" {
				events = append(events, StreamEvent{
					Type:      "ai.tool.result",
					Content:   block.Content,
					Meta:      map[string]any{"is_error": block.IsError},
					Timestamp: ts,
				})
			}
		}

	case "error":
		events = append(events, StreamEvent{
			Type:      "sys.error",
			Content:   msg.Error,
			Timestamp: ts,
		})

	default:
		// Fallback for untyped text
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				events = append(events, StreamEvent{
					Type:      "ai.answer",
					Content:   block.Text,
					Timestamp: ts,
				})
			}
		}
	}

	return events
}

// BuildUserMessage constructs the JSON payload for user input.
func (s *BiDirectionalStreamer) BuildUserMessage(text string) map[string]any {
	return map[string]any{
		"type": "user",
		"message": map[string]any{
			"role": "user",
			"content": []map[string]string{
				{
					"type": "text",
					"text": text,
				},
			},
		},
	}
}
