package agent

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"time"
)

// StreamEventMeta provides strongly-typed metadata for StreamEvent.
// StreamEventMeta 为 StreamEvent 提供强类型的元数据。
type StreamEventMeta struct {
	ToolName  string `json:"tool_name,omitempty"`   // Tool name for tool_use events
	ToolID    string `json:"tool_id,omitempty"`     // Tool ID for tool_use events
	IsError   bool   `json:"is_error,omitempty"`    // Error flag for tool_result events
	FilePath  string `json:"file_path,omitempty"`   // File path for file operations
	ExitCode  int    `json:"exit_code,omitempty"`   // Process exit code for run events
	Duration  int    `json:"duration_ms,omitempty"` // Operation duration in milliseconds
	SessionID string `json:"session_id,omitempty"`  // Associated session ID

	// Raw allows access to any additional fields not in the typed struct.
	// Raw 允许访问类型化结构之外的任何其他字段。
	Raw map[string]any `json:"-"`
}

// StreamEvent represents a standardized event for the Web UI.
type StreamEvent struct {
	Type      string           `json:"type"`           // thinking, tool_use, tool_result, answer, error
	Content   string           `json:"content"`        // The actual text content
	Meta      *StreamEventMeta `json:"meta,omitempty"` // Strongly-typed metadata
	Timestamp int64            `json:"timestamp"`
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
// Uses frontend-compatible event types (ParrotEventType).
func (s *BiDirectionalStreamer) transformMessageToEvents(msg StreamMessage) []StreamEvent {
	var events []StreamEvent
	ts := time.Now().UnixMilli()

	switch msg.Type {
	case "thinking", "status":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				events = append(events, StreamEvent{
					Type:      "thinking", // Frontend: ParrotEventType.THINKING
					Content:   block.Text,
					Timestamp: ts,
				})
			}
		}

	case "tool_use":
		meta := &StreamEventMeta{
			ToolName: msg.Name,
			ToolID:   "", // May be available in block.ID
		}
		// Store input in Raw for flexibility
		if msg.Input != nil {
			meta.Raw = map[string]any{"input": msg.Input}
		}
		events = append(events, StreamEvent{
			Type:      "tool_use", // Frontend: ParrotEventType.TOOL_USE
			Content:   msg.Name,
			Meta:      meta,
			Timestamp: ts,
		})

	case "tool_result":
		content := msg.Output
		if content == "" {
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
			Type:      "tool_result", // Frontend: ParrotEventType.TOOL_RESULT
			Content:   content,
			Meta:      &StreamEventMeta{IsError: isError},
			Timestamp: ts,
		})

	case "message", "assistant", "text":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				events = append(events, StreamEvent{
					Type:      "answer", // Frontend: ParrotEventType.ANSWER
					Content:   block.Text,
					Timestamp: ts,
				})
			} else if block.Type == "tool_use" {
				meta := &StreamEventMeta{
					ToolName: block.Name,
					ToolID:   block.ID,
				}
				if block.Input != nil {
					meta.Raw = map[string]any{"input": block.Input}
				}
				events = append(events, StreamEvent{
					Type:      "tool_use", // Frontend: ParrotEventType.TOOL_USE
					Content:   block.Name,
					Meta:      meta,
					Timestamp: ts,
				})
			}
		}

	case "user":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "tool_result" {
				events = append(events, StreamEvent{
					Type:      "tool_result", // Frontend: ParrotEventType.TOOL_RESULT
					Content:   block.Content,
					Meta:      &StreamEventMeta{IsError: block.IsError},
					Timestamp: ts,
				})
			}
		}

	case "error":
		events = append(events, StreamEvent{
			Type:      "error", // Frontend: ParrotEventType.ERROR
			Content:   msg.Error,
			Timestamp: ts,
		})

	default:
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				events = append(events, StreamEvent{
					Type:      "answer", // Frontend: ParrotEventType.ANSWER
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
