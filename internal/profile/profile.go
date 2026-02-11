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

// Profile is the configuration to start main server.
type Profile struct {
	AIDeepSeekAPIKey     string
	AIEmbeddingModel     string
	TikaServerURL        string
	UNIXSock             string
	Mode                 string
	DSN                  string
	Driver               string
	Version              string
	InstanceURL          string
	OCRLanguages         string
	AIEmbeddingProvider  string
	AILLMProvider        string
	Addr                 string
	TessdataPath         string
	Data                 string
	AIDeepSeekBaseURL    string
	AIOpenAIAPIKey       string
	AIOpenAIBaseURL      string
	AIOllamaBaseURL      string
	AISiliconFlowAPIKey  string
	AIRerankModel        string
	AILLMModel           string
	AIZAI_APIKey         string // Z.AI (智谱) API key
	AIZAIBaseURL         string // Z.AI base URL
	TesseractPath        string
	AISiliconFlowBaseURL string
	Port                 int
	OCREnabled           bool
	TextExtractEnabled   bool
	AIEnabled            bool
}

func (p *Profile) IsDev() bool {
	return p.Mode != "prod"
}

// IsAIEnabled returns true if AI is enabled and at least one API key or base URL is configured.
func (p *Profile) IsAIEnabled() bool {
	return p.AIEnabled && (p.AISiliconFlowAPIKey != "" || p.AIOpenAIAPIKey != "" || p.AIOllamaBaseURL != "" || p.AIDeepSeekAPIKey != "" || p.AIZAI_APIKey != "")
}

// getEnvOrDefault returns the environment variable value or the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// FromEnv loads configuration from environment variables.
func (p *Profile) FromEnv() {
	p.AIEnabled = getEnvOrDefault("DIVINESENSE_AI_ENABLED", "false") == "true"
	p.AIEmbeddingProvider = getEnvOrDefault("DIVINESENSE_AI_EMBEDDING_PROVIDER", "siliconflow")
	p.AILLMProvider = getEnvOrDefault("DIVINESENSE_AI_LLM_PROVIDER", "zai")
	p.AISiliconFlowAPIKey = getEnvOrDefault("DIVINESENSE_AI_SILICONFLOW_API_KEY", "")
	p.AISiliconFlowBaseURL = getEnvOrDefault("DIVINESENSE_AI_SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1")
	p.AIDeepSeekAPIKey = getEnvOrDefault("DIVINESENSE_AI_DEEPSEEK_API_KEY", "")
	p.AIDeepSeekBaseURL = getEnvOrDefault("DIVINESENSE_AI_DEEPSEEK_BASE_URL", "https://api.deepseek.com")
	p.AIOpenAIAPIKey = getEnvOrDefault("DIVINESENSE_AI_OPENAI_API_KEY", "")
	p.AIOpenAIBaseURL = getEnvOrDefault("DIVINESENSE_AI_OPENAI_BASE_URL", "https://api.openai.com/v1")
	p.AIOllamaBaseURL = getEnvOrDefault("DIVINESENSE_AI_OLLAMA_BASE_URL", "http://localhost:11434")
	p.AIZAI_APIKey = getEnvOrDefault("DIVINESENSE_AI_ZAI_API_KEY", "")
	p.AIZAIBaseURL = getEnvOrDefault("DIVINESENSE_AI_ZAI_BASE_URL", "https://open.bigmodel.cn/api/paas/v4")
	p.AIEmbeddingModel = getEnvOrDefault("DIVINESENSE_AI_EMBEDDING_MODEL", "BAAI/bge-m3")
	p.AIRerankModel = getEnvOrDefault("DIVINESENSE_AI_RERANK_MODEL", "BAAI/bge-reranker-v2-m3")
	p.AILLMModel = getEnvOrDefault("DIVINESENSE_AI_LLM_MODEL", "glm-4.7")

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
