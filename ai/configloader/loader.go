package configloader

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader is a unified configuration loader for AI-related YAML files.
type Loader struct {
	baseDir string
	cache   sync.Map
}

// NewLoader creates a new configuration loader.
func NewLoader(baseDir string) *Loader {
	return &Loader{
		baseDir: baseDir,
	}
}

// Load loads a single YAML file and unmarshals it into target.
func (l *Loader) Load(subPath string, target any) error {
	data, err := l.ReadFileWithFallback(subPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", subPath, err)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal YAML %s: %w", subPath, err)
	}

	return nil
}

// LoadCached loads a configuration with caching.
// If the file is already cached, returns the cached value.
// Otherwise, calls factory to create the target and caches it.
func (l *Loader) LoadCached(subPath string, factory func() any) (any, error) {
	// Check cache first
	if cached, ok := l.cache.Load(subPath); ok {
		return cached, nil
	}

	// Create new instance using factory
	target := factory()

	// Load data
	if err := l.Load(subPath, target); err != nil {
		return nil, err
	}

	// Store in cache
	l.cache.Store(subPath, target)

	return target, nil
}

// LoadDir loads all YAML files from a directory.
// The factory function is called for each file to create the target struct.
func (l *Loader) LoadDir(subDir string, factory func(path string) (any, error)) (map[string]any, error) {
	dirPath := filepath.Join(l.baseDir, subDir)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dirPath, err)
	}

	result := make(map[string]any)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		filePath := filepath.Join(subDir, entry.Name())
		target, err := factory(filePath)
		if err != nil {
			return nil, fmt.Errorf("create target for %s: %w", filePath, err)
		}

		if err := l.Load(filePath, target); err != nil {
			return nil, fmt.Errorf("load %s: %w", filePath, err)
		}

		result[filePath] = target
	}

	return result, nil
}

// ReadFileWithFallback tries to read file from path relative to baseDir,
// then falls back to executable directory for production builds.
func (l *Loader) ReadFileWithFallback(path string) ([]byte, error) {
	// Try relative to baseDir first
	absPath := filepath.Join(l.baseDir, path)
	data, err := os.ReadFile(absPath)
	if err == nil {
		return data, nil
	}

	// Fallback: try relative to executable directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	execDir := filepath.Dir(execPath)
	execAbsPath := filepath.Join(execDir, l.baseDir, path)

	return os.ReadFile(execAbsPath)
}

// ClearCache clears the configuration cache.
func (l *Loader) ClearCache() {
	l.cache = sync.Map{}
}
