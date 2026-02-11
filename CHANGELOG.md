# Changelog

All notable changes to this project will be documented in this file.

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范：
- **Major (主版本号)**：不兼容的 API 变更
- **Minor (次版本号)**：向下兼容的新功能
- **Patch (补丁号)**：向下兼容的问题修复

---

## [Unreleased]

### Added
- （待添加）

### Changed
- （待添加）

### Fixed
- （待添加）

---

## [v0.98.0] - 2026-02-10

### Added
- **Memo Editor Redesign (#124)**: Complete UI overhaul with bottom-positioned input
  - New `FixedEditor` component with responsive PC/Mobile layouts
  - `MemoBlockV2` with Fluid Card design and swipe gestures
  - `HeroSection` with inline search and progressive disclosure
  - `MemoList` with modern grid layout and infinite scroll
  - Desktop: All action buttons visible in footer
  - Mobile: Compact layout with dropdown menu for secondary actions
- **Agent Engineering**: Comprehensive research and best practices documentation
  - ReflexionExecutor for self-improving agents
  - TimeContext for temporal reasoning
  - Agent architecture patterns and prompt engineering guides
- **UI Components**: New `ServiceUnavailable` component and `alert-dialog` primitive
- **Documentation**: Extensive design docs for memo block, layout, and UI system

### Changed
- Optimize chat list sync message count display
- Apply agent engineering research findings to parrot configs
- Update AI chat components for better UX
- Refactor layouts for consistent responsive width tokens

### Fixed
- Fix memo edit navigation (use UID instead of full name)
- Address remaining PR #143 review issues
- Fix various AI chat component issues

### Removed
- Remove unused divinesense-code-reviewer agent
- Remove `AdminSignIn` page (no longer needed)

---

## [v0.97.0] - 2026-02-10

### Added
- 项目文档更新和重构
- 版本徽章添加到 README
- 对比表格展示项目优势

### Changed
- 优化 README 结构，添加功能特性详细说明
- 更新技术栈描述，补充 AI 模型信息
- 修正 Docker 镜像名称为 `ghcr.io/hrygo/divinesense:stable`

### Documentation
- 重构贡献指南 (CONTRIBUTING.md)
- 优化部署指南 (BINARY_DEPLOYMENT.md)
- 优化用户指南 (CHAT_APPS.md)
- 添加常见问题章节

---

## [v0.93.0] - 2026-02-04

### Added
- 添加 `session_stats` 事件类型用于会话完成统计
- 实现 `result` 消息的统计提取（耗时、成本、token）

### Changed
- **agent_session_stats** 表：Token 使用、成本追踪、工具调用、文件操作
- **异步持久化队列**：后台队列 (size: 100)，优雅关闭
- **成本统计**：日成本聚合、最高消费会话、趋势分析
- Vite 生产构建自动移除 console.log (terser drop_console)

### Fixed
- 修复 goroutine 泄漏和竞态条件 (cc_runner)
- 优化日志输出，移除冗余日志
- 修复 SessionID 显示 (使用真实 UUID 而非 conv_N 格式)

### 📚 规格说明书完善 (cc_runner_async_arch.md v1.3)

#### 事件类型丰富
- 添加 `session_stats` 事件类型用于会话完成统计
- 实现 `result` 消息的统计提取（耗时、成本、token）
- 消除 "unknown message type" 日志警告

#### 可观测性增强
- **agent_session_stats** 表：Token 使用、成本追踪、工具调用、文件操作
- **异步持久化队列**：后台队列 (size: 100)，优雅关闭
- **成本统计**：日成本聚合、最高消费会话、趋势分析

### 💬 Chat UI 改进

#### 微信风格时间戳
- 居中显示在对话界面中央
- 仅当与上一条消息间隔 > 3 分钟时显示
- 灰色胶囊样式 (bg-muted/50)

#### 5 个鹦鹉主题
- **MEMO** (灰灰): slate-800
- **SCHEDULE** (时巧): cyan-600
- **AMAZING** (折衷): emerald-600
- **GEEK** (极客): violet-600 ← 新增
- **EVOLUTION** (进化): rose-600 ← 新增

### 🔧 构建优化
- Vite 生产构建自动移除 console.log (terser drop_console)
- 添加 rollup-plugin-visualizer 用于包分析
- Go embed 兼容性修复 (lodash 内部模块打包)

### 🐛 Bug 修复
- 修复 goroutine 泄漏和竞态条件 (cc_runner)
- 优化日志输出，移除冗余日志
- 修复 SessionID 显示 (使用真实 UUID 而非 conv_N 格式)

## [v0.91.0] - 2026-02-03

### 🤖 CC Runner Session Stats & Cost Tracking

#### Database Schema (PostgreSQL)
- **agent_session_stats**: Full session tracking table
  - Token usage breakdown (input/output/cache read/cache write)
  - Duration metrics (thinking/tool/generation)
  - Cost tracking (total_cost_usd)
  - Tool usage and file operations
  - Error status tracking
- **user_cost_settings**: Budget management table
  - Daily budget limits
  - Per-session cost thresholds
  - Alert preferences (email/in-app)
- **agent_security_audit**: Security audit log table
  - Risk level tracking (low/medium/high/critical)
  - Command pattern matching
  - Action taken logging

#### Backend API
- **GetSessionStats**: Retrieve single session by session_id
- **ListSessionStats**: List sessions with pagination (limit/offset)
- **GetCostStats**: Aggregated N-day cost statistics with daily breakdown
- **GetUserCostSettings**: User budget and alert preferences
- **SetUserCostSettings**: Update cost control settings
- All handlers include user authentication and ownership verification

#### Frontend Components
- **CostTrendChart**: Visualize cost trends over time with daily breakdown
- **SessionSummaryPanel**: Enhanced with cost display and stats
- i18n support for cost tracking UI (en/zh-Hans)

#### Async Persistence
- **Persister**: Background queue-based stats persistence
  - Configurable queue size (default 100)
  - Graceful shutdown with data loss tracking
  - 5-second save timeout per record

#### Testing
- **cc_event_test.go**: 69 comprehensive test cases
  - All CLI message types (11 types)
  - Content block extraction (direct/nested)
  - Result message stats extraction
  - Event dispatch coverage
  - UUID v5 deterministic mapping
  - Session stats collection
  - Edge case handling
  - Concurrent safety

#### Security Fixes
- SQL injection fix in getDailyCostBreakdown (parameterized query)
- rows.Err() checks after all QueryContext iterations
- Proper sql.ErrNoRows vs actual error distinction
- MaxOffset limit (10000) to prevent unbounded pagination

#### Performance Optimizations
- parseStringArray: O(n) performance using strings.Builder
- Partial index `idx_session_stats_user_success` for is_error=false queries
- Removed redundant index on user_cost_settings.user_id

#### Database Improvements
- conversation_id type: INTEGER (matches ai_conversation.id, was BIGINT)
- Constraint name standardized: chk_agent_session_stats_type

#### Documentation
- CC Runner optimization plan specification
- Message handling research report
- Test coverage documentation

## [v0.80.5] - 2026-02-01

### 🔧 Development Workflow
- **Git Hooks**: Added pre-commit + pre-push workflow for local CI validation
  - `pre-commit`: Lightweight checks (go fmt + go vet + pnpm lint:fix), ~5 seconds
  - `pre-push`: Full CI checks (golangci-lint + go test + pnpm build), ~1 minute
  - Catch CI issues locally before pushing to remote

### ✨ Enhancements
- **Makefile**: Added new targets for local CI validation
  - `make install-hooks` — Install pre-commit + pre-push hooks
  - `make ci-check` — Run full CI checks locally (same as GitHub Actions)
  - `make ci-backend` — Backend checks only (golangci-lint + test)
  - `make ci-frontend` — Frontend checks only (lint + build)

### 📚 Documentation
- **README**: Added development workflow section with git hooks documentation
- **BACKEND_DB**: Added Git Hooks & Local CI checks sections
- **FRONTEND**: Added Git Hooks section for frontend workflow
- **ARCHITECTURE**: Added Git Hooks workflow section

## [v0.80.4] - 2026-02-01

### 🐛 Bug Fixes
- **Frontend**: Fixed `__core-js_shared__` error in vendor chunks
  - Inject core-js polyfills as traditional script before modules
  - Removed graph-vendor and utils-vendor chunks to avoid polyfill timing issues
- **PostgreSQL**: Fixed SSL error when running binary without .env file
  - Set default DSN in postgres.go matching .env.example defaults

### ✨ Enhancements
- **UX**: Auto-load .env file when running binary directly
  - Silently loads .env from current directory
  - Skips loading when running as systemd service (uses /etc/divinesense/config)
- **UX**: Added comprehensive database connection error messages
  - PostgreSQL not running → show docker/systemd start commands
  - SSL errors → show how to add sslmode=disable
  - Auth/permission errors → show specific fixes
  - Database not exist → show create commands
  - Detects .env file presence and provides hints

## [v0.80.3] - 2026-02-01

### 🐛 Bug Fixes
- **PostgreSQL**: Fixed SSL error when running binary without .env file
  - Set default DSN in postgres.go matching .env.example defaults
  - Default: `postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable`
  - Resolves: `pq: SSL is not enabled on the server` error

## [v0.80.2] - 2026-02-01

### 📚 Documentation
- **README**: Added "CC Runner 异步架构" section with component overview
  - SessionManager, Streamer, DangerDetector, SessionStats, StopChat RPC
  - Frontend components: EventBadge, ToolCallCard, SessionSummaryPanel, TerminalOutput
  - Architecture advantages: persistent sessions, full-duplex interaction, millisecond streaming
- **ARCHITECTURE**: Added comprehensive CC Runner async architecture section
  - Architecture diagram (Frontend → Backend → CLI)
  - Core components with file paths
  - Session mapping model (UUID v5 deterministic mapping)
  - Interaction protocol (WebSocket events)
  - Security & risk controls
  - API endpoints
  - Link to spec document: `docs/specs/cc_runner_async_arch.md`

## [v0.80.1] - 2026-02-01

### 🐛 Bug Fixes
- **Tests**: Fixed flaky schedule conflict resolver tests failing at month boundaries
  - Replaced `time.Now()` with fixed `testBaseDate` (2026-02-15 UTC)
  - Changed all `time.Local` to `time.UTC` for consistency
  - Resolved CI failures when tests run at Jan 31 UTC crossing into February

## [v0.80.0] - 2026-02-01

### 🤖 CC Runner Async Upgrade (Major)

#### Core Architecture
- **SessionManager**: Persistent session lifecycle management with 30-minute idle timeout
- **Streamer**: Bidirectional event streaming for Claude Code CLI (stdin/stdout/stderr)
- **DangerDetector**: Security layer detecting dangerous operations (rm -rf, format, etc.)
- **SessionStats**: Real-time metrics collection (thinking time, generation time, tokens, tools)

#### Concurrency & Safety
- Fixed goroutine leaks (startup monitoring with 30s timeout)
- Fixed pipe file descriptor leaks (proper cleanup on error paths)
- Fixed timer race conditions (Session.close() with mutex protection)
- Fixed context propagation (defer cancel() on all paths)

#### API Enhancements
- **StopChat RPC**: New endpoint with conversation ownership verification
- **Event Metadata**: Enhanced streaming events with timing and tool info

#### Frontend Components
- **EventBadge**: Visual indicator for event types (thinking, tool_use, answer)
- **ToolCallCard**: Display tool invocation details with status
- **SessionSummaryPanel**: Compact session metrics (duration, tokens, tools, files)
- **TerminalOutput**: Real-time CLI output display for Geek/Evolution modes

#### Documentation
- CC Runner async architecture specification
- Claude Stream JSON format research
- Event type UI research report
- Agent Technology Report 2025
- Git workflow guide (with rebase best practices)

### 🔧 Development Workflow
- Added Git workflow documentation with rebase guidelines
- Enforced conventional commits and PR review process

## [v0.71.0] - 2026-01-31

### 🚀 Deployment Architecture & SSOT
- **SSOT Configuration**: Unified deployment configurations into a "Single Source of Truth". Binary and Docker modes now share the same production template (`deploy/aliyun/.env.prod.example`), reducing maintenance overhead.
- **Smart Installer**: Refactored `install.sh` to dynamically fetch configuration templates from GitHub, supporting version-aware downloads without hardcoded scripts.
- **Geek Mode Config**: Introduced `DIVINESENSE_CLAUDE_CODE_WORKDIR` env var to allow fully configurable sandbox/workspace directories for Geek Mode agents.

### 📚 Documentation
- **Deployment Guide**: Comprehensive update to `deploy/aliyun/README.md`, adding clear "Binary Mode" vs "Docker Mode" operation manuals and explicit file structure maps.
- **Geek Mode Onboarding**: Added detailed step-by-step guides for enabling Geek Mode in Binary deployments.
- **Binary Deployment**: Updated `BINARY_DEPLOYMENT.md` to reflect the new configuration strategies.

### 🧠 Session Management (Preview)
- **Session Redesign**: Laid the groundwork for the new "Hot/Cool Zone" session management strategy to handle large context (Gen UI outputs) more efficiently.
- **Research**: Added `docs/research/20260131-session-management-redesign.md` detailing the new architecture.

### ⚡ Performance & Runtime Optimization
- **Static Assets**: Implemented `Gzip` compression (Level 5) for all embedded assets and API responses, significantly reducing transfer size.
- **Cache Strategy**: Enabled ultra-long (1 year) `immutable` caching for Vite's hashed assets while enforcing `no-cache` for `index.html` to ensure zero-stale UI updates.
- **Security**: Added `X-Content-Type-Options: nosniff` to prevent MIME-sniffing attacks on embedded files.
- **Artifact Hosting**: Optimized `/file/geek/...` route with zero-cache headers for real-time artifact verification and directory-to-index.html fallback.

### 🛠️ Maintenance
- **GitHub Templates**: Added new Pull Request template and verified Issue templates.

## [v0.62.2] - 2026-01-30

### 🛠️ Maintenance

- **Proto**: Formatted protobuf files with `buf format` for CI compliance.
- **Tests**: Improved test output formatting.

## [v0.62.1] - 2026-01-30

### 🛠️ Bug Fixes & Maintenance

- **Lint**: Resolved all remaining linting errors in backend and frontend codebases.
- **Lint**: Updated `golangci-lint` configuration for CI compliance.
- **Cron Tests**: Simplified test patterns and use `time.Equal` for proper time comparison.
- **Tests**: Fixed struct field order in test literals across multiple packages.

### ⚡ Performance

- **Tests**: Optimized test execution time and increased timeout from 30s to 2m.

### ✨ Features

- **AI Context**: Added device context support to Geek Agent for richer contextual awareness.

### 📝 Documentation

- **README**: Updated with research links and detailed agent information.

## [v0.62.0] - 2026-01-30

### 🤓 Geek Mode: The 4th Parrot
- **GeekParrot Agent**: Dedicated agent for code-related tasks. It communicates directly with Claude Code CLI, offering zero LLM latency and robust execution capabilities.
- **Dedicated Routing**: Replaced heuristic keyword matching with a clean, user-controlled Geek Mode toggle that routes inputs directly to the GeekParrot.
- **UI Integration**: Added a dedicated Geek Mode toggle in the `ChatInput` toolbar for quick switching between conversational and coding modes.

### 🎨 UI & UX
- **Chat Input**: Redesigned toolbar with integrated Geek Mode toggle and improved button accessibility.
- **Terminal Aesthetics**: Introduced terminal-style placeholders and icons for Geek Mode.
- **Mobile Refinements**: Minor layout adjustments for better mobile experience.

### 📝 Documentation & Research
- **CLI Agent Best Practices**: Added a comprehensive research document on optimizing CLI agents like Claude Code.
- **Architecture**: Updated `ARCHITECTURE.md` to reflect the new 4-agent system.

### 🛠️ Maintenance
- **Makefile**: Fixed `db-reset` command where the `--migrate` flag was incorrectly used in some contexts.

## [v0.61.0] - 2026-01-29

### 🤓 Geek Mode (Agent Code)
- **Integration**: Added `Geek Mode` configuration support, enabling Claude 3.7 based agentic coding capabilities directly on your server.
- **Manual Setup**: Implemented a secure, opt-in installation process for Claude Code CLI to ensure server security.
- **Documentation**: Added comprehensive guides for both Binary and Docker deployments to enable Geek Mode.

### 🎨 UI Overhaul & Rebranding
- **SVG Header System**: Implemented a dynamic SVG header generation system for unified visual consistency across pages (Attachments, Inbox, Review, etc.).
- **Mobile Experience**: Fixed "double header" issues on mobile views across key pages; optimized layout for smaller screens.
- **Logo Upgrade**: New "DivineSense / 神识" bilingual logos with automatic dark/light mode switching.
- **Chinese Alignment**: Unified sidebar menu text to 4-character alignment (e.g., "闪念笔记", "资源附件") for better aesthetic balance.

### 📝 Documentation
- **Deployment Guides**: Added detailed "One-Click Deployment" guides for Aliyun/Tencent Cloud (2C2G), covering Docker and Binary modes.
- **Env Configuration**: Updated `.env.example` with detailed comments and Zhipu AI / GLM configuration recommendations.
# Changelog

All notable changes to this project will be documented in this file.

## [v0.60.2] - 2026-01-28

### 🐛 Bug Fixes

- **CI/CD**: Switch from Docker Hub to GitHub Container Registry (GHCR) for improved stability and security (#418, #419).
- **TypeScript**: Resolved type validation errors in schedule components and tests (#424, #425).
- **i18n**: Fixed nested translation keys structure for Quick Edit feature (#433, #434).

## [v0.60.1] - 2026-01-28

### 🐛 Bug Fixes & polish

- **Assets**: Fixed outdated PWA icons (Android Chrome) to match new DivineSense branding.
- **Frontend**: Resolved circular dependency warnings in Rollup by fixing `MemoView` imports in pages.
- **CI**: Enabled manual `workflow_dispatch` trigger for stable build workflows.

## [v0.60.0] - 2026-01-28

### 🌟 Rebranding & Major Refactor

- **Brand Identity**: Officially rebranded to **DivineSense (神识)**.
- **Visual Identity**: New logo design with "Neural Spark" concept, supporting both Light and Dark modes.
- **Codebase Structure**: Major refactoring of command entry points (`cmd/divinesense`) and module paths.
- **Architecture**: Cleaned up protobuf generation and dependency management.

### 🛠️ Improvements

- **Git Workflow**: Optimized `.gitignore` and removed local tracking files (`.loki`).
- **Build System**: Updated build scripts and Docker configuration for the new project structure.

## [v0.54.0] - 2026-01-27

### 🚀 Features & UX Improvements

- **Schedule Management**: Optimized scheduling workflow with significantly fewer confirmation steps ("约定 > 配置 > 询问").
- **AI Chat Refactor**: Enhanced AI native experience with refactored Timeline and Calendar components.
- **Conflict Resolution**: Implemented automatic conflict resolution for scheduling with undo option.

### 🌐 Internationalization

- **Locale Cleanup**: Cleaned up i18n locales, focusing on English, Simplified Chinese (简体中文), and Traditional Chinese (繁體中文) for better maintainability.

---

## [v0.53.0] - 2026-01-26

### 📝 Documentation & Code Quality

- **Tailwind Grid Guidelines**: Added critical CSS pitfalls to CLAUDE.md - avoid `max-w-*` on Grid containers
- **Code Formatting**: Standardized AIChat component code style for consistency

---

## [v0.52.0] - 2026-01-25

### 💬 AI Chat Session Persistence

- **Conversation Memory**: AI conversations now persist across sessions with automatic context management
- **Context Separators**: Clear conversation context with visual separators (✂️) - prevents duplicate creation
- **Fixed Conversations**: 5 pinned conversations always visible in history (MEMO, SCHEDULE, AMAZING, CREATIVE, DEFAULT)
- **Real-time Message Count**: Message count updates immediately in conversation list (no page refresh needed)

### 📅 Schedule Optimization

- **Intelligent Conflict Resolution**: Auto-rescheduling with smart time slot suggestions
- **Enhanced Conflict Detection**: Improved detection of overlapping schedules
- **Recurrence Support**: Better handling of recurring events

### 🛡️ Security & Stability

- **Shell Hardening**: Deploy script now uses `tr` and `xargs` to sanitize environment variables
- **Goroutine Safety**: Added 5-second timeout protection for channel draining
- **Cross-platform**: Consistent file size checking using `wc -c` instead of `stat`

### 🔧 Refactoring

- **Parrot Framework**: Migrated DEFAULT parrot to standard parrot framework
- **Migration Consolidation**: PostgreSQL migrations consolidated to 0.51.0 baseline
- **Error Handling**: Improved error logging and DRY compliance

### 🚀 Deployment

- **Aliyun Production Scripts**: Complete deployment automation for Aliyun
- **China-Friendly Mirrors**: Docker registry and npm mirror configurations

---

## [v0.51.0] - 2026-01-23

### 📱 Mobile UI & UX Overhaul

- **Dynamic Navigation**: Fixed mobile header to display current Parrot Agent name and icon.
- **Streamlined Headers**: Simplified mobile sub-header to a single "Back to Nest" button for better chat immersion.
- **Interactive Feedback**: Added micro-scale touch feedback (`active:scale`) to all core buttons and agent cards.
- **Navigation Fix**: Resolved issue where clicking the Logo would cause the sidebar to flash and disappear.

### 🎨 Visual & i18n Polish

- **Unified Avatars**: All AI agents (including default assistant) now use high-quality image avatars instead of emojis.
- **Bilingual Identity**: Updated "Back" text to "返回鹦巢" / "Back to Nest" across en/zh-Hans/zh-Hant.
- **i18n Cleanup**: Optimized locale files by removing 50+ duplicate keys and fixing structure in all supported languages.

## [v0.50.0] - 2026-01-23

### 🦜 Parrot Multi-Agent System - First Release

- **Four Specialized Agents**: Complete implementation of Memo (灰灰), Schedule (金刚), Amazing (惊奇), and Creative (灵灵) Parrots
- **Agent Selection UI**: ParrotHub component with @-mention popover for quick agent switching
- **Metacognition API**: Agents now have self-awareness of capabilities, personality, and limitations
- **Bilingual Support**: Full i18n translations (en/zh-Hans) for all AI chat features
- **Static Assets**: Background images and icons for each parrot agent type
- **UI Polish**: Enhanced chat components with conflict detection and AI suggestions

### 🔧 Improvements

- **Performance**: Code cleanup and optimizations across web components
- **Refactoring**: Extracted common utilities to eliminate duplication
- **Schedule**: Week start day now defaults to Monday

## [v0.31.0] - 2026-01-21

### 🤖 Schedule Agent V2

- **Full Connect RPC Integration**: Migrated Schedule Agent to gRPC Connect protocols for robust streaming support.
- **Streaming Response**: Enabled real-time character streaming for smoother AI interactions, resolving previous gRPC-Gateway buffering issues.
- **Automated Testing Suite**: Added `scripts/test_schedule_agent.sh` and `QUICKSTART_AGENT.md` for comprehensive capabilities verification.
- **Agent Architecture**: Consolidated agent logic into `plugin/ai/agent/`, separating concerns between tools, core logic, and service layers.
- **Environment Management**: Improved dev scripts to handle `.env` loading and project root detection more intelligently.

## [v0.30.0] - 2026-01-21

### 📅 Intelligent Schedule Assistant

- **Smart Query Mode**: Introduced `AUTO`, `STANDARD`, and `STRICT` modes for precise schedule query control.
- **Explicit Year Support**: Parsing for full date formats (e.g., '2025年1月21日', '2025-01-21').
- **Relative Year Keywords**: Added support forms like "后年" (Year after next), "前年" (Year before last).

### 🧠 AI Architecture

- **Adaptive Retrieval**: Context-aware routing for Schedule vs Memo vs QA queries.
- **Query Optimization**: Enhanced filtering logic and schedule integration in search pipeline.

## [v0.26.1-ai.3] - 2026-01-21

### 📅 Schedule UI/UX Polish

- **Compact View**: Redesigned Schedule Calendar and Timeline for better information density and visual appeal.
- **Interaction Enhancements**: Unified "finger" cursors for all interactive elements, optimized "Today" button style.
- **Strict Conflict Policy**: Enforced backend conflict rules by removing "Create Anyway" and guiding users to "Modify/Adjust".
- **Date Formatting**: Standardized on "YYYY MMMM" format and Monday-start weeks.
- **Bug Fixes**: Resolved unused variables and React key warnings in Schedule components.

## [v0.26.1-ai.2] - 2026-01-21

### 🚀 Phase 1 Completion: Advanced AI Architecture

- **Adaptive Retrieval Engine**: Implemented a smart hybrid search system that dynamically switches between BM25 (keyword), Semantic (vector), and Hybrid strategies based on query intent.
- **Intelligent Query Routing**: Added `QueryRouter` to automatically classify user queries (Schedule vs. Memo vs. General QA) and route them to the most effective retrieval pipeline.
- **FinOps Cost Monitoring**: Integrated `CostMonitor` to track token usage and estimate costs for Embedding and LLM calls.
- **Service Modularization**: Refactored `AIService` into focused components (`ai_service_chat.go`, `ai_service_semantic.go`, `ai_service_intent.go`) for better maintainability.
- **Performance Optimization**: optimized Vector Search with parallelism and memory-efficient data structures.

## [v0.26.1-ai.1] - 2026-01-20

### ✨ New Features

- **AI Copilot Chat** - Interactive AI chat page with semantic search capabilities
- **Schedule Assistant** - New scheduling service with AI-powered time extraction
  - Proto definitions and gRPC/REST endpoints
  - Database migrations for MySQL, PostgreSQL, SQLite
  - Full CRUD operations for schedules

### 🔧 Improvements

- **Dev Scripts** - Improved `restart` command (app only, keeps PostgreSQL running)
- **Dev Scripts** - Fixed `stop` command to properly clean up orphan processes
- **i18n** - Simplified internationalization and improved language transition UX
- **Ports** - Updated development ports configuration

### 🐛 Bug Fixes

- Fixed "address already in use" errors after stop/restart
- Fixed `go run` orphan process cleanup on port binding
- Silenced secret context warnings in CI

### 📦 Infrastructure

- Refactored Docker setup for embedding store
- Removed deprecated dev container configs
- Cleaned up memos container service from `prod.yml`
