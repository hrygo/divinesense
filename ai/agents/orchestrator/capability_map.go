// Package orchestrator implements the Orchestrator-Workers pattern for multi-agent coordination.
// It uses LLM to dynamically decompose tasks, dispatch to expert agents, and aggregate results.
package orchestrator

import (
	"context"
	"log/slog"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"

	agents "github.com/hrygo/divinesense/ai/agents"
)

// Capability represents a single capability that an expert agent can provide.
// Capability represents a single capability that an expert agent can provide.
type Capability string

// ExpertInfo contains information about an expert agent.
// ExpertInfo 包含专家代理的信息。
type ExpertInfo struct {
	Name         string   `json:"name"`
	Emoji        string   `json:"emoji"`
	Title        string   `json:"title"`
	Capabilities []string `json:"capabilities"`
}

// KeywordIndex provides fast keyword-based routing.
// KeywordIndex 提供快速的基于关键词的路由，用于 Layer 2 规则匹配。
type KeywordIndex struct {
	// keyword -> expert names
	keywords map[string][]string
	// compiled regex -> expert names
	patterns map[*regexp.Regexp][]string
	// exclude patterns (compiled)
	excludes []*regexp.Regexp
	// expert name -> priority
	priorities map[string]int
	// expert name -> title (for fuzzy matching)
	titles map[string]string
}

// EmbeddingProvider defines an interface for generating embeddings.
// This enables lazy initialization of semantic index without circular dependencies.
type EmbeddingProvider interface {
	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// SemanticIndex provides semantic-based routing using embeddings.
// SemanticIndex 提供基于向量的语义路由，用于 Layer 3 匹配。
type SemanticIndex struct {
	initialized bool
	// expert name -> example embeddings (averaged)
	expertEmbeddings map[string][]float32
	// expert name -> original examples (for debugging)
	expertExamples map[string][]string
	// similarity threshold for routing
	threshold float32
}

// NewSemanticIndex creates a new SemanticIndex.
func NewSemanticIndex() *SemanticIndex {
	return &SemanticIndex{
		initialized:      false,
		expertEmbeddings: make(map[string][]float32),
		expertExamples:   make(map[string][]string),
		threshold:        0.5, // Default threshold for cosine similarity
	}
}

// NewKeywordIndex creates an empty KeywordIndex.
func NewKeywordIndex() *KeywordIndex {
	return &KeywordIndex{
		keywords:   make(map[string][]string),
		patterns:   make(map[*regexp.Regexp][]string),
		excludes:   make([]*regexp.Regexp, 0),
		priorities: make(map[string]int),
		titles:     make(map[string]string),
	}
}

// CapabilityMap provides a thread-safe mapping from capabilities to expert agents.
// It is used at runtime to build the capability-to-expert mapping.
type CapabilityMap struct {
	mu                    sync.RWMutex
	capabilityToExperts   map[Capability][]*ExpertInfo
	keywordToCapabilities map[string][]Capability
	experts               map[string]*ExpertInfo

	// Keyword-based routing index for Layer 2 rule matching
	keywordIndex *KeywordIndex
	// Semantic-based routing index for Layer 3 matching
	semanticIndex *SemanticIndex
	// Embedding provider for semantic matching
	embeddingProvider EmbeddingProvider
}

// NewCapabilityMap creates an empty CapabilityMap.
// NewCapabilityMap 创建一个空的 CapabilityMap。
func NewCapabilityMap() *CapabilityMap {
	return &CapabilityMap{
		capabilityToExperts:   make(map[Capability][]*ExpertInfo),
		keywordToCapabilities: make(map[string][]Capability),
		experts:               make(map[string]*ExpertInfo),
	}
}

// BuildFromConfigs builds the capability map from ParrotSelfCognition configurations.
// BuildFromConfigs 从 ParrotSelfCognition 配置构建能力映射。
func (cm *CapabilityMap) BuildFromConfigs(configs []*agents.ParrotSelfCognition) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Clear existing mappings
	cm.capabilityToExperts = make(map[Capability][]*ExpertInfo)
	cm.keywordToCapabilities = make(map[string][]Capability)
	cm.experts = make(map[string]*ExpertInfo)

	for _, config := range configs {
		if config == nil {
			continue
		}

		expert := &ExpertInfo{
			Name:         config.Name,
			Emoji:        config.Emoji,
			Title:        config.Title,
			Capabilities: config.Capabilities,
		}

		// Add to experts map
		cm.experts[config.Name] = expert

		// Add to capabilityToExperts map
		for _, cap := range config.Capabilities {
			normalizedCap := cm.normalizeCapability(cap)
			if normalizedCap == "" {
				continue
			}
			cm.capabilityToExperts[Capability(normalizedCap)] = append(
				cm.capabilityToExperts[Capability(normalizedCap)],
				expert,
			)
		}

		// Add to keywordToCapabilities map
		for capName, triggers := range config.CapabilityTriggers {
			normalizedCap := cm.normalizeCapability(capName)
			if normalizedCap == "" {
				continue
			}

			for _, trigger := range triggers {
				normalizedTrigger := cm.normalizeCapability(trigger) // Reuse normalization logic
				if normalizedTrigger == "" {
					continue
				}
				cm.keywordToCapabilities[normalizedTrigger] = append(
					cm.keywordToCapabilities[normalizedTrigger],
					Capability(normalizedCap),
				)
			}
		}
	}
}

// BuildKeywordIndex builds the keyword index from routing configs.
// This enables fast Layer 2 rule-based routing without LLM.
// BuildKeywordIndex 从路由配置构建关键词索引，用于 Layer 2 规则匹配。
func (cm *CapabilityMap) BuildKeywordIndex(configs []*agents.ParrotSelfCognition) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.keywordIndex = NewKeywordIndex()

	for _, config := range configs {
		if config == nil {
			continue
		}

		expertName := config.Name
		if expertName == "" {
			continue
		}

		// Store title for fuzzy matching
		if config.Title != "" {
			cm.keywordIndex.titles[expertName] = config.Title
		}

		// If no routing config, skip indexing
		if config.Routing == nil {
			continue
		}

		routing := config.Routing

		// Index keywords
		for _, kw := range routing.Keywords {
			kwLower := strings.ToLower(kw)
			if kwLower == "" {
				continue
			}
			cm.keywordIndex.keywords[kwLower] = append(
				cm.keywordIndex.keywords[kwLower], expertName,
			)
		}

		// Compile and index patterns
		for _, pat := range routing.Patterns {
			if pat == "" {
				continue
			}
			re, err := regexp.Compile(pat)
			if err != nil {
				slog.Warn("invalid routing pattern",
					"expert", expertName,
					"pattern", pat,
					"error", err)
				continue
			}
			cm.keywordIndex.patterns[re] = append(cm.keywordIndex.patterns[re], expertName)
		}

		// Compile exclude patterns
		for _, ex := range routing.Excludes {
			if ex == "" {
				continue
			}
			re, err := regexp.Compile(ex)
			if err != nil {
				slog.Warn("invalid exclude pattern",
					"expert", expertName,
					"pattern", ex,
					"error", err)
				continue
			}
			cm.keywordIndex.excludes = append(cm.keywordIndex.excludes, re)
		}

		// Set priority
		cm.keywordIndex.priorities[expertName] = routing.Priority
	}
}

// BuildSemanticIndex builds the semantic index from routing configs.
// This pre-computes embeddings for semantic_examples at startup.
// It requires an EmbeddingProvider to generate embeddings.
// BuildSemanticIndex 从路由配置构建语义索引，在启动时预计算示例的 embedding 向量。
func (cm *CapabilityMap) BuildSemanticIndex(ctx context.Context, configs []*agents.ParrotSelfCognition, provider EmbeddingProvider) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Store embedding provider for semantic matching
	cm.embeddingProvider = provider

	// Initialize semantic index
	semanticIndex := NewSemanticIndex()

	for _, config := range configs {
		if config == nil {
			continue
		}

		expertName := config.Name
		if expertName == "" {
			continue
		}

		// If no routing config, skip
		if config.Routing == nil {
			continue
		}

		examples := config.Routing.SemanticExamples
		if len(examples) == 0 {
			continue
		}

		// Store examples for debugging
		semanticIndex.expertExamples[expertName] = examples

		// Generate embeddings for all examples
		var embeddings [][]float32
		for _, example := range examples {
			if example == "" {
				continue
			}
			emb, err := provider.Embed(ctx, example)
			if err != nil {
				slog.Warn("failed to embed semantic example",
					"expert", expertName,
					"example", example,
					"error", err)
				continue
			}
			embeddings = append(embeddings, emb)
		}

		if len(embeddings) == 0 {
			continue
		}

		// Average pool embeddings
		avgEmbedding := averageEmbeddings(embeddings)
		semanticIndex.expertEmbeddings[expertName] = avgEmbedding
		slog.Debug("semantic index built for expert",
			"expert", expertName,
			"examples", len(embeddings),
			"embedding_dim", len(avgEmbedding))
	}

	// Store semantic index
	cm.semanticIndex = semanticIndex
	semanticIndex.initialized = true
	slog.Info("semantic index built successfully", "experts", len(semanticIndex.expertEmbeddings))
}

// averageEmbeddings computes the element-wise average of multiple embeddings.
func averageEmbeddings(embeddings [][]float32) []float32 {
	if len(embeddings) == 0 {
		return nil
	}

	// All embeddings should have the same dimension
	n := len(embeddings[0])
	if n == 0 {
		return nil
	}

	result := make([]float32, n)

	// Sum all embeddings
	for _, emb := range embeddings {
		for i := 0; i < n; i++ {
			result[i] += emb[i]
		}
	}

	// Divide by count
	count := float32(len(embeddings))
	for i := 0; i < n; i++ {
		result[i] /= count
	}

	return result
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct float32
	var normA, normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// MatchInput matches input against the keyword index.
// Returns sorted expert names (by priority) and match confidence.
// MatchInput 将输入与关键词索引匹配，返回排序后的专家名称和置信度。
func (cm *CapabilityMap) MatchInput(input string) ([]string, float64) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.keywordIndex == nil {
		return nil, 0
	}

	inputLower := strings.ToLower(input)
	matchedExperts := make(map[string]int) // expert -> match score

	// Check exclude patterns first
	for _, ex := range cm.keywordIndex.excludes {
		if ex.MatchString(input) {
			return nil, 0 // Excluded
		}
	}

	// Match keywords (each keyword = 1 point)
	for kw, experts := range cm.keywordIndex.keywords {
		if strings.Contains(inputLower, kw) {
			for _, exp := range experts {
				matchedExperts[exp]++
			}
		}
	}

	// Match patterns (each pattern = 2 points, higher weight)
	for pat, experts := range cm.keywordIndex.patterns {
		if pat.MatchString(input) {
			for _, exp := range experts {
				matchedExperts[exp] += 2
			}
		}
	}

	if len(matchedExperts) == 0 {
		return nil, 0
	}

	// Sort by score, then by priority
	var results []string
	for exp := range matchedExperts {
		results = append(results, exp)
	}
	sort.Slice(results, func(i, j int) bool {
		expI, expJ := results[i], results[j]
		if matchedExperts[expI] != matchedExperts[expJ] {
			return matchedExperts[expI] > matchedExperts[expJ] // Higher score first
		}
		// Then by priority
		prioI := cm.keywordIndex.priorities[expI]
		prioJ := cm.keywordIndex.priorities[expJ]
		return prioI > prioJ
	})

	// Calculate confidence (normalize by max possible score)
	maxScore := 0
	for _, score := range matchedExperts {
		if score > maxScore {
			maxScore = score
		}
	}
	confidence := float64(maxScore) / 5.0 // Normalize: assume 5 is high score
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.3 {
		return nil, 0 // Too low confidence
	}

	return results, confidence
}

// MatchSemantic matches input against the semantic index using embeddings.
// This is used for Layer 3 semantic routing when Layer 2 rule matching fails.
// MatchSemantic 使用 embedding 与语义索引匹配，用于 Layer 3 语义路由。
func (cm *CapabilityMap) MatchSemantic(ctx context.Context, input string) ([]string, float64) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.semanticIndex == nil || !cm.semanticIndex.initialized {
		return nil, 0
	}

	if cm.embeddingProvider == nil {
		slog.Warn("no embedding provider available for semantic matching")
		return nil, 0
	}

	// Generate embedding for input
	inputEmbedding, err := cm.embeddingProvider.Embed(ctx, input)
	if err != nil {
		slog.Warn("failed to embed input for semantic matching",
			"input", input,
			"error", err)
		return nil, 0
	}

	// Calculate similarity with each expert's embedding
	bestExpert := ""
	bestSimilarity := float32(0)

	for expertName, expertEmbedding := range cm.semanticIndex.expertEmbeddings {
		similarity := cosineSimilarity(inputEmbedding, expertEmbedding)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestExpert = expertName
		}
	}

	// Check threshold
	if bestSimilarity < cm.semanticIndex.threshold {
		return nil, 0
	}

	return []string{bestExpert}, float64(bestSimilarity)
}

// IdentifyAgent resolves an agent name to its canonical ID.
// Supports exact match, fuzzy match (partial match), and title-based match.
// This is used by HandoffHandler to validate and normalize agent names.
// IdentifyAgent 将代理名称解析为规范 ID，支持精确匹配、模糊匹配和标题匹配。
func (cm *CapabilityMap) IdentifyAgent(name string) string {
	if name == "" {
		return ""
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.keywordIndex == nil {
		// Fallback: check experts map directly
		nameLower := strings.ToLower(name)
		for expertName := range cm.experts {
			if strings.EqualFold(expertName, nameLower) {
				return expertName
			}
		}
		return ""
	}

	nameLower := strings.ToLower(name)

	// 1. Exact match on name
	if _, ok := cm.keywordIndex.priorities[nameLower]; ok {
		return nameLower
	}

	// 2. Fuzzy match: partial match (e.g., "memo" matches "memo")
	// or title contains the name
	for expertName := range cm.keywordIndex.priorities {
		expertLower := strings.ToLower(expertName)
		if strings.Contains(expertLower, nameLower) || strings.Contains(nameLower, expertLower) {
			return expertName
		}
		// Also check title
		if title, ok := cm.keywordIndex.titles[expertName]; ok {
			if strings.Contains(strings.ToLower(title), nameLower) {
				return expertName
			}
		}
	}

	// 3. Fallback to experts map
	for expertName := range cm.experts {
		if strings.EqualFold(expertName, nameLower) {
			return expertName
		}
	}

	return ""
}

// GetAllExpertNames returns all registered expert names.
func (cm *CapabilityMap) GetAllExpertNames() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	names := make([]string, 0, len(cm.experts))
	for name := range cm.experts {
		names = append(names, name)
	}
	return names
}

// FindExpertsByCapability returns all experts that provide the given capability.
// FindExpertsByCapability 返回提供指定能力的所有专家。
func (cm *CapabilityMap) FindExpertsByCapability(capability string) []*ExpertInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	normalizedCap := cm.normalizeCapability(capability)
	if normalizedCap == "" {
		return nil
	}

	experts, ok := cm.capabilityToExperts[Capability(normalizedCap)]
	if !ok {
		return nil
	}

	// Return a copy to avoid external mutation
	result := make([]*ExpertInfo, len(experts))
	copy(result, experts)
	return result
}

// FindAlternativeExperts returns all experts that provide the given capability,
// excluding the specified expert. This is useful for finding fallback experts.
// FindAlternativeExperts 返回提供指定能力的所有专家，但排除指定的专家。
// 这在寻找备用专家时很有用。
func (cm *CapabilityMap) FindAlternativeExperts(capability string, excludeExpert string) []*ExpertInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	normalizedCap := cm.normalizeCapability(capability)
	if normalizedCap == "" {
		return nil
	}

	experts, ok := cm.capabilityToExperts[Capability(normalizedCap)]
	if !ok {
		return nil
	}

	// Filter out the excluded expert
	var result []*ExpertInfo
	for _, expert := range experts {
		if expert.Name != excludeExpert {
			result = append(result, expert)
		}
	}

	return result
}

// IdentifyCapabilities identifies capabilities from a text based on registered triggers.
// IdentifyCapabilities 根据注册的触发器从文本中识别能力。
func (cm *CapabilityMap) IdentifyCapabilities(text string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	normalizedText := strings.ToLower(text)
	var matchedCapabilities []string
	seen := make(map[string]bool)

	for trigger, caps := range cm.keywordToCapabilities {
		if cm.matchesTrigger(normalizedText, trigger) {
			for _, cap := range caps {
				capStr := string(cap)
				if !seen[capStr] {
					seen[capStr] = true
					matchedCapabilities = append(matchedCapabilities, capStr)
				}
			}
		}
	}

	return matchedCapabilities
}

// matchesTrigger checks if the trigger exists in the text.
// For ASCII triggers, it enforces word boundaries to avoid partial matches.
// For non-ASCII triggers (e.g. Chinese), it uses simple containment.
func (cm *CapabilityMap) matchesTrigger(text, trigger string) bool {
	// 1. Basic containment check
	idx := strings.Index(text, trigger)
	if idx == -1 {
		return false
	}

	// 2. If trigger contains non-ASCII characters (e.g. Chinese), containment is sufficient
	if isNonASCII(trigger) {
		return true
	}

	// 3. For ASCII triggers, verify word boundaries
	// We must check all occurrences
	for idx != -1 {
		// Check left boundary
		leftOk := (idx == 0) || !isWordChar(text[idx-1])

		// Check right boundary
		end := idx + len(trigger)
		rightOk := (end == len(text)) || !isWordChar(text[end])

		if leftOk && rightOk {
			return true
		}

		// Find next occurrence
		next := strings.Index(text[idx+1:], trigger)
		if next == -1 {
			break
		}
		idx += 1 + next
	}

	return false
}

func isNonASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return true
		}
	}
	return false
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// GetAllExperts returns all registered experts.
// GetAllExperts 返回所有已注册的专家。
func (cm *CapabilityMap) GetAllExperts() []*ExpertInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	experts := make([]*ExpertInfo, 0, len(cm.experts))
	for _, expert := range cm.experts {
		experts = append(experts, expert)
	}
	return experts
}

// normalizeCapability normalizes a capability string for consistent lookup.
// It converts the capability to lowercase and trims whitespace.
// normalizeCapability 标准化能力字符串以进行一致的查找。
// 它将能力转换为小写并去除空白。
func (cm *CapabilityMap) normalizeCapability(cap string) string {
	return strings.ToLower(strings.TrimSpace(cap))
}

// GetKeywordsForExpert returns all trigger keywords associated with a specific expert.
// This enables HILT feedback to adjust weights for specific keywords.
// GetKeywordsForExpert 返回与指定专家关联的所有触发关键词，用于 HILT 反馈权重调整。
func (cm *CapabilityMap) GetKeywordsForExpert(expertName string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.keywordIndex == nil {
		return nil
	}

	expertLower := strings.ToLower(expertName)
	var keywords []string
	seen := make(map[string]bool)

	// Reverse lookup: find all keywords that map to this expert
	for kw, experts := range cm.keywordIndex.keywords {
		for _, exp := range experts {
			if strings.EqualFold(exp, expertLower) {
				if !seen[kw] {
					seen[kw] = true
					keywords = append(keywords, kw)
				}
				break
			}
		}
	}

	return keywords
}
