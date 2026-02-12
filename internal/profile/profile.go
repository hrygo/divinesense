package profile

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// Profile is configuration to start main server.
type Profile struct {
	// Unified LLM configuration (OpenAI-compatible protocol)
	// All providers (zai, deepseek, openai, siliconflow, ollama) use the same config
	ALLMProvider string // Provider identifier: zai, deepseek, openai, siliconflow, dashscope, openrouter, ollama
	ALLMAPIKey   string // Unified LLM API key
	ALLMBaseURL  string // Unified LLM base URL (optional, has default per provider)
	ALLMModel    string // Model name: glm-4.7, deepseek-chat, gpt-4o, etc.

	// Embedding configuration
	AIEmbeddingProvider string
	AIEmbeddingModel    string
	AIEmbeddingAPIKey   string
	AIEmbeddingBaseURL  string

	// Reranker configuration
	AIRerankProvider string
	AIRerankModel    string
	AIRerankAPIKey   string
	AIRerankBaseURL  string

	// Intent Classifier configuration
	AIIntentProvider string
	AIIntentModel    string
	AIIntentAPIKey   string
	AIIntentBaseURL  string

	// Other configurations
	TikaServerURL      string
	UNIXSock           string
	Mode               string
	DSN                string
	Driver             string
	Version            string
	InstanceURL        string
	OCRLanguages       string
	Addr               string
	TessdataPath       string
	Data               string
	TesseractPath      string
	Port               int
	OCREnabled         bool
	TextExtractEnabled bool
	AIEnabled          bool
}

// Provider default configurations for LLM.
// Used when LLM_BASE_URL is not explicitly set.
var llmProviderDefaults = map[string]struct {
	BaseURL string
	Model   string
}{
	"zai": {
		BaseURL: "https://open.bigmodel.cn/api/paas/v4",
		Model:   "glm-4.7", // Latest stable: glm-4.7, Flagship: glm-5
	},
	"deepseek": {
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat", // DeepSeek-V3.2
	},
	"openai": {
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-5.2", // GPT-5.2 (Feb 2026)
	},
	"siliconflow": {
		BaseURL: "https://api.siliconflow.cn/v1",
		Model:   "Qwen/Qwen2.5-72B-Instruct",
	},
	"dashscope": {
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Model:   "qwen-max-latest",
	},
	"openrouter": {
		BaseURL: "https://openrouter.ai/api/v1",
		Model:   "deepseek/deepseek-chat", // Varies by user preference
	},
	"ollama": {
		BaseURL: "http://localhost:11434",
		Model:   "llama3.1",
	},
}

func (p *Profile) IsDev() bool {
	return p.Mode != "prod"
}

// IsAIEnabled returns true if AI is enabled and LLM API key is configured.
func (p *Profile) IsAIEnabled() bool {
	return p.ALLMAPIKey != ""
}

// getEnvOrDefault returns environment variable value or default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// FromEnv loads configuration from environment variables.
func (p *Profile) FromEnv() {
	// Unified LLM configuration
	p.ALLMProvider = getEnvOrDefault("DIVINESENSE_AI_LLM_PROVIDER", "zai")
	p.ALLMAPIKey = getEnvOrDefault("DIVINESENSE_AI_LLM_API_KEY", "")
	p.ALLMBaseURL = getEnvOrDefault("DIVINESENSE_AI_LLM_BASE_URL", "")
	p.ALLMModel = getEnvOrDefault("DIVINESENSE_AI_LLM_MODEL", "")

	// AI is enabled if API key is configured
	p.AIEnabled = p.ALLMAPIKey != ""

	// Validate and apply provider defaults if not explicitly set
	if p.ALLMProvider != "" {
		if _, ok := llmProviderDefaults[p.ALLMProvider]; !ok {
			slog.Warn("Unknown LLM provider, using default: zai", "provider", p.ALLMProvider)
			p.ALLMProvider = "zai"
		}
	}
	if p.ALLMBaseURL == "" || p.ALLMModel == "" {
		if defaults, ok := llmProviderDefaults[p.ALLMProvider]; ok {
			if p.ALLMBaseURL == "" {
				p.ALLMBaseURL = defaults.BaseURL
			}
			if p.ALLMModel == "" {
				p.ALLMModel = defaults.Model
			}
		}
	}

	// Embedding configuration
	// Embedding configuration
	p.AIEmbeddingProvider = getEnvOrDefault("DIVINESENSE_AI_EMBEDDING_PROVIDER", "siliconflow")
	p.AIEmbeddingModel = getEnvOrDefault("DIVINESENSE_AI_EMBEDDING_MODEL", "BAAI/bge-m3")
	p.AIEmbeddingAPIKey = getEnvOrDefault("DIVINESENSE_AI_EMBEDDING_API_KEY", "")
	p.AIEmbeddingBaseURL = getEnvOrDefault("DIVINESENSE_AI_EMBEDDING_BASE_URL", "https://api.siliconflow.cn/v1")

	// Reranker configuration
	p.AIRerankProvider = getEnvOrDefault("DIVINESENSE_AI_RERANK_PROVIDER", "siliconflow")
	p.AIRerankModel = getEnvOrDefault("DIVINESENSE_AI_RERANK_MODEL", "BAAI/bge-reranker-v2-m3")
	p.AIRerankAPIKey = getEnvOrDefault("DIVINESENSE_AI_RERANK_API_KEY", "")
	p.AIRerankBaseURL = getEnvOrDefault("DIVINESENSE_AI_RERANK_BASE_URL", "https://api.siliconflow.cn/v1")

	// Intent Classifier configuration
	p.AIIntentProvider = getEnvOrDefault("DIVINESENSE_AI_INTENT_PROVIDER", "siliconflow")
	p.AIIntentModel = getEnvOrDefault("DIVINESENSE_AI_INTENT_MODEL", "Qwen/Qwen2.5-7B-Instruct")
	p.AIIntentAPIKey = getEnvOrDefault("DIVINESENSE_AI_INTENT_API_KEY", "")
	p.AIIntentBaseURL = getEnvOrDefault("DIVINESENSE_AI_INTENT_BASE_URL", "https://api.siliconflow.cn/v1")

	// Attachment processing configuration
	p.OCREnabled = getEnvOrDefault("DIVINESENSE_OCR_ENABLED", "false") == "true"
	p.TextExtractEnabled = getEnvOrDefault("DIVINESENSE_TEXTEXTRACT_ENABLED", "false") == "true"
	p.TesseractPath = getEnvOrDefault("DIVINESENSE_OCR_TESSERACT_PATH", "tesseract")
	p.TessdataPath = getEnvOrDefault("DIVINESENSE_OCR_TESSDATA_PATH", "")
	p.OCRLanguages = getEnvOrDefault("DIVINESENSE_OCR_LANGUAGES", "chi_sim+eng")
	p.TikaServerURL = getEnvOrDefault("DIVINESENSE_TEXTEXTRACT_TIKA_URL", "http://localhost:9998")
}

func checkDataDir(dataDir string) (string, error) {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(dataDir) {
		relativeDir := filepath.Join(filepath.Dir(os.Args[0]), dataDir)
		absDir, err := filepath.Abs(relativeDir)
		if err != nil {
			return "", err
		}
		dataDir = absDir
	}

	// Trim trailing \ or / in case user supplies
	dataDir = strings.TrimRight(dataDir, "\\/")
	if _, err := os.Stat(dataDir); err != nil {
		return "", errors.Wrapf(err, "unable to access data folder %s", dataDir)
	}
	return dataDir, nil
}

func (p *Profile) Validate() error {
	if p.Mode != "demo" && p.Mode != "dev" && p.Mode != "prod" {
		p.Mode = "demo"
	}

	if p.Mode == "prod" && p.Data == "" {
		if runtime.GOOS == "windows" {
			p.Data = filepath.Join(os.Getenv("ProgramData"), "divinesense")
			if _, err := os.Stat(p.Data); os.IsNotExist(err) {
				if err := os.MkdirAll(p.Data, 0770); err != nil {
					slog.Error("failed to create data directory", slog.String("data", p.Data), slog.String("error", err.Error()))
					return err
				}
			}
		} else {
			p.Data = "/var/opt/divinesense"
		}
	}

	dataDir, err := checkDataDir(p.Data)
	if err != nil {
		slog.Error("failed to check dsn", slog.String("data", dataDir), slog.String("error", err.Error()))
		return err
	}

	p.Data = dataDir
	if p.Driver == "sqlite" && p.DSN == "" {
		dbFile := fmt.Sprintf("divinesense_%s.db", p.Mode)
		// Add _loc=auto and _allow_load_extension=1 for sqlite-vec support
		p.DSN = filepath.Join(dataDir, dbFile) + "?_loc=auto&_allow_load_extension=1"
	} else if p.Driver == "sqlite" && p.DSN != "" {
		// Ensure _loc=auto and _allow_load_extension=1 are set for custom DSN
		separator := "?"
		if strings.Contains(p.DSN, "?") {
			separator = "&"
		}
		if !strings.Contains(p.DSN, "_loc=") {
			p.DSN += separator + "_loc=auto"
			separator = "&"
		}
		if !strings.Contains(p.DSN, "_allow_load_extension=") {
			p.DSN += separator + "_allow_load_extension=1"
		}
	}

	return nil
}
