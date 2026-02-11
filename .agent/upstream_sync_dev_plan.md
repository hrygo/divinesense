# Upstream Sync Development Plan (2026-02-10)

## Phase 1: Critical Fixes (High Priority)
Focus on stability, authentication, and performance.

### 1. OAuth PKCE Optionality
- **Upstream Commit**: `cf0a285e` (fix(auth): make PKCE optional for OAuth sign-in)
- **Target Files**:
  - `web/src/pages/SignIn.tsx`
  - `web/src/utils/oauth.ts`
- **Rationale**: Ensure OAuth works across different environments (HTTP/HTTPS) and prevents crashes when `crypto.subtle` is unavailable.
- **Action**: `git cherry-pick cf0a285e` (Resolve conflicts if any).

### 2. Activity Calendar Performance
- **Upstream Commit**: `74b63b27` (perf: disable tooltips in year calendar to fix lag)
- **Target Files**:
  - `web/src/components/ActivityCalendar/YearCalendar.tsx`
  - `web/src/components/ActivityCalendar/MonthCalendar.tsx`
  - `web/src/components/ActivityCalendar/CalendarCell.tsx`
- **Rationale**: Significant performance improvement by reducing DOM nodes (tooltips) in the year view.
- **Action**: `git cherry-pick 74b63b27`.

### 3. Calendar Navigation Logic
- **Upstream Commit**: `b5108b4f` (fix(web): calendar navigation should use current page path)
- **Target Files**: `web/src/hooks/useDateFilterNavigation.ts`
- **Rationale**: UX fix for proper navigation context.
- **Action**: `git cherry-pick b5108b4f`.

## Phase 2: User Experience Polish (Medium Priority)

### 4. Shortcut Edit Logic
- **Upstream Commit**: `e7605d90` (fix: shortcut edit button opens create dialog instead of edit dialog)
- **Target Files**: `web/src/components/MemoContent/index.tsx` (Likely location, verify on inspect)
- **Action**: `git cherry-pick e7605d90`.

## Phase 3: Technical Debt & Cleanup (Low Priority)
- **Linter Fixes**: `b623162d` (chore: fix static check linter warnings)
- **Dead Code Removal**: `cf65f086` (refactor: remove hide-scrollbar utility)

## Verification Plan
1.  **Auth**: Test OAuth login with HTTP (if possible) and HTTPS.
2.  **Calendar**: Test year view rendering speed and navigation between months.
3.  **Shortcuts**: Test editing a memo via keyboard shortcut.
