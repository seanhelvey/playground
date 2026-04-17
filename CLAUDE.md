# Playground

A personal flywheel for habits, dreams, goals, and projects. Go API + SQLite + PWA. Deployed on Fly.io.

---

## STOP — Discuss Before Changing

**Do not write code, edit files, or make commits until the user agrees on the plan. No exceptions.**

Ask. Wait. Then act.

---

## Working with Claude

- **Say "I don't know" instead of guessing.** If the root cause isn't proven, say so.
- **Maintain a small integration test suite.** A handful of high-level tests covering critical paths only — auth, item CRUD, logging. Do not add a test for every feature. Add one only when no existing test covers the path. Delete tests that duplicate coverage. Keep the suite fast and readable.
- **Track work in GitHub Issues.** Every bug or feature should have an issue. Reference it in commits.
- **Don't make unrequested changes.** Fix what was asked, nothing more.
- **Names are NOT primary keys.** The `items` table must use `id INTEGER PRIMARY KEY AUTOINCREMENT`. Name is a mutable user-facing label — `TEXT NOT NULL`, no uniqueness constraint. `id` is identity. Never constrain mutable fields for uniqueness; that breaks when users rename things.
- **Multi-tenancy is coming.** Items will need a `user_id` column. Don't design anything that assumes items are globally shared. See issue for tracking this migration.

---

## Engineering rules

These exist because each one burned us.

- **Wire it or don't write it.** Every new column, endpoint, or JS function must be traceable to something that calls or renders it. If you can't show the full path (UI action → API → DB, or DB column → API response → UI render), don't add it.
- **Migrations are schema-only.** `migrate()` touches table structure only — `CREATE TABLE`, `ALTER TABLE`. Never put `INSERT`, `UPDATE`, or `DELETE` of row data in migration code. One-off data changes go through the API or a manual query, then get removed. User data lives in the DB and is managed through the UI.
- **Handle every error at the boundary.** Every `db.Exec` in a handler must check its error. On failure: log it, return 4xx/5xx, stop. Never return a success response if a write failed. A UI save confirmation is only trustworthy if the server actually confirmed it.
- **Mutable data is never identity.** Before writing any dedup, lookup, or sort, ask: what happens when this value changes? Names change. Timestamps drift. IDs from different autoincrement sequences don't compare. Use stable IDs.
- **Dates come from the client.** The server clock is UTC. For anything user-facing, accept the date in the request body (validated as `YYYY-MM-DD`); fall back to server time only when absent.
- **Closing a mode cleans up its children.** When UI state changes (edit mode off, panel closed), explicitly reset all sub-state that mode owned — open menus, pending inputs, selections.
- **When fixing a bug, scan for the same pattern.** One bug usually has siblings. Grep for the anti-pattern across the codebase before closing the issue.
- **After any structural HTML/CSS change, manually verify edit mode.** Restructuring the DOM (e.g. changing which element holds a class) silently breaks CSS selectors. Check that edit mode shows reorder buttons and hides interactive controls before shipping.

---

## Git

- Push directly to `main`, no branches.
- Exception: Claude Code web sessions create a feature branch via the session harness. In that case, use the assigned branch but merge to `main` at end of session — don't leave fixes sitting on unmerged branches.

## Local dev

```bash
cd api
go mod tidy
go run .
```

Open `http://localhost:8080`. The server serves the PWA from `static/`.

## Deploy

Pushing to `main` triggers an automatic deploy via `.github/workflows/deploy.yml`. Requires `FLY_API_TOKEN` set as a GitHub Actions secret.

**Verify a deploy succeeded:**
```bash
# Check GitHub Actions (unauthenticated, rate-limited to 60/hr)
curl -s "https://api.github.com/repos/seanhelvey/playground/actions/runs?per_page=1" | python3 -c "import json,sys; r=json.load(sys.stdin)['workflow_runs'][0]; print(r['conclusion'], r['display_title'])"

# Check what SHA is live on Fly.io
curl -s https://playground-flywheel.fly.dev/api/health
```
Compare the `sha` in the health response to the latest git commit (`git rev-parse --short HEAD`).

---

## Data model

### Items

Each item is a single unified shape — no type distinction between a daily habit and a long-term goal:

| Column | Type | Notes |
|---|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` | Identity. Never use name as PK. |
| `name` | `TEXT NOT NULL` | Mutable label. Not unique. |
| `input_type` | `TEXT` | `boolean`, `counter`, `slider` |
| `step_size` | `INTEGER` | Increment per log tap |
| `step_unit` | `TEXT` | `min`, `hr`, `species`, etc. |
| `target_value` | `INTEGER` | Goal amount per period |
| `target_period` | `TEXT` | `daily`, `weekly`, `monthly` |
| `range_min/max` | `INTEGER` | For sliders |
| `display_order` | `INTEGER` | Sort order |
| `active` | `INTEGER` | 0 = soft deleted |
| `completed_date` | `TEXT` | Set when permanently done; null = ongoing |
| `last_updated` | `TEXT` | ISO date of last log |
| `group_id` | `INTEGER` | FK to `groups.id`; null = ungrouped |

Progress = net sum of log entries in current period window. Handles thrashing correctly (+5+5−5 = 5).

### Groups

User-defined clusters for habit stacking and time-of-day organization. Rendered as pill tabs and card sections in the UI.

| Column | Notes |
|---|---|
| `id` | PK |
| `name` | Display label (Morning, Evening, etc.) |
| `display_order` | Sort order for tabs |

### Logs

Append-only. References item by `item_id INTEGER`.

| Column | Notes |
|---|---|
| `item_id` | FK to `items.id` |
| `date` | ISO date |
| `type` | Optional: `recommendation` |
| `note` | Short text |

### Milestones

Wins scoped to an item. References item by `item_id INTEGER`.

| Column | Notes |
|---|---|
| `item_id` | FK to `items.id` |
| `date` | ISO date |
| `label` | Short text |

### Check-ins

Weekly wellness snapshots: `body`, `mind`, `social` (1–10), `feeling` (one word), `more_of`, `less_of`.

### Wins

Cross-cutting good moments: `{ date, note }`. Log during check-ins or whenever something good happens.

---

## How check-ins work

Two channels, not connected yet:

1. **App (PWA)** — user logs activity, records wins, does weekly check-in. Data goes to SQLite via API. **Source of truth for all user data.**
2. **Claude Code (iOS)** — separate tool for system improvement: reviewing issues, making code changes, discussing what to build next.

---

## Public-facing — treat like a portfolio

This repo is public. Nothing sensitive, private, or unprofessional. No full names, specific addresses, personal struggles, financial details, or private relationships. When in doubt, leave it out.

---

## Vision: Intelligent PDCA Flywheel

**This is the entire point.** Not a dashboard. A system that learns and adapts.

```
PLAN ──── DO ──── CHECK ──── ACT ──── REPEAT
Today: human-driven. Goal: system-driven.
```

### Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────┐
│  PWA         │────▶│  Go API      │────▶│  SQLite  │
│  (phone)     │◀────│  (Fly.io)    │◀────│  (volume)│
└──────────────┘     └──────────────┘     └──────────┘

Claude Code = dev tool only (no tokens in deployed app)
```

### Design principles

- **Don't make me think** — Interface explains itself.
- **Single source of truth** — CLAUDE.md for system design. Database for user data.
- **Flywheel > features** — Every addition must make the daily loop better.
- **Phone-first** — If it doesn't work on the phone, it doesn't work.
- **Ship simple, improve always** — A working v1 beats a perfect plan.
- **Engineer for observability** — Errors, save confirmations, health signals are not extras. Keep them readable on the phone without dev tools.
- **Always display the deployed SHA** — The UI must show the short git SHA fetched from `/api/health`. This makes it immediately obvious whether a deploy succeeded without opening dev tools.
- **After every `git push origin main`, output the SHA** — Run `git rev-parse --short HEAD` and tell the user the SHA so they can verify it appears in the app.
