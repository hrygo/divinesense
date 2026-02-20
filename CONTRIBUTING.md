# Contributing to DivineSense

> AI-powered personal second brain with Orchestrator-Workers multi-agent architecture.

---

## Quick Start

### Prerequisites

| Tool       | Version | Install                |
| :--------- | :------ | :--------------------- |
| Go         | >= 1.25 | `brew install go`      |
| Node.js    | >= 20   | `fnm install 22`       |
| pnpm       | >= 10   | `npm i -g pnpm`        |
| Docker     | latest  | Docker Desktop         |
| Make       | any     | built-in (macOS/Linux) |
| GitHub CLI | latest  | `brew install gh`      |

### Setup

```bash
# Clone
git clone https://github.com/hrygo/divinesense.git && cd divinesense

# Install dependencies
make deps-all

# Install git hooks (required)
make install-hooks

# Start database
make docker-up

# Start dev server
make start
```

Access: http://localhost:25173

### Verify

```bash
make check-all  # Run all checks
make status     # Check service status
```

---

## Tech Stack

| Layer    | Tech                                            |
| :------- | :---------------------------------------------- |
| Backend  | Go 1.25, Echo, Connect RPC, pgvector            |
| Frontend | React 18, Vite 7, TypeScript, Tailwind CSS 4    |
| Database | PostgreSQL 16+ (prod), SQLite (dev)             |
| AI       | GLM/DeepSeek (chat), SiliconFlow (embed/rerank) |

---

## Project Structure

```
divinesense/
├── cmd/divinesense/          # App entry
├── server/                   # HTTP/gRPC server
├── ai/                       # AI core
│   ├── agents/orchestrator/  # Orchestrator-Workers
│   ├── routing/              # Smart routing
│   └── core/                 # LLM/embeddings
├── web/                      # React frontend
├── store/                    # Data layer
├── proto/                    # Protobuf definitions
├── config/                   # Config files
└── deploy/                   # Deployment scripts
```

---

## Architecture

```
User Input → ChatRouter
                 ↓
         ├─> Cache (LRU)
         ├─> Rule Matcher (config-driven)
         │     └─> confidence < 0.8 ? → Orchestrator
         └─> UniversalParrot (config-driven Agent)
         └─> EvolutionParrot/GeekParrot (Code-driven Agent → CCRunner Hot-Multiplexing)
```

| Component       | File                                     | Purpose             |
| :-------------- | :--------------------------------------- | :------------------ |
| ChatRouter      | `ai/agents/chat_router.go`               | Request routing     |
| FastRouter      | `ai/routing/service.go`                  | Two-layer routing   |
| Orchestrator    | `ai/agents/orchestrator/orchestrator.go` | Task decomposition  |
| UniversalParrot | `ai/agents/universal/parrot.go`          | Config-driven Agent |

**Expert Agents** (config-driven via UniversalParrot):
- MemoParrot (memo search) → memo.yaml
- ScheduleParrot (calendar) → schedule.yaml
- GeneralParrot (general tasks) → general.yaml

**External Executors** (Code-driven via CCRunner v2.0 Global Singleton):
- GeekParrot: Direct code terminal execution
- EvolutionParrot: Self-modification & PR generation (Admin only)
- Engineering: Persistent OS Process Groups (PGID) + Hot-Multiplexing Stdin/Stdout flows

---

## Development

### Code Style

**Go**
- Files: `snake_case.go`
- Logs: `log/slog`
- Always handle errors

**React/TypeScript**
- Components: `PascalCase.tsx`
- Hooks: `use` prefix
- Styles: Tailwind CSS
- Types: no `any`

### Tailwind CSS 4 Trap

> Never use `max-w-sm/md/lg/xl` — resolves to ~16px in v4

```tsx
// Wrong
<div className="max-w-md">

// Correct
<div className="max-w-[28rem]">  // 448px
```

### i18n

All UI text must be bilingual:

```bash
# Add key to both files
web/src/locales/en.json
web/src/locales/zh-Hans.json

# Verify
make check-i18n
```

---

## Git Workflow

```
Issue → Branch → Develop → PR → Merge
```

### Branch Naming

| Type     | Format               | Example                |
| :------- | :------------------- | :--------------------- |
| Feature  | `feat/<id>-desc`     | `feat/123-ai-router`   |
| Fix      | `fix/<id>-desc`      | `fix/456-session-bug`  |
| Refactor | `refactor/<id>-desc` | `refactor/789-cleanup` |

### Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type       | Example                           |
| :--------- | :-------------------------------- |
| `feat`     | `feat(ai): add intent router`     |
| `fix`      | `fix(db): resolve race condition` |
| `refactor` | `refactor(ui): extract hooks`     |
| `docs`     | `docs: update README`             |
| `test`     | `test(ai): add router tests`      |
| `chore`    | `chore(deps): upgrade deps`       |

### Create PR

```bash
# Sync with main
git fetch origin && git rebase origin/main

# Run checks
make check-all

# Create PR
gh pr create --title "feat(scope): description" --body "$(cat <<'EOF'
## Summary
Brief description of changes

Resolves #XXX

## Changes
- Change 1
- Change 2

## Test plan
- [ ] Local tests pass
- [ ] `make check-all` passes
EOF
)"
```

---

## Commands

| Command           | Description              |
| :---------------- | :----------------------- |
| `make help`       | Show all commands        |
| `make deps-all`   | Install dependencies     |
| `make docker-up`  | Start database           |
| `make start`      | Start backend + frontend |
| `make test`       | Run backend tests        |
| `make check-all`  | Full check               |
| `make check-i18n` | Verify i18n sync         |

---

## Resources

| Resource       | Path                                              |
| :------------- | :------------------------------------------------ |
| Architecture   | `docs/architecture/overview.md`                   |
| Backend Guide  | `docs/dev-guides/backend/database.md`             |
| Frontend Guide | `docs/dev-guides/frontend/overview.md`            |
| Deployment     | `docs/dev-guides/deployment/BINARY_DEPLOYMENT.md` |
| Debug Lessons  | `docs/research/DEBUG_LESSONS.md`                  |

---

## Getting Help

1. Check [Issues](https://github.com/hrygo/divinesense/issues)
2. Search [Discussions](https://github.com/hrygo/divinesense/discussions)
3. Create new issue: `gh issue create --interactive`

---

*Last updated: 2026-02-20*
