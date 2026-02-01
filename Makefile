# DivineSense Makefile
# SPDX-License-Identifier: MIT

# Load .env file if exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ===========================================================================
# Configuration
# ===========================================================================

.DEFAULT_GOAL := help

# Database Configuration (PostgreSQL)
DIVINESENSE_DRIVER ?= postgres
DIVINESENSE_DSN ?= postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable
POSTGRES_CONTAINER ?= divinesense-postgres-dev
POSTGRES_PORT ?= 25432
POSTGRES_USER ?= divinesense
POSTGRES_DB ?= divinesense

# AI Configuration
AI_EMBEDDING_PROVIDER ?= siliconflow
AI_LLM_PROVIDER ?= deepseek
AI_EMBEDDING_MODEL ?= BAAI/bge-m3
AI_RERANK_MODEL ?= BAAI/bge-reranker-v2-m3
AI_LLM_MODEL ?= deepseek-chat
AI_OPENAI_BASE_URL ?= https://api.siliconflow.cn/v1

# Paths
DOCKER_COMPOSE_DEV ?= docker/compose/dev.yml
DOCKER_COMPOSE_PROD ?= docker/compose/prod.yml
DEPLOY_DIR ?= deploy/aliyun
DEPLOY_SCRIPT ?= $(DEPLOY_DIR)/deploy.sh
SCRIPT_DIR ?= scripts

# Backend
BACKEND_BIN ?= bin/divinesense
BACKEND_CMD ?= cmd/divinesense
BACKEND_PORT ?= 28081

# Frontend
WEB_DIR ?= web

# ===========================================================================
# Phony Targets
# ===========================================================================

.PHONY: help run dev web test deps clean
.PHONY: install-hooks ci-check
.PHONY: docker-up docker-down docker-logs docker-reset
.PHONY: db-connect db-reset db-vector
.PHONY: start stop restart status logs
.PHONY: logs-backend logs-frontend logs-postgres
.PHONY: logs-follow-backend logs-follow-frontend logs-follow-postgres
.PHONY: git-status git-diff git-log git-push
.PHONY: check-branch check-build check-test check-i18n check-i18n-hardcode check-all
.PHONY: deps deps-web deps-ai deps-all
.PHONY: build build-web build-all build-verify
.PHONY: clean clean-all
.PHONY: test test-ai test-embedding test-runner
.PHONY: release-build release-package release-all bin-install bin-deploy
.PHONY: docs-check docs-ref docs-tree docs-tidy docs-index
.PHONY: dev-logs dev-logs-backend dev-logs-frontend dev-logs-follow
.PHONY: check-embed-frontend check-embed-backend check-embed-all
.PHONY: checksum verify-checksum

# ===========================================================================
# Development Commands
# ===========================================================================

##@ Development

run: ## 启动后端 (PostgreSQL + AI)
	@echo "Starting DivineSense with AI support..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) \
		DIVINESENSE_DSN=$(DIVINESENSE_DSN) \
		DIVINESENSE_AI_ENABLED=true \
		DIVINESENSE_AI_EMBEDDING_PROVIDER=$(AI_EMBEDDING_PROVIDER) \
		DIVINESENSE_AI_LLM_PROVIDER=$(AI_LLM_PROVIDER) \
		DIVINESENSE_AI_SILICONFLOW_API_KEY=$(SILICONFLOW_API_KEY) \
		DIVINESENSE_AI_DEEPSEEK_API_KEY=$(DEEPSEEK_API_KEY) \
		DIVINESENSE_AI_OPENAI_API_KEY=$(OPENAI_API_KEY) \
		DIVINESENSE_AI_OPENAI_BASE_URL=$(AI_OPENAI_BASE_URL) \
		DIVINESENSE_AI_EMBEDDING_MODEL=$(AI_EMBEDDING_MODEL) \
		DIVINESENSE_AI_RERANK_MODEL=$(AI_RERANK_MODEL) \
		DIVINESENSE_AI_LLM_MODEL=$(AI_LLM_MODEL) \
		go run ./$(BACKEND_CMD) --mode dev --port $(BACKEND_PORT)

dev: run ## Alias for run

web: ## 启动前端开发服务器
	@cd $(WEB_DIR) && pnpm dev

start: build ## 一键启动所有服务 (自动构建最新版本)
	@$(SCRIPT_DIR)/dev.sh start

stop: ## 一键停止所有服务
	@$(SCRIPT_DIR)/dev.sh stop

restart: build ## 重启所有服务 (自动构建最新版本)
	@$(SCRIPT_DIR)/dev.sh restart

status: ## 查看所有服务状态
	@$(SCRIPT_DIR)/dev.sh status

logs: ## 查看所有服务日志
	@$(SCRIPT_DIR)/dev.sh logs

logs-backend: ## 查看后端日志
	@$(SCRIPT_DIR)/dev.sh logs backend

logs-frontend: ## 查看前端日志
	@$(SCRIPT_DIR)/dev.sh logs frontend

logs-postgres: ## 查看 PostgreSQL 日志
	@$(SCRIPT_DIR)/dev.sh logs postgres

logs-follow-backend: ## 实时跟踪后端日志
	@$(SCRIPT_DIR)/dev.sh logs backend -f

logs-follow-frontend: ## 实时跟踪前端日志
	@$(SCRIPT_DIR)/dev.sh logs frontend -f

logs-follow-postgres: ## 实时跟踪 PostgreSQL 日志
	@$(SCRIPT_DIR)/dev.sh logs postgres -f

# 统一日志视图 (新增开发命令)
dev-logs: ## 统一日志视图 (前后端合并, 颜色区分)
	@chmod +x $(SCRIPT_DIR)/unified-logs.sh
	@$(SCRIPT_DIR)/unified-logs.sh all

dev-logs-backend: ## 查看后端日志 (格式化)
	@chmod +x $(SCRIPT_DIR)/unified-logs.sh
	@$(SCRIPT_DIR)/unified-logs.sh backend

dev-logs-frontend: ## 查看前端日志 (格式化)
	@chmod +x $(SCRIPT_DIR)/unified-logs.sh
	@$(SCRIPT_DIR)/unified-logs.sh frontend

dev-logs-follow: ## 实时跟踪所有日志 (格式化)
	@chmod +x $(SCRIPT_DIR)/unified-logs.sh
	@$(SCRIPT_DIR)/unified-logs.sh all -f

# ===========================================================================
# Dependencies
# ===========================================================================

##@ Dependencies

deps: ## 安装后端依赖
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

deps-web: ## 安装前端依赖
	@cd $(WEB_DIR) && pnpm install

deps-ai: ## 安装 AI 依赖
	@echo "Installing AI dependencies..."
	@go get github.com/tmc/langchaingo
	@go mod tidy

deps-all: deps deps-web ## 安装所有依赖

# ===========================================================================
# Docker (PostgreSQL)
# ===========================================================================

##@ Docker

docker-up: ## 启动 PostgreSQL
	@echo "Starting PostgreSQL..."
	@docker compose -f $(DOCKER_COMPOSE_DEV) up -d

docker-down: ## 停止 PostgreSQL
	@echo "Stopping PostgreSQL..."
	@docker compose -f $(DOCKER_COMPOSE_DEV) down --remove-orphans

docker-logs: ## 查看 PostgreSQL 日志
	@docker compose -f $(DOCKER_COMPOSE_DEV) logs -f postgres

docker-reset: ## 重置 PostgreSQL 数据 (危险!)
	@echo "Resetting PostgreSQL data..."
	@docker compose -f $(DOCKER_COMPOSE_DEV) down -v
	@docker volume rm divinesense_postgres_data 2>/dev/null || true
	@$(MAKE) docker-up

docker-prod-up: ## 启动生产环境
	@echo "Starting production environment..."
	@docker compose -f $(DOCKER_COMPOSE_PROD) up -d

docker-prod-down: ## 停止生产环境
	@echo "Stopping production environment..."
	@docker compose -f $(DOCKER_COMPOSE_PROD) down

docker-prod-logs: ## 查看生产环境日志
	@docker compose -f $(DOCKER_COMPOSE_PROD) logs -f

# ===========================================================================
# Database Commands
# ===========================================================================

##@ Database

db-connect: ## 连接 PostgreSQL shell
	@docker exec -it $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

db-reset: ## 重置数据库 schema
	@echo "Resetting database schema..."
	@docker exec $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@go run ./$(BACKEND_CMD) --mode dev --driver postgres --dsn "$(DIVINESENSE_DSN)"

db-vector: ## 验证 pgvector 扩展
	@docker exec $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"

# ===========================================================================
# Test Commands
# ===========================================================================

##@ Testing

test: ## 运行所有测试
	@echo "Running tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test -tags=noui $$(go list ./... | grep -v -E "(^github.com/hrygo/divinesense/plugin/cron$$|^github.com/hrygo/divinesense/proto/)") -short -timeout 2m 2>&1 | grep -E "^(ok |FAIL|\?)" | tee test-summary.log
	@echo ""
	@echo "Test summary:"
	@echo "  Passed: $$(grep -c '^ok ' test-summary.log || echo 0) packages"
	@if grep -q "^FAIL" test-summary.log 2>/dev/null; then \
		echo "  Failed: $$(grep -c '^FAIL' test-summary.log) packages"; \
		exit 1; \
	else \
		echo "  All tests passed!"; \
	fi

.PHONY: test-verbose
test-verbose: ## 运行所有测试(详细输出)
	@echo "Running tests with verbose output..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test -tags=noui $$(go list ./... | grep -v -E "(^github.com/hrygo/divinesense/plugin/cron$$|^github.com/hrygo/divinesense/proto/)") -v -timeout 2m

test-ai: ## 运行 AI 测试
	@echo "Running AI tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./plugin/ai/... -v

test-embedding: ## 运行 Embedding 测试
	@echo "Running Embedding tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./plugin/ai/... -run Embedding -v

test-runner: ## 运行 Runner 测试
	@echo "Running Runner tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./server/runner/embedding/... -v

# ===========================================================================
# Build Commands
# ===========================================================================

##@ Build

build: ## 构建后端
	@echo "Building backend..."
	@go build -o $(BACKEND_BIN) ./$(BACKEND_CMD)
	@if [ "$$(go env GOOS)" = "darwin" ] && command -v codesign >/dev/null 2>&1; then \
		echo "Signing binary with ad-hoc signature..."; \
		codesign --force --deep --sign - $(BACKEND_BIN); \
	fi

build-web: ## 构建前端
	@echo "Building frontend..."
	@cd $(WEB_DIR) && pnpm build

build-all: build build-web ## 构建前后端
	@echo "✓ 构建完成"

##@ Build Verification

build-verify: check-embed-all ## 验证构建产物完整性
	@echo "✓ 构建验证通过"

check-embed-frontend: ## 检查前端嵌入完整性 (dist vs index.html)
	@chmod +x $(SCRIPT_DIR)/check-embed-integrity.sh
	@$(SCRIPT_DIR)/check-embed-integrity.sh

check-embed-backend: ## 检查后端嵌入配置 (embed files exist)
	@chmod +x $(SCRIPT_DIR)/check-backend-embed.sh
	@$(SCRIPT_DIR)/check-backend-embed.sh

check-embed-all: check-embed-backend check-embed-frontend ## 检查所有嵌入完整性

checksum: ## 生成构建产物 SHA256 校验和
	@chmod +x $(SCRIPT_DIR)/generate-checksum.sh
	@$(SCRIPT_DIR)/generate-checksum.sh

verify-checksum: ## 验证构建产物校验和
	@if [ ! -f .checksums ]; then \
		echo "错误: 校验和文件不存在，请先运行: make checksum"; \
		exit 1; \
	fi
	@echo "验证构建产物..."
	@if command -v shasum >/dev/null 2>&1; then \
		shasum -a 256 -c .checksums; \
	else \
		sha256sum -c .checksums; \
	fi

# ===========================================================================
# Clean Commands
# ===========================================================================

##@ Clean

clean: ## 清理构建文件
	@rm -rf bin/
	@cd $(WEB_DIR) && rm -rf dist/ node_modules/.vite

clean-all: clean ## 清理所有
	@cd $(WEB_DIR) && rm -rf node_modules/
	@go clean -modcache

# ===========================================================================
# Git Workflow Commands
# ===========================================================================

##@ Git Workflow

git-status: ## 查看 Git 状态
	@echo "Current Git status:"
	@git status --short

git-diff: ## 查看变更详情
	@echo "Showing changes..."
	@git diff --stat

git-log: ## 查看最近提交
	@echo "Recent commits:"
	@git log --oneline -10

git-push: ## 推送到远程 (需先检查)
	@echo "Checking branch and pushing..."
	@if [ "$$(git branch --show-current)" = "main" ]; then \
		echo "ERROR: Cannot push to main directly. Create a feature branch first."; \
		exit 1; \
	fi
	@git push origin "$$(git branch --show-current)"

check-branch: ## 检查当前分支
	@echo "Current branch: $$(git branch --show-current)"
	@if [ "$$(git branch --show-current)" = "main" ]; then \
		echo "WARNING: You are on main branch. Consider creating a feature branch."; \
	fi

check-build: ## 检查编译
	@echo "Checking build..."
	@go build $$(go list ./... | grep -v "^github.com/hrygo/divinesense/proto/") || { echo "Build failed"; exit 1; }
	@echo "Build OK"

check-test: ## 检查测试
	@echo "Running tests..."
	@go test -tags=noui $$(go list ./... | grep -v -E "(^github.com/hrygo/divinesense/plugin/cron$$|^github.com/hrygo/divinesense/proto/)") -short -timeout 30s || { echo "Tests failed"; exit 1; }
	@echo "Tests OK"

check-i18n: ## 检查 i18n 翻译完整性 (强制)
	@echo "Checking i18n translations..."
	@chmod +x $(SCRIPT_DIR)/check-i18n.sh
	@$(SCRIPT_DIR)/check-i18n.sh

check-i18n-hardcode: ## 检查前端硬编码文本
	@echo "Checking hardcoded text..."
	@chmod +x $(SCRIPT_DIR)/check-i18n-hardcode.sh
	@$(SCRIPT_DIR)/check-i18n-hardcode.sh

##@ CI Quality Gates

check-all: check-build check-test check-lint check-i18n ## 运行所有检查

install-hooks: ## 安装 git hooks (pre-commit + pre-tag)
	@echo "📦 Installing git hooks..."
	@$(SCRIPT_DIR)/install-hooks.sh

ci-check: ## 模拟 CI 运行所有检查（与 GitHub Actions 一致）
	@$(MAKE) --no-print-directory ci-check-internal

ci-check-internal:
	@echo "🔍 Running CI checks locally..."
	@echo ""
	@$(MAKE) --no-print-directory ci-backend || { echo ""; exit 1; }
	@$(MAKE) --no-print-directory ci-frontend || { echo ""; exit 1; }
	@echo ""
	@echo "✅ All CI checks passed!"

ci-backend: ## 后端 CI 检查 (go mod tidy + golangci-lint + test)
	@echo "📦 Backend:"
	@echo "  → go mod tidy check..."
	@cp go.mod go.mod.bak 2>/dev/null || true; \
		cp go.sum go.sum.bak 2>/dev/null || true; \
		go mod tidy; \
		if ! git diff --quiet go.mod go.sum; then \
			echo "  ❌ go.mod/go.sum not tidy. Run: go mod tidy"; \
			mv go.mod.bak go.mod 2>/dev/null || true; \
			mv go.sum.bak go.sum 2>/dev/null || true; \
			exit 1; \
		fi; \
		rm -f go.mod.bak go.sum.bak
	@echo "  → golangci-lint..."
	@golangci-lint run --config=.golangci.yaml --timeout=3m --build-tags=noui
	@echo "  → go test..."
	@go test -short -timeout=30s -tags=noui ./...
	@echo "  ✅ Backend checks passed"

ci-frontend: ## 前端 CI 检查 (lint + build)
	@echo "🎨 Frontend:"
	@cd web && \
		echo "  → pnpm lint..." && \
		pnpm lint --silent && \
		echo "  → pnpm build..." && \
		pnpm build >/dev/null 2>&1 && \
		cd .. && \
		echo "  ✅ Frontend checks passed"

lint: ## 运行 golangci-lint (使用 .golangci.yaml 配置)
	@echo "Running golangci-lint..."
	@golangci-lint run --config=.golangci.yaml --timeout=3m --build-tags=noui || { echo "Linting failed"; exit 1; }
	@echo "Linting OK"

vet: ## 运行 go vet
	@echo "Running go vet..."
	@go vet ./... || { echo "Vet failed"; exit 1; }
	@echo "Vet OK"

check-lint: lint vet ## 检查代码风格 (Lint + Vet)

# ===========================================================================
# Documentation Management Commands
# ===========================================================================

##@ Documentation

docs-check: ## 检查文档完整性和链接
	@echo "📋 Checking documentation..."
	@python3 .claude/skills/docs-manager/docs_helper.py check

docs-ref: ## 显示文档引用关系
	@echo "🔗 Building reference graph..."
	@python3 .claude/skills/docs-manager/docs_helper.py refs

docs-tree: ## 显示文档结构树
	@echo "📂 docs/ structure:"
	@python3 .claude/skills/docs-manager/docs_helper.py tree

docs-tidy: ## 整理文档(检测重复、命名规范)
	@echo "🧹 Tidy up documentation..."
	@python3 .claude/skills/docs-manager/docs_helper.py duplicates

docs-index: ## 更新文档索引(需指定目录)
	@echo "⚠️ Usage: make docs-index DIR={research|specs|dev}"
	@if [ -z "$(DIR)" ]; then \
		echo "Error: DIR parameter required. Example: make docs-index DIR=research"; \
		exit 1; \
	fi
	@echo "Updating index for $(DIR)..."
	@echo "⚠️ Please use /docs-index command for automated index updates"

.PHONY: docs-check docs-ref docs-tree docs-tidy docs-index

# ===========================================================================
# Release Commands (Binary Deployment)
# ===========================================================================

##@ Release

release-build: ## 构建发布二进制 (linux/amd64 + linux/arm64)
	@echo "Building release binaries..."
	@chmod +x scripts/release/build-release.sh
	@./scripts/release/build-release.sh $(VERSION)

release-package: ## 打包发布文件
	@echo "Packaging release..."
	@chmod +x scripts/release/package-release.sh
	@./scripts/release/package-release.sh $(VERSION)

release-all: release-build release-package ## 完整发布流程 (构建 + 打包)
	@echo "Release complete!"

# ===========================================================================
# Binary Deployment Commands
# ===========================================================================

##@ Binary Deployment

bin-install: ## 本地安装二进制 (开发测试)
	@echo "Installing binary locally..."
	@chmod +x deploy/aliyun/install.sh
	@sudo ./deploy/aliyun/install.sh --mode=binary $(VERSION)

bin-deploy: ## 部署管理脚本
	@echo "Binary deployment management..."
	@chmod +x deploy/aliyun/deploy-binary.sh
	@./deploy/aliyun/deploy-binary.sh $(CMD)

# ===========================================================================
# Production Deployment Commands (2C2G)
# ===========================================================================

##@ Production Deployment

prod-build: ## 构建生产镜像
	@echo "Building production image..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) build

prod-deploy: ## 部署到生产环境
	@echo "Deploying to production..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) deploy

prod-restart: ## 重启生产服务
	@echo "Restarting production services..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) restart

prod-stop: ## 停止生产服务
	@echo "Stopping production services..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) stop

prod-logs: ## 查看生产服务日志
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) logs

prod-status: ## 查看生产服务状态
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) status

prod-backup: ## 备份生产数据库
	@echo "Backing up production database..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) backup

# ===========================================================================
# Help
# ===========================================================================

##@ Help

help: ## 显示此帮助信息
	@printf "\033[1m\033[36m\nDivineSense Development Commands\033[0m\n\n"
	@printf "\033[1mQuick Start:\033[0m\n"
	@printf "  make docker-up               # 启动 PostgreSQL\n"
	@printf "  make start                   # 启动后端 + 前端\n"
	@printf "  访问 http://localhost:25173 # 打开前端\n\n"
	@printf "\033[1mNew Commands:\033[0m\n"
	@printf "  make dev-logs                 # 统一日志视图 (前后端合并)\n"
	@printf "  make dev-logs-follow         # 实时跟踪日志\n"
	@printf "  make check-embed-all         # 检查构建完整性\n"
	@printf "  make checksum                # 生成校验和\n\n"
	@awk 'BEGIN { section = ""; old_section = ""; printed_first = 0 } \
		/^##@/ { section = $$0; gsub(/^##@ /, "", section); next } \
		/^[a-zA-Z0-9_%-]+:.*?##/ { \
			split($$0, a, ":"); cmd = a[1]; \
			for(i = 2; i <= length(a); i++) { if(i == 2) desc = a[i]; else desc = desc ":" a[i]; } \
			sub(/.*## /, "", desc); \
			if (section != old_section) { \
				if (printed_first == 0) printf "\n\033[1m%s:\033[0m\n", section; \
				else printf "\n\033[1m%s:\033[0m\n", section; \
				old_section = section; \
				printed_first = 1; \
			} \
			printf "  \033[36m%-26s\033[0m %s\n", cmd, desc \
		}' Makefile
