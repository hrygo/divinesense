# Task: Implement Persistent Sessions (Process Keep-Alive)

## Objective
Enable `GeekParrot` and `EvolutionParrot` to reuse the same `Claude Code CLI` process for multiple commands within a 30-minute window, improving responsiveness and context management.

## Checklist

- [x] **GeekParrot Process Keep-Alive**
    - [x] Research: Confirm architecture feasibility with mock (`StartAsyncSession` + `WriteInput`)
    - [x] Plan: Create implementation plan for persistent session handling
    - [x] Implement: Refactor `CCRunner` for persistent monitoring
    - [x] Implement: Add global `SessionManager` to `GeekParrot`
    - [x] Implement: Rewrite `GeekParrot.Execute` to use persistent sessions
    - [x] Verify: Compilation and regression testing (`keepalive.go`)

- [x] **EvolutionParrot Process Keep-Alive**
    - [x] Plan: Draft implementation details for EvolutionParrot refactoring
    - [x] Implement: Add global `SessionManager` for Evolution Mode
    - [x] Implement: Refactor `EvolutionParrot.Execute` to use `StartAsyncSession`
    - [x] Verify: Compilation and logic check

- [x] **Comprehensive Audit**
    - [x] Architecture Review: Validate persistent session & isolation design across modes
    - [x] Code Quality Review: Verify DRY/SOLID compliance in shared logic
    - [x] Security Review: Audit environment isolation and path handling
    - [x] Concurrency Review: Check SessionManager locking and state transitions (Fixed Race Condition)
    - [x] Documentation Review: Ensure architecture docs match implementation
