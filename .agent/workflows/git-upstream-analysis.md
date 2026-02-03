# Git Upstream Alignment Analysis Workflow (Updated)

This workflow helps you scientifically analyze, triage, and plan the synchronization of your local codebase with the upstream repository (`https://github.com/usememos/memos`). 
It evolves from a simple list of changes to a **comprehensive feasibility and ROI analysis**.

**Output Language**: Chinese (Simplified)

## 1. Setup & Discovery

Initialize environment and capture the diff range.

1.  **Configure Temporary Remote**:
    ```bash
    REMOTE_NAME="_agent_upstream_temp"
    git remote add $REMOTE_NAME https://github.com/usememos/memos 2>/dev/null || true
    git fetch $REMOTE_NAME --tags --force
    ```

2.  **Determine Analysis Range**:
    *Robust logic to handle first-time runs or disconnected histories.*
    ```bash
    STATE_FILE=".agent/upstream-sync-state"
    REMOTE_NAME="_agent_upstream_temp"
    TARGET="$REMOTE_NAME/main"
    
    if [ -f "$STATE_FILE" ]; then
        START_POINT=$(cat "$STATE_FILE")
        echo "🔄 Continue from last sync: $START_POINT"
    else
        # Try common ancestor
        START_POINT=$(git merge-base HEAD $TARGET 2>/dev/null)
        if [ -z "$START_POINT" ]; then
             echo "⚠️ No common ancestor found. Defaulting to last 50 commits."
             RANGE="-n 50"
        else
             echo "🆕 Using common ancestor ($START_POINT)"
             RANGE="$START_POINT..$TARGET"
        fi
    fi
    ```

## 2. Intelligence Gathering & Triage

Don't just list titles. Categorize and filter.

1.  **Generate Raw Log**:
    Run commands to get a categorized view of `BREAKING`, `FEAT`, and `FIX`.
    ```bash
    # (Agent: Run 'git log' commands as previously established, filtering by type)
    ```

2.  **Interactive Triage (Agent)**:
    *   **Present** the raw list to the user.
    *   **Ask**: "Which modules or features are we interested in?" (e.g., "Full sync", "Only Security Fixes", "Specific Feature X").

## 3. Deep Dive & Feasibility Analysis (ROI)

**CRITICAL STEP**: Before planning, you must understand the *cost* of synchronization.

*   **For Selected Features**:
    1.  **Code Inspection**: Use `git show --stat <commit>` to see which files changed.
    2.  **Local Comparison**: Check if the corresponding local files exists.
        *   *Example*: "Upstream changed `plugin/scheduler`. Do we utilize `plugin` in `server.go`?"
    3.  **Conflict Prediction**: Identify heavily modified files (e.g., `go.mod`, `package.json`, Core UI components).

*   **Output**: Generate a **Feasibility Report** (`.agent/upstream_sync_analysis.md`) containing:
    *   **Gap Analysis**: What exists locally vs upstream?
    *   **ROI**: Is it a simple copy-paste or a complex refactor?
    *   **Risk**: Are there breaking changes (e.g., DB Schema, API Contracts)?

## 4. External Verification

*   **Security/Bug Checks**: If the upstream fixes a bug (e.g., "Fix Auth", "CSRF"), use `search_web` to verify the validity and severity of the issue (e.g., MDN docs, CVEs).
    *   *Goal*: Confirm if it's a "Must Fix" or "Nice to have".

## 5. Execution Planning

Formulate the Action Plan.

1.  **Draft Development Plan**:
    Create a checklist (`.agent/upstream_sync_dev_plan.md`) separated by phases:
    *   **Phase 1: Critical Fixes** (Blockers, Security).
    *   **Phase 2: Backend/Core** (Dependencies, Infrastructure).
    *   **Phase 3: Frontend/UI** (Components, Styles).
    *   **Phase 4: Verification** (How to test).

2.  **Submit Issue**:
    Interact with the user to confirm the plan, then create/update the GitHub Issue.
    ```bash
    gh issue create --title "🏗️ Upstream Sync: [Topic]" --body-file .agent/upstream_sync_dev_plan.md --label "maintenance"
    ```

3.  **Update State**:
    (Only after successful plan adoption)
    ```bash
    git rev-parse $TARGET > .agent/upstream-sync-state
    ```

## 6. Cleanup

```bash
git remote remove _agent_upstream_temp 2>/dev/null
```
