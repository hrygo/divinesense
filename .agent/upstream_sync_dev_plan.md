# 📅 Upstream Sync Development Plan: Phase 2 (Architecture Integration)

**Status**: Phase 1 (OAuth Fix) ✅ Completed.
**Focus**: Backend Plugin Wiring & System Verification.

## 1. 🎯 Objectives
*   **Activate Plugins**: 使现有的 `plugin/scheduler` (Cron) 和 `plugin/email` (SMTP) 代码真正运行起来。
*   **Verify Features**: 确保 Scheduler 能触发任务，Email 能发送邮件，Markdown 扩展能渲染。
*   **Cleanup**: 移除不再需要的临时代码或配置。

## 2. 🛠️ Implementation Steps

### 2.1 Backend: Service Wiring (High Priority)
*   **Target**: `server/server.go`
*   **Action**:
    1.  Initialize `scheduler.NewScheduler()` in `NewServer`.
    2.  Integrate `scheduler` into the Server struct.
    3.  Manage Lifecycle: Ensure `scheduler.Start()` and `scheduler.Stop()` are called during server startup/shutdown.
    4.  (Optional) Wire `email` service if configuration exists in `profile`.

### 2.2 Frontend: Feature Verification
*   **Mermaid & LaTeX**:
    *   Create a Memo with ```mermaid graph TD; A-->B;``` and check rendering.
    *   Create a Memo with `$$ E = mc^2 $$` and check rendering.
*   **Focus Mode**:
    *   Test `Cmd+Shift+F` shortcut.
    *   Test Exit via `Esc` or Backdrop click.
    *   **Fix**: If UI is broken (CSS conflict), apply `FOCUS_MODE_STYLES` from upstream.

### 2.3 Bug Verification (Privacy)
*   **Target**: `web/src/pages/Home.tsx`
*   **Action**: Simulate "Token Refresh" execution flow (using DevTools to clear token temporarily) and observe if Memos list disappears.
*   **Fix**: If verified, apply the `enabled={!!user}` removal patch.

## 3. 📋 Execution Checklist

- [x] **Phase 1: Critical Fixes**
    - [x] OAuth PKCE Fallback (`web/src/utils/oauth.ts`)

- [ ] **Phase 2: Backend Architecture**
    - [ ] Initialize Scheduler in `server/server.go`
    - [ ] Initialize Email Service in `server/server.go`
    - [ ] Update `profile` to support SMTP config (if missing)

- [ ] **Phase 3: Frontend Alignment**
    - [ ] Verify/Fix Focus Mode Styles
    - [ ] Verify Mermaid/LaTeX Dependencies

- [ ] **Phase 4: Final Validation**
    - [ ] Full Integration Test (Login -> Write Memo -> Focus Mode -> Logout)

## 4. 📝 Notes for Developer
*   **Scheduler**: Check `server/router/api/v1/schedule_service.go` to see if it relies on the new `plugin/scheduler`.
*   **Email**: Check `profile` struct for SMTP configuration mapping.
