# Implementation Plan: EvolutionParrot Persistent Sessions (DRY Refactor)

## Goal
Enable `EvolutionParrot` to use persistent `Claude Code CLI` sessions (30m keep-alive), same as `GeekParrot`.
Refactor functionality to adhere to **SOLID** and **DRY** principles by extracting common session execution logic.

## User Review Required
> [!IMPORTANT]
> This refactoring implies that both `GeekParrot` and `EvolutionParrot` will now rely on a shared `ExecutePersistentSession` function in the `geek` package.

## Proposed Changes

### 1. Extract Shared Logic
Create a new file `ai/agents/geek/session_common.go` (or `common.go`) to house the session management logic.

#### [NEW] [session_common.go](file:///Users/huangzhonghui/divinesense_fork/ai/agents/geek/session_common.go)
-   `func ExecutePersistentSession(ctx, runner, config, input, callback)`
-   Encapsulates:
    -   `StartAsyncSession` call
    -   Callback wrapping (handling `result`, `error`, `answer`)
    -   `SetCallback` / `SetStats`
    -   `WriteInput`
    -   Turn completion wait logic

### 2. Refactor GeekParrot
Update `GeekParrot` to use the new shared function.

#### [MODIFY] [parrot.go](file:///Users/huangzhonghui/divinesense_fork/ai/agents/geek/parrot.go)
-   Replace manual session code in `Execute` with `ExecutePersistentSession`.

### 3. Update EvolutionParrot
Implement persistent session logic in `EvolutionParrot` using the shared function.

#### [MODIFY] [evolution.go](file:///Users/huangzhonghui/divinesense_fork/ai/agents/geek/evolution.go)
-   Refactor `NewEvolutionParrot` to use the shared `SessionManager` (or its own instance).
    -   **Decision**: Evolution Mode uses its own dedicated SessionManager instance to guarantee process isolation if desired, or share the global one.
    -   To fulfill "Evolution Mode uses independent processes", distinct `SessionID` is sufficient. Sharing `SessionManager` code instance is fine and simplifies resource tracking.
    -   However, to satisfy the user's explicit request for "independent sessions AND processes", having distinct `CCRunner` instances (which `EvolutionParrot` already does) with their own `SessionManager` instances is the safest architectural choice for isolation, even if slightly redundant in resource management.
    -   *Correction*: `CCRunner` holds the `SessionManager`. So if `EvolutionParrot` creates a new `CCRunner` with `NewCCRunner` (default), it gets a new `SessionManager`. That's perfect isolation.
-   Update `Execute` to use `ExecutePersistentSession`.

## Verification Plan

### Automated Tests
-   Verify build succeeds: `go build ./ai/agents/geek/...`.
-   Verify `keepalive.go` still works (indirectly tests shared logic if GeekParrot uses it).

### Manual Verification
-   Confirm code structure is clean and DRY.
