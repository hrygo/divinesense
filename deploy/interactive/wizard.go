// Package interactive provides interactive deployment wizard for DivineSense
package interactive

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// WizardConfig holds all configuration for the deployment
type WizardConfig struct {
	// Deployment mode
	Mode string `json:"mode"` // docker | binary

	// System paths
	InstallDir string `json:"installDir"`
	ConfigDir  string `json:"configDir"`

	// Service configuration
	Port           int  `json:"port"`
	AutoStart      bool `json:"autoStart"`
	WithBackups    bool `json:"withBackups"`
	EnableFirewall bool `json:"enableFirewall"`

	// Database configuration
	DbType     string `json:"dbType"` // docker | system | remote
	DbPort     int    `json:"dbPort"`
	DbHost     string `json:"dbHost"`
	DbName     string `json:"dbName"`
	DbUser     string `json:"dbUser"`
	DbPassword string `json:"dbPassword"`

	// AI features
	EnableAI       bool              `json:"enableAI"`
	AIProvider     string            `json:"aiProvider"` // siliconflow | deepseek | openai
	EmbeddingModel string            `json:"embeddingModel"`
	ChatModel      string            `json:"chatModel"`
	APIKeys        map[string]string `json:"apiKeys"`

	// Geek Mode
	EnableGeekMode    bool   `json:"enableGeekMode"`
	InstallClaudeCode bool   `json:"installClaudeCode"`
	Workdir           string `json:"workdir"`

	// Evolution Mode
	EnableEvolution bool `json:"enableEvolution"`
	AdminOnly       bool `json:"adminOnly"`

	// User information
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`

	// Installation source
	SourceRepo string `json:"sourceRepo"`
	Branch     string `json:"branch"`
	Version    string `json:"version"`

	// Advanced options
	InstallNode     bool `json:"installNode"`
	InstallDocker   bool `json:"installDocker"`
	InstallPostgres bool `json:"installPostgres"`
	AutoConfigure   bool `json:"autoConfigure"`
}

// DefaultWizardConfig returns sensible defaults
func DefaultWizardConfig() WizardConfig {
	return WizardConfig{
		Mode:            "binary",
		InstallDir:      "/opt/divinesense",
		ConfigDir:       "/etc/divinesense",
		Port:            5230,
		AutoStart:       true,
		WithBackups:     true,
		EnableFirewall:  true,
		DbType:          "docker", // Use Docker PostgreSQL by default
		DbPort:          25432,
		DbHost:          "localhost",
		DbName:          "divinesense",
		DbUser:          "divinesense",
		EnableAI:        true,
		AIProvider:      "siliconflow",
		EmbeddingModel:  "BAAI/bge-m3",
		ChatModel:       "deepseek-chat",
		APIKeys:         make(map[string]string),
		EnableGeekMode:  true,
		Workdir:         "/opt/divinesense/data",
		EnableEvolution: false,
		AdminOnly:       true,
		SourceRepo:      "https://github.com/hrygo/divinesense.git",
		Branch:          "main",
		Version:         "latest",
		AutoConfigure:   true,
	}
}

// Wizard handles the interactive deployment process
type Wizard struct {
	config    WizardConfig
	reader    *bufio.Reader
	writer    *bufio.Writer
	style     lipgloss.Style
	width     int
	height    int
	isTty     bool
	detectors *Detectors
	step      int
}

// Detectors holds system detection results
type Detectors struct {
	hasDocker   bool
	hasPostgres bool
	hasNode     bool
	hasNpm      bool
	hasGit      bool
	hasClaude   bool
	isRoot      bool
	osInfo      OSInfo
	portsInUse  map[int]string
	pkgManager  string
}

// OSInfo holds detected OS information
type OSInfo struct {
	ID         string
	Version    string
	PkgManager string
	InitSystem string
	Arch       string
}

// NewWizard creates a new deployment wizard
func NewWizard() *Wizard {
	return &Wizard{
		config:    DefaultWizardConfig(),
		reader:    bufio.NewReader(os.Stdin),
		writer:    bufio.NewWriter(os.Stdout),
		style:     lipgloss.NewStyle(),
		detectors: NewDetectors(),
		step:      0,
	}
}

// Run starts the interactive wizard
func (w *Wizard) Run() error {
	w.initTerminal()
	defer func() {
		if err := w.writer.Flush(); err != nil {
			// Can't use w.println here as writer might be in bad state
			fmt.Fprintf(os.Stderr, "Warning: failed to flush output: %v\n", err)
		}
	}()

	w.printWelcome()

	// Step 1: System check
	if err := w.stepSystemCheck(); err != nil {
		return err
	}
	w.step++

	// Step 2: Deployment mode selection
	if err := w.stepModeSelection(); err != nil {
		return err
	}
	w.step++

	// Step 3: Database configuration
	if err := w.stepDatabase(); err != nil {
		return err
	}
	w.step++

	// Step 4: AI configuration
	if err := w.stepAI(); err != nil {
		return err
	}
	w.step++

	// Step 5: Geek Mode
	if err := w.stepGeekMode(); err != nil {
		return err
	}
	w.step++

	// Step 6: Evolution Mode (optional)
	if err := w.stepEvolutionMode(); err != nil {
		return err
	}
	w.step++

	// Step 7: Admin account
	if err := w.stepAdminAccount(); err != nil {
		return err
	}
	w.step++

	// Step 8: Summary and confirmation
	if err := w.stepSummary(); err != nil {
		return err
	}
	w.step++

	// Step 9: Installation
	if err := w.stepInstallation(); err != nil {
		return err
	}

	w.printCompletion()
	return nil
}

func (w *Wizard) initTerminal() {
	w.isTty = isTerminal(os.Stdin)
	fd := int(os.Stdin.Fd())
	width, height, err := term.GetSize(fd)
	if err == nil {
		w.width = width
		w.height = height
	}
	if w.width == 0 {
		w.width = 80
	}
	if w.height == 0 {
		w.height = 24
	}
}

func (w *Wizard) printWelcome() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Margin(1, 2)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Margin(1, 2)

	w.println(titleStyle.Render("╔════════════════════════════════════════════════════════════╗"))
	w.println(titleStyle.Render("  DivineSense 交互式部署向导 v4.0"))
	w.println(titleStyle.Render("╠════════════════════════════════════════════════════════════╣"))
	w.println(borderStyle.Render("  本向导将引导您完成 DivineSense 的完整部署配置"))
	w.println(borderStyle.Render(""))

	// Show system info
	osInfo := w.detectors.osInfo
	w.println(borderStyle.Render(fmt.Sprintf("  检测到系统: %s %s (%s)",
		osInfo.ID, osInfo.Version, osInfo.Arch)))
	w.println(borderStyle.Render(""))

	if w.detectors.isRoot {
		w.println(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("  ✓ 以 root 权限运行"))
	} else {
		w.println(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("  ⚠ 非 root 权限，某些操作需要 sudo"))
	}

	w.println(borderStyle.Render(""))
}

func (w *Wizard) stepSystemCheck() error {
	w.printHeader("1. 系统检查与依赖检测")

	// Check system resources
	resourcesOK, missing := w.checkSystemResources()
	if !resourcesOK {
		w.printError("系统资源不足，无法继续部署: %s", strings.Join(missing, ", "))
		return fmt.Errorf("insufficient resources")
	}

	// Show detection results
	w.println(w.style.Foreground(lipgloss.Color("86")).Render("检测依赖:"))

	detections := []struct {
		name      string
		available bool
		action    string
	}{
		{"Docker", w.detectors.hasDocker, w.getAction("install", "docker")},
		{"PostgreSQL", w.detectors.hasPostgres, w.getAction("install", "PostgreSQL")},
		{"Node.js", w.detectors.hasNode, w.getAction("install", "Node.js")},
		{"npm", w.detectors.hasNpm, w.getAction("install", "npm")},
		{"Git", w.detectors.hasGit, w.getAction("install", "git")},
		{"Claude Code CLI", w.detectors.hasClaude, w.getAction("install", "Claude Code CLI")},
	}

	for _, d := range detections {
		status := "✓"
		color := "42"
		action := "已安装"
		if !d.available {
			status = "✗"
			color = "208"
			action = d.action
		}
		w.println(fmt.Sprintf("  [%s] %-20s %s",
			w.style.Foreground(lipgloss.Color(color)).Render(status),
			d.name,
			action))
	}

	// Check ports
	w.println("")
	w.println(w.style.Foreground(lipgloss.Color("86")).Render("端口检查:"))
	ports := []int{5230, 25432}
	for _, port := range ports {
		if reason, ok := w.detectors.portsInUse[port]; ok {
			w.println(fmt.Sprintf("  [✗] 端口 %d 被 %s 占用", port, reason))
		} else {
			w.println(fmt.Sprintf("  [✓] 端口 %d 可用", port))
		}
	}

	// Offer to install missing dependencies
	w.println("")
	if !w.confirm("是否继续?", true) {
		return fmt.Errorf("user cancelled")
	}

	return nil
}

func (w *Wizard) stepModeSelection() error {
	w.printHeader("2. 部署模式选择")

	// Display mode comparison table
	w.println(w.style.Foreground(lipgloss.Color("86")).Render("\n模式对比:"))
	headers := []string{"特性", "Docker 模式", "二进制模式"}
	rows := [][]string{
		{"Geek Mode 支持", "⚠️ 需额外配置", "✅ 原生支持"},
		{"Evolution Mode 支持", "❌ 不支持", "✅ 原生支持"},
		{"资源占用", "高 (容器开销)", "低"},
		{"启动速度", "慢", "快"},
		{"更新方式", "重建镜像", "替换二进制"},
		{"适用场景", "快速部署/测试", "生产环境/Geek Mode"},
	}

	t := table.New().
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			}
			return lipgloss.NewStyle()
		})

	for _, row := range rows {
		t.Row(row...)
	}
	t.Render()

	// Mode selection
	w.println("")
	defaultMode := "binary"
	if w.detectors.hasDocker && !w.detectors.hasPostgres {
		defaultMode = "docker"
	}

	choice, err := w.promptSelect("选择部署模式 (1=Docker, 2=Binary)", []string{"docker", "binary"}, defaultMode)
	if err != nil {
		return err
	}
	w.config.Mode = choice

	w.println(fmt.Sprintf("已选择: %s 模式", choice))
	return nil
}

func (w *Wizard) stepDatabase() error {
	w.printHeader("3. 数据库配置")

	// Ask for PostgreSQL setup
	options := []string{"docker", "system", "remote"}

	defaultIdx := 0
	if w.detectors.hasPostgres {
		defaultIdx = 1
		w.printInfo("检测到系统已安装 PostgreSQL，建议使用系统安装")
	}

	choice, err := w.promptSelect("PostgreSQL 安装方式 (1=Docker, 2=系统, 3=远程)", options, options[defaultIdx])
	if err != nil {
		return err
	}
	w.config.DbType = choice

	switch choice {
	case "docker":
		w.println("  → Docker PostgreSQL 容器将自动部署")
		w.config.DbPort = 25432
		w.config.InstallDocker = true
	case "system":
		w.println("  → 将安装系统 PostgreSQL 包")
		w.config.InstallPostgres = true
		if !w.confirm("是否安装 PostgreSQL?", true) {
			return fmt.Errorf("user cancelled")
		}
	case "remote":
		w.println("  → 使用远程 PostgreSQL")
		host, err := w.promptInput("远程数据库主机 (host:port)", "localhost:5432", false)
		if err != nil {
			return err
		}
		w.config.DbHost = host
		w.config.DbType = "remote"
		// Parse host:port
		if parts := strings.Split(host, ":"); len(parts) == 2 {
			w.config.DbHost = parts[0]
			port, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("invalid port number: %s", parts[1])
			}
			if port <= 0 || port > 65535 {
				return fmt.Errorf("port out of range: %d", port)
			}
			w.config.DbPort = port
		}
	}

	// Database credentials
	if w.config.DbType != "remote" {
		if err := w.configureDatabaseCredentials(); err != nil {
			return err
		}
	}

	return nil
}

func (w *Wizard) stepAI() error {
	w.printHeader("4. AI 功能配置")

	// Ask if AI features should be enabled
	if !w.confirm("是否启用 AI 功能? (推荐)", true) {
		w.config.EnableAI = false
		return nil
	}

	w.config.EnableAI = true

	// AI Provider selection
	providers := []string{"siliconflow", "deepseek", "openai"}
	providerDesc := map[string]string{
		"siliconflow": "SiliconFlow (推荐 - 国内网络优化)",
		"deepseek":    "DeepSeek",
		"openai":      "OpenAI (官方)",
	}

	defaultProvider := "siliconflow"
	w.println("")
	for i, p := range providers {
		w.println(fmt.Sprintf("  [%d] %s", i+1, providerDesc[p]))
	}

	choice, err := w.promptSelect("选择 AI 提供商", providers, defaultProvider)
	if err != nil {
		return err
	}
	w.config.AIProvider = choice

	// API Keys
	w.println("")
	w.printInfo("需要配置 API Key 才能使用 AI 功能")

	var requiredKeys []string
	switch choice {
	case "siliconflow":
		requiredKeys = []string{"SILICONFLOW_API_KEY"}
	case "deepseek":
		requiredKeys = []string{"DEEPSEEK_API_KEY"}
	case "openai":
		requiredKeys = []string{"OPENAI_API_KEY"}
	}

	w.config.APIKeys = make(map[string]string)

	for _, key := range requiredKeys {
		w.printInfo(fmt.Sprintf("%s (输入不会显示):", key))
		input, err := w.readPassword()
		if err != nil {
			return fmt.Errorf("取消输入: %w", err)
		}
		if input != "" {
			w.config.APIKeys[key] = input
			w.printSuccess(fmt.Sprintf("  %s 已配置", key))
		}
	}

	// Model selection
	w.config.ChatModel = "deepseek-chat"
	if choice == "openai" {
		w.config.ChatModel = "gpt-4o-mini"
	}

	return nil
}

func (w *Wizard) stepGeekMode() error {
	w.printHeader("5. Geek Mode 配置")

	if !w.confirm("是否启用 Geek Mode? (代码助手功能)", true) {
		w.config.EnableGeekMode = false
		return nil
	}

	w.config.EnableGeekMode = true
	w.config.EnableAI = true // Geek Mode requires AI

	// Check Claude Code CLI
	if w.detectors.hasClaude {
		w.printSuccess("  Claude Code CLI 已安装")
	} else {
		w.printInfo("  Claude Code CLI 未安装")
		if w.confirm("是否自动安装 Claude Code CLI? (y/n)", true) {
			w.config.InstallClaudeCode = true
			w.printInfo("  安装指令将在部署脚本中执行")
		}
	}

	// Work directory
	w.printInfo("Geek Mode 工作目录 (Claude Code CLI 将在此目录执行代码)")
	defaultDir := "/opt/divinesense/data"

	workdir, err := w.promptInput("工作目录路径", defaultDir, false)
	if err != nil {
		return err
	}

	w.config.Workdir = workdir
	w.printSuccess(fmt.Sprintf("  工作目录: %s", workdir))

	return nil
}

func (w *Wizard) stepEvolutionMode() error {
	w.printHeader("6. Evolution Mode (高级)")

	// Evolution Mode requires Geek Mode first
	if !w.config.EnableGeekMode {
		w.printInfo("  Evolution Mode 需要 Geek Mode 先启用")
		return nil
	}

	w.printWarn("  Evolution Mode 允许 AI 修改 DivineSense 源代码")
	w.printInfo("  ⚠️ 这是一项高级功能，仅推荐给开发者使用")

	if !w.confirm("是否启用 Evolution Mode?", false) {
		w.config.EnableEvolution = false
		return nil
	}

	w.config.EnableEvolution = true
	w.config.EnableAI = true
	w.config.AdminOnly = true

	// Git repository check
	if !w.detectors.hasGit {
		w.printError("  未检测到 Git，无法使用 Evolution Mode")
		w.printInfo("  请先安装 Git: yum install git 或 apt install git")
		return fmt.Errorf("git not available")
	}

	// Working directory should be source root
	if !w.isGitRepo(w.config.InstallDir) {
		w.printWarn("  当前目录不是 Git 仓库")
		if !w.confirm("是否继续? Evolution Mode 将无法提交代码", false) {
			return fmt.Errorf("not a git repository")
		}
	}

	w.printSuccess("  Evolution Mode 配置完成")
	w.printInfo("  代码将提交为 Pull Request 供审查")

	return nil
}

func (w *Wizard) stepAdminAccount() error {
	w.printHeader("7. 管理员账户")

	// Only required for fresh installations
	if w.config.AutoConfigure {
		w.printInfo("  管理员账户将在首次访问时创建")
		return nil
	}

	username, err := w.promptInput("管理员用户名", "admin", false)
	if err != nil {
		return err
	}

	password, err := w.readPassword()
	if err != nil {
		return err
	}

	w.config.AdminUsername = username
	w.config.AdminPassword = password

	return nil
}

func (w *Wizard) stepSummary() error {
	w.printHeader("8. 配置摘要与确认")

	// Display configuration summary
	w.println("")
	w.println(w.style.Foreground(lipgloss.Color("86")).Render("配置摘要:"))

	summary := [][]string{
		{"部署模式", strings.ToUpper(w.config.Mode)},
		{"安装目录", w.config.InstallDir},
		{"配置目录", w.config.ConfigDir},
		{"服务端口", fmt.Sprintf(":%d", w.config.Port)},
		{"AI 功能", boolToYesNo(w.config.EnableAI)},
		{"Geek Mode", boolToYesNo(w.config.EnableGeekMode)},
		{"Evolution Mode", boolToYesNo(w.config.EnableEvolution)},
	}

	t := table.New().
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			}
			return lipgloss.NewStyle()
		})

	for _, row := range summary {
		t.Row(row...)
	}
	w.println(t.Render())

	// Show database info
	w.println("")
	w.println(w.style.Foreground(lipgloss.Color("86")).Render("数据库:"))
	dbInfo := [][]string{
		{"类型", w.config.DbType},
		{"主机", w.config.DbHost},
		{"端口", fmt.Sprintf("%d", w.config.DbPort)},
		{"数据库", w.config.DbName},
	}
	for _, row := range dbInfo {
		t.Row(row...)
	}
	w.println(t.Render())

	// Show AI provider info
	if w.config.EnableAI {
		w.println("")
		w.println(w.style.Foreground(lipgloss.Color("86")).Render("AI 配置:"))
		aiInfo := [][]string{
			{"提供商", w.config.AIProvider},
			{"模型", w.config.ChatModel},
			{"Embedding", w.config.EmbeddingModel},
		}
		for _, row := range aiInfo {
			t.Row(row...)
		}
		w.println(t.Render())

		// Show configured keys (without values)
		keys := getNonEmptyKeys(w.config.APIKeys)
		if len(keys) > 0 {
			w.println("  已配置 API Keys: " + strings.Join(keys, ", "))
		}
	}

	// Show Geek/Evolution specific info
	if w.config.EnableGeekMode || w.config.EnableEvolution {
		w.println("")
		w.println(w.style.Foreground(lipgloss.Color("86")).Render("高级模式:"))
		advancedInfo := [][]string{
			{"Geek Mode", boolToYesNo(w.config.EnableGeekMode)},
			{"工作目录", w.config.Workdir},
			{"Claude Code CLI", boolToYesNo(w.detectors.hasClaude)},
			{"Evolution Mode", boolToYesNo(w.config.EnableEvolution)},
		}
		for _, row := range advancedInfo {
			t.Row(row...)
		}
		w.println(t.Render())
	}

	// Final confirmation
	w.println("")
	if !w.confirm("确认开始安装? 配置将写入系统", true) {
		return fmt.Errorf("user cancelled")
	}

	return nil
}

func (w *Wizard) stepInstallation() error {
	w.printHeader("9. 开始安装")

	// Create install script based on configuration
	installScript := w.generateInstallScript()

	scriptPath := "/tmp/divinesense-install.sh"
	if err := os.WriteFile(scriptPath, []byte(installScript), 0755); err != nil {
		return fmt.Errorf("failed to write install script: %w", err)
	}
	// Clean up temporary script after execution
	defer func() {
		if err := os.Remove(scriptPath); err != nil {
			// Only log if file still exists (might have been manually cleaned)
			if _, statErr := os.Stat(scriptPath); statErr == nil {
				w.printWarn(fmt.Sprintf("无法清理临时脚本 %s: %v", scriptPath, err))
			}
		}
	}()

	w.printSuccess("安装脚本已生成: " + scriptPath)

	// Ask for confirmation before running
	w.println("")
	w.printWarn("即将执行安装脚本，需要 root 权限")
	if !w.confirm("继续执行安装? (y/n)", true) {
		return fmt.Errorf("user cancelled")
	}

	// Execute the script
	w.println("")
	w.println(w.style.Foreground(lipgloss.Color("86")).Render("执行中..."))

	cmd := exec.Command("sudo", "bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	w.printSuccess("安装完成!")
	return nil
}

func (w *Wizard) printCompletion() {
	w.println("")
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	w.println(successStyle.Render("╔════════════════════════════════════════════════════════════╗"))
	w.println(successStyle.Render("║  DivineSense 部署成功！                                           ║"))
	w.println(successStyle.Render("╠════════════════════════════════════════════════════════════╣"))
	w.println(successStyle.Render("║  访问地址:                                                       ║"))

	serverIP := w.getServerIP()
	w.println(successStyle.Render(fmt.Sprintf("║  http://%s:%d                                             ║", serverIP, w.config.Port)))
	w.println(successStyle.Render("║                                                                  ║"))
	w.println(successStyle.Render("║  管理命令:                                                       ║"))
	w.println(successStyle.Render("║  cd /opt/divinesense && ./deploy-binary.sh status              ║"))
	w.println(successStyle.Render("║  journalctl -u divinesense -f                                  ║"))
	w.println(successStyle.Render("║                                                                  ║"))
	w.println(successStyle.Render("╚════════════════════════════════════════════════════════════╝"))
	w.println("")

	if w.config.EnableGeekMode {
		w.printInfo("Geek Mode 已启用")
		if w.detectors.hasClaude {
			w.printSuccess("  Claude Code CLI: " + getClaudeVersion())
		}
	}

	if w.config.EnableEvolution {
		w.printWarn("Evolution Mode 已启用 - 仅管理员可用")
	}
}

// Helper methods

func (w *Wizard) printHeader(step string) {
	w.println("")
	w.println(w.style.Foreground(lipgloss.Color("42")).Render(fmt.Sprintf("[%d/9] %s", w.step+1, step)))
	w.println(w.style.Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", 60)))
}

func (w *Wizard) printError(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	w.println(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("  ✗ " + formatted))
}

func (w *Wizard) printWarn(msg string) {
	w.println(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("  ⚠ " + msg))
}

func (w *Wizard) printInfo(msg string) {
	w.println("  " + msg)
}

func (w *Wizard) printSuccess(msg string) {
	w.println(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("  ✓ " + msg))
}

func (w *Wizard) println(text string) {
	// Ignore write errors for interactive output - if stdout fails,
	// there's not much we can do in a TUI context
	_, _ = w.writer.WriteString(text + "\n") //nolint:errcheck // write to stdout in TUI context, failure is non-critical
}

func (w *Wizard) confirm(msg string, defaultYes bool) bool {
	if !w.isTty {
		return defaultYes
	}

	for {
		prompt := msg
		if defaultYes {
			prompt += " [Y/n]: "
		} else {
			prompt += " [y/N]: "
		}
		w.printInfo(prompt)

		input, err := w.reader.ReadString('\n')
		if err != nil {
			return defaultYes
		}

		input = strings.TrimSpace(input)
		input = strings.ToLower(input)

		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		case "":
			return defaultYes
		default:
			w.printWarn("请输入 y 或 n")
		}
	}
}

func (w *Wizard) promptSelect(label string, options []string, defaultVal string) (string, error) {
	if !w.isTty {
		return defaultVal, nil
	}

	for {
		w.println("")
		for i, opt := range options {
			w.println(fmt.Sprintf("  [%d] %s", i+1, opt))
		}
		w.printInfo(label + ": ")

		input, err := w.reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(options) {
			w.printWarn("请输入有效选项 (1-" + fmt.Sprintf("%d", len(options)) + ")")
			continue
		}
		return options[idx-1], nil
	}
}

func (w *Wizard) promptInput(label, defaultValue string, required bool) (string, error) {
	prompt := label
	if defaultValue != "" {
		prompt += fmt.Sprintf(" [%s]", defaultValue)
	}
	prompt += ": "

	w.printInfo(prompt)
	input, err := w.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

func (w *Wizard) readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !w.isTty {
		return "", fmt.Errorf("not a terminal")
	}

	// Get current terminal state
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}

	// Ensure terminal is restored even on panic
	defer func() {
		if restoreErr := term.Restore(fd, oldState); restoreErr != nil {
			// Last-ditch effort to restore terminal using stty
			_ = exec.Command("stty", "sane").Run() //nolint:errcheck // best-effort terminal restore, failure acceptable
		}
	}()

	// Read password byte by byte
	var password []byte
	buf := make([]byte, 1)
	const maxPasswordLength = 1000

	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 && buf[0] == '\n' {
			break
		}
		if err != nil {
			return "", err
		}
		if n > 0 && buf[0] != '\r' {
			if len(password) < maxPasswordLength {
				//nolint:gosec // G602: buf[0] access is protected by n > 0 check above
				password = append(password, buf[0])
			}
		}
		// Handle EOF (n == 0) - prevent infinite loop
		if n == 0 {
			break
		}
	}

	w.println("") // Newline after password input
	return string(password), nil
}

func (w *Wizard) checkSystemResources() (bool, []string) {
	var missing []string

	// Check RAM
	totalMem := w.detectors.GetTotalMemoryMB()
	if totalMem < 1800 {
		missing = append(missing, "内存不足 (需要 1800MB)")
	}

	// Check disk
	availDisk := w.detectors.GetAvailableDiskMB()
	if availDisk < 4096 {
		missing = append(missing, "磁盘空间不足 (需要 4096MB)")
	}

	return len(missing) == 0, missing
}

func (w *Wizard) configureDatabaseCredentials() error {
	w.println("")
	w.printInfo("配置数据库凭据")

	// Database name
	dbName, err := w.promptInput("数据库名称", "divinesense", false)
	if err != nil {
		return err
	}
	w.config.DbName = dbName

	// Database user
	dbUser, err := w.promptInput("数据库用户", "divinesense", false)
	if err != nil {
		return err
	}
	w.config.DbUser = dbUser

	// Database password (auto-generate by default)
	if w.confirm("自动生成数据库密码? (推荐) (y/n)", true) {
		password := generatePassword(16)
		w.config.DbPassword = password
		w.printSuccess("  数据库密码已生成")
	} else {
		w.printInfo("数据库密码 (输入不会显示): ")
		password, err := w.readPassword()
		if err != nil {
			return err
		}
		if len(password) < 8 {
			return fmt.Errorf("密码至少需要 8 个字符")
		}
		w.config.DbPassword = password
	}

	return nil
}

func (w *Wizard) isGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func (w *Wizard) getServerIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}

func (w *Wizard) getAction(action, target string) string {
	switch w.detectors.osInfo.PkgManager {
	case "apt":
		return fmt.Sprintf("sudo apt install %s", target)
	case "yum":
		return fmt.Sprintf("sudo yum install %s", target)
	case "pacman":
		return fmt.Sprintf("sudo pacman -S %s", target)
	default:
		return fmt.Sprintf("请安装 %s", target)
	}
}

func (w *Wizard) generateInstallScript() string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# DivineSense 自动生成的安装脚本\n")
	sb.WriteString(fmt.Sprintf("# 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("# 配置: %s mode, port %d\n\n", w.config.Mode, w.config.Port))
	sb.WriteString("set -e\n\n")

	// Color definitions
	sb.WriteString("RED='\\033[0;31m'\n")
	sb.WriteString("GREEN='\\033[0;32m'\n")
	sb.WriteString("YELLOW='\\033[1;33m'\n")
	sb.WriteString("BLUE='\\033[0;34m'\n")
	sb.WriteString("CYAN='\\033[0;36m'\n")
	sb.WriteString("NC='\\033[0m'\n\n")

	// Logging functions
	sb.WriteString("log_info() { echo -e \"${BLUE}[INFO]${NC} $1\"; }\n")
	sb.WriteString("log_success() { echo -e \"${GREEN}[OK]${NC} $1\"; }\n")
	sb.WriteString("log_warn() { echo -e \"${YELLOW}[WARN]${NC} $1\"; }\n")
	sb.WriteString("log_error() { echo -e \"${RED}[ERROR]${NC} $1\"; }\n")
	sb.WriteString("log_step() { echo -e \"${CYAN}[STEP]${NC} $1\"; }\n\n")

	// Print banner
	sb.WriteString("print_banner() {\n")
	sb.WriteString("    echo -e \"${CYAN}╔════════════════════════════════════════════════════════════╗${NC}\"\n")
	sb.WriteString("    echo -e \"${CYAN}║${NC}  ${GREEN}DivineSense 自动安装${NC}                                        ${CYAN}║${NC}\"\n")
	sb.WriteString("    echo -e \"${CYAN}╠════════════════════════════════════════════════════════════╣${NC}\"\n")
	modeUpper := strings.ToUpper(w.config.Mode)
	sb.WriteString(fmt.Sprintf("    echo -e \"${CYAN}║${NC}  模式: ${YELLOW}%s${NC}                                       ${CYAN}║${NC}\"\n", modeUpper))
	sb.WriteString("    echo -e \"${CYAN}╚════════════════════════════════════════════════════════════╝${NC}\"\n")
	sb.WriteString("    echo \"\"\n")
	sb.WriteString("}\n\n")

	// Export configuration
	sb.WriteString("# 配置环境变量\n")
	sb.WriteString(fmt.Sprintf("export DEPLOY_MODE=%q\n", w.config.Mode))
	sb.WriteString(fmt.Sprintf("export INSTALL_DIR=%q\n", w.config.InstallDir))
	sb.WriteString(fmt.Sprintf("export CONFIG_DIR=%q\n", w.config.ConfigDir))
	sb.WriteString(fmt.Sprintf("export PORT=%d\n", w.config.Port))
	sb.WriteString(fmt.Sprintf("export DB_TYPE=%q\n", w.config.DbType))
	sb.WriteString(fmt.Sprintf("export DB_PORT=%d\n", w.config.DbPort))
	sb.WriteString(fmt.Sprintf("export DB_HOST=%q\n", w.config.DbHost))
	sb.WriteString(fmt.Sprintf("export DB_NAME=%q\n", w.config.DbName))
	sb.WriteString(fmt.Sprintf("export DB_USER=%q\n", w.config.DbUser))
	sb.WriteString(fmt.Sprintf("export DB_PASSWORD=%q\n", w.config.DbPassword))
	sb.WriteString(fmt.Sprintf("export ENABLE_AI=%s\n", boolToStr(w.config.EnableAI)))
	sb.WriteString(fmt.Sprintf("export ENABLE_GEEK=%s\n", boolToStr(w.config.EnableGeekMode)))
	sb.WriteString(fmt.Sprintf("export ENABLE_EVOLUTION=%s\n", boolToStr(w.config.EnableEvolution)))
	sb.WriteString(fmt.Sprintf("export WORKDIR=%q\n", w.config.Workdir))
	sb.WriteString(fmt.Sprintf("export ADMIN_ONLY=%s\n", boolToStr(w.config.AdminOnly)))
	sb.WriteString("\n")

	// API Keys
	if w.config.EnableAI {
		sb.WriteString("# API Keys\n")
		for key, value := range w.config.APIKeys {
			if value != "" {
				sb.WriteString(fmt.Sprintf("export %s=%q\n", key, value))
			}
		}
		sb.WriteString("\n")
	}

	// Main installation
	sb.WriteString("# 主安装流程\n")
	sb.WriteString("main() {\n")
	sb.WriteString("    print_banner\n")
	sb.WriteString("    log_step \"下载 DivineSense 安装脚本...\"\n\n")

	sb.WriteString("    INSTALL_URL=\"https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh\"\n")
	sb.WriteString("    cd /tmp\n")

	if w.config.Mode == "binary" {
		sb.WriteString("    curl -fsSL \"$INSTALL_URL\" -o install.sh\n")
		sb.WriteString(fmt.Sprintf("    bash install.sh --mode=%s\n", w.config.Mode))
	} else {
		sb.WriteString("    curl -fsSL \"$INSTALL_URL\" | bash\n")
	}

	sb.WriteString("\n    log_success \"安装完成!\"\n")
	sb.WriteString("    show_access_info\n")
	sb.WriteString("}\n\n")

	// Show access info
	sb.WriteString("show_access_info() {\n")
	sb.WriteString("    local ip=$(hostname -I | awk '{print $1}')\n")
	sb.WriteString("    echo \"\"\n")
	sb.WriteString("    echo -e \"${GREEN}访问地址:${NC} http://${ip}:5230\"\n")
	sb.WriteString("    echo \"\"\n")
	sb.WriteString("}\n\n")

	sb.WriteString("main \"$@\"\n")

	return sb.String()
}

// NewDetectors creates a new detector instance
func NewDetectors() *Detectors {
	d := &Detectors{
		portsInUse: make(map[int]string),
	}
	d.DetectOS()
	d.detectAll()
	return d
}

func (d *Detectors) DetectOS() OSInfo {
	if f, err := os.Open("/etc/os-release"); err == nil {
		defer f.Close() //nolint:errcheck // read-only file close, failure is acceptable
		var id, version string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "ID=") {
				id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			} else if strings.HasPrefix(line, "VERSION_ID=") {
				version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
		pkgManager := "unknown"
		switch id {
		case "centos", "rhel", "rocky", "almalinux":
			pkgManager = "yum"
		case "debian", "ubuntu":
			pkgManager = "apt"
		case "arch", "manjaro":
			pkgManager = "pacman"
		}
		d.pkgManager = pkgManager
		d.osInfo = OSInfo{ID: id, Version: version, PkgManager: pkgManager}
		return d.osInfo
	}
	return OSInfo{ID: "unknown", PkgManager: "unknown"}
}

func (d *Detectors) detectAll() {
	d.isRoot = os.Geteuid() == 0

	// Check for Docker
	_, err := exec.LookPath("docker")
	d.hasDocker = err == nil

	// Check for PostgreSQL
	_, err = exec.LookPath("psql")
	d.hasPostgres = err == nil

	// Check for Node.js
	_, err = exec.LookPath("node")
	d.hasNode = err == nil

	// Check for npm
	_, err = exec.LookPath("npm")
	d.hasNpm = err == nil

	// Check for Git
	_, err = exec.LookPath("git")
	d.hasGit = err == nil

	// Check for Claude Code CLI
	_, err = exec.LookPath("claude")
	d.hasClaude = err == nil

	// Check ports
	ports := []int{5230, 25432}
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", port), 2*time.Second)
		if err == nil {
			conn.Close() //nolint:errcheck // connection test close, failure is acceptable
			d.portsInUse[port] = "端口被占用"
		}
	}
}

func (d *Detectors) GetTotalMemoryMB() int {
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseInt(fields[len(fields)-2], 10, 64); err == nil {
						return int(kb / 1024)
					}
				}
			}
		}
	}
	return 0
}

func (d *Detectors) GetAvailableDiskMB() int {
	cmd := exec.Command("df", "-m", "/opt")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(output))
	if len(fields) >= 4 {
		avail, err := strconv.Atoi(fields[len(fields)-3])
		if err != nil {
			return 0
		}
		return avail
	}
	return 0
}

// Utility functions

func isTerminal(f *os.File) bool {
	fd := int(f.Fd())
	return term.IsTerminal(fd)
}

func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback to simpler random generation
			n.Int64()
		}
		result[i] = charset[n.Int64()%int64(len(charset))]
	}
	return string(result)
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

func getNonEmptyKeys(m map[string]string) []string {
	var keys []string
	for k, v := range m {
		if v != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func getClaudeVersion() string {
	if out, err := exec.Command("claude", "--version").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 3 {
			return parts[len(parts)-1]
		}
	}
	return "未安装"
}

// Run starts the wizard
func Run() error {
	w := NewWizard()
	return w.Run()
}
