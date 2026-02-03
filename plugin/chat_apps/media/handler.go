// Package media provides multimedia processing for chat apps.
package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// MediaHandler processes multimedia messages.
type MediaHandler struct {
	config *MediaConfig
	client *http.Client
}

// MediaConfig holds configuration for media processing.
type MediaConfig struct {
	// Whisper (voice-to-text)
	WhisperEndpoint string
	WhisperAPIKey   string

	// OCR (image text extraction)
	OCREngine string // "tesseract" or "api"
	OCRBin    string // Path to tesseract binary (if using local)

	// Limits
	MaxPhotoSizeMB    int64
	MaxDocumentSizeMB int64
	MaxAudioSizeMB    int64
	MaxVideoSizeMB    int64
}

// NewMediaHandler creates a new media handler.
func NewMediaHandler(config *MediaConfig) *MediaHandler {
	if config.MaxPhotoSizeMB == 0 {
		config.MaxPhotoSizeMB = 20
	}
	if config.MaxDocumentSizeMB == 0 {
		config.MaxDocumentSizeMB = 50
	}
	if config.MaxAudioSizeMB == 0 {
		config.MaxAudioSizeMB = 50
	}
	if config.MaxVideoSizeMB == 0 {
		config.MaxVideoSizeMB = 50
	}

	// Configure HTTP client with timeout and connection pool
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true, // Media downloads don't benefit from compression
			// Force HTTP/2 for better performance (if supported)
			ForceAttemptHTTP2: true,
		},
	}

	return &MediaHandler{
		config: config,
		client: client,
	}
}

// ProcessAudio converts audio data to text using Whisper API.
func (h *MediaHandler) ProcessAudio(ctx context.Context, data []byte, mimeType string) (string, error) {
	if h.config.WhisperEndpoint == "" {
		return "", fmt.Errorf("whisper endpoint not configured")
	}

	// Check file size (max 25MB for Whisper API)
	const maxWhisperSize = 25 * 1024 * 1024
	if int64(len(data)) > maxWhisperSize {
		return "", fmt.Errorf("audio file too large: %d MB (max 25 MB)", len(data)/(1024*1024))
	}

	// Create multipart request
	req, err := h.createWhisperRequest(ctx, data, mimeType)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper API error: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result WhisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Text, nil
}

// ProcessImage extracts text from images using OCR.
func (h *MediaHandler) ProcessImage(ctx context.Context, data []byte) (string, error) {
	switch h.config.OCREngine {
	case "tesseract":
		return h.processWithTesseract(ctx, data)
	case "api":
		return h.processWithOCRAPI(ctx, data)
	default:
		return "", fmt.Errorf("OCR engine not configured")
	}
}

// SaveTempFile saves data to a temporary file and returns the path.
func (h *MediaHandler) SaveTempFile(data []byte, ext string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "media_*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// processWithTesseract uses local tesseract binary.
func (h *MediaHandler) processWithTesseract(ctx context.Context, data []byte) (string, error) {
	// Save to temp file
	tmpPath, err := h.SaveTempFile(data, ".png")
	defer os.Remove(tmpPath)
	if err != nil {
		return "", err
	}

	// Run tesseract
	cmd := exec.CommandContext(ctx, h.config.OCRBin, tmpPath, "stdout")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w", err)
	}

	return string(output), nil
}

// processWithOCRAPI uses an OCR API service.
func (h *MediaHandler) processWithOCRAPI(ctx context.Context, data []byte) (string, error) {
	// Implement OCR API call (e.g., Google Cloud Vision, AWS Textract)
	return "", fmt.Errorf("API OCR not implemented")
}

// createWhisperRequest creates a multipart request to the Whisper API.
func (h *MediaHandler) createWhisperRequest(ctx context.Context, data []byte, mimeType string) (*http.Request, error) {
	// Determine file extension from MIME type
	ext := ".m4a"
	switch {
	case strings.HasPrefix(mimeType, "audio/ogg"):
		ext = ".ogg"
	case strings.HasPrefix(mimeType, "audio/wav"):
		ext = ".wav"
	case strings.HasPrefix(mimeType, "audio/mp3"):
		ext = ".mp3"
	case strings.HasPrefix(mimeType, "audio/mpeg"):
		ext = ".mp3"
	}

	// Create multipart body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", "audio"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}

	// Add model field
	writer.WriteField("model", "whisper-1")

	// Add response format
	writer.WriteField("response_format", "text")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", h.config.WhisperEndpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if h.config.WhisperAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.config.WhisperAPIKey)
	}

	return req, nil
}

// DetectMIMEType detects the MIME type of data.
func DetectMIMEType(data []byte) string {
	mimeType := http.DetectContentType(data)
	return mimeType
}

// GetFileExtension returns the file extension for a MIME type.
func GetFileExtension(mimeType string) string {
	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(exts) == 0 {
		return ".bin"
	}
	return exts[0]
}

// WhisperResponse represents the Whisper API response.
type WhisperResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Words    []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"words,omitempty"`
}

// Size validation methods

// ValidatePhotoSize checks if photo is within size limits.
func (h *MediaHandler) ValidatePhotoSize(size int64) error {
	if size > h.config.MaxPhotoSizeMB*1024*1024 {
		return fmt.Errorf("photo too large: %d MB (max %d MB)", size/(1024*1024), h.config.MaxPhotoSizeMB)
	}
	return nil
}

// ValidateDocumentSize checks if document is within size limits.
func (h *MediaHandler) ValidateDocumentSize(size int64) error {
	if size > h.config.MaxDocumentSizeMB*1024*1024 {
		return fmt.Errorf("document too large: %d MB (max %d MB)", size/(1024*1024), h.config.MaxDocumentSizeMB)
	}
	return nil
}

// ValidateAudioSize checks if audio is within size limits.
func (h *MediaHandler) ValidateAudioSize(size int64) error {
	if size > h.config.MaxAudioSizeMB*1024*1024 {
		return fmt.Errorf("audio too large: %d MB (max %d MB)", size/(1024*1024), h.config.MaxAudioSizeMB)
	}
	return nil
}
