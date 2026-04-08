# Playground

A personal flywheel for habits, dreams, goals, and projects. Go API + SQLite + PWA. Deployed on Fly.io.

## Git
- Push directly to `main`, no branches.
- Exception: Claude Code web sessions create a feature branch via the session harness. In that case, use the assigned branch but merge to `main` at end of session — don't leave fixes sitting on unmerged branches.

## Working with Claude
- **Discuss before changing.** Talk through the approach first. Don't start writing code until the user agrees on the plan. No exceptions.
- **Maintain a small integration test suite.** A handful of high-level tests covering the critical paths. Not one per fix — use judgement. New tests only when an existing one doesn't cover it. Keep them passing.
- **Say "I don't know" instead of guessing.** If the root cause isn't proven, say so. Don't state confident answers without evidence.
- **Track work in GitHub Issues.** Every bug or feature should have an issue. Reference it in commits.
- **Don't make unrequested changes.** Fix what was asked, nothing more.
- **Names are NOT primary keys.** The `items` table must use `id INTEGER PRIMARY KEY AUTOINCREMENT` with `name TEXT UNIQUE NOT NULL`. Never use a mutable user-facing string as a PK.

## Local dev
```bash
cd api
go mod tidy
go run .
```
Open `http://localhost:8080`. The server seeds SQLite from `data.json` on first run and serves the PWA from `static/`.

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

## Data model

### Items (`data.json`)
Each item has:
- **Type**: `Core` (daily non-negotiables), `Habit` (daily practices being built), `Dream` (bigger aspirations), `Goal` (SMART goals with target dates). Type captures expected **cadence** — a daily habit and a slow-burn goal like homeownership both have momentum, but "stalling" means different things for each. Evaluate momentum relative to the item's natural rhythm.
- **Momentum**: `rising`, `steady`, `stalling`, `dormant` — updated based on log activity relative to expected cadence, not guesswork.
- **Focus**: one honest sentence about where this actually stands right now.
- **Next**: one specific, concrete action.
- **Milestones**: array of `{ date, label }` — wins and achievements. Proof that things are real.
- **Log**: append-only. Each entry has a date, optional type, and a short note.
  - Regular: `{ date, note }`
  - Recommendations: `{ date, type: "recommendation", note }` — specific, actionable, timely. Refreshed during check-ins.

### Goals (SMART)
Goal-type items also have:
- **target_date**: when it should be done (e.g. `"2027-03-31"`)
- **success_criteria**: one sentence defining what "done" looks like

### Engagement metric ("Stick with the process")
This item is **derived, not self-reported**. Its momentum comes from:
- Check-in streak (how many consecutive days the daily check-in ran)
- % of items updated in the last 7 days
- Goals on pace vs their target dates
The daily check-in agent should compute and report this.

### Wins (`data.json`)
`wins` array — cross-cutting good moments that prove the flywheel is working:
```json
{ "date": "2026-04-12", "note": "First time fishing the bay. Didn't catch anything but loved it." }
```
Wins can relate to specific items or cut across many. Log them during check-ins or whenever something good happens. They're the record of the life being built.

### Check-ins (`data.json`)
`check_ins` array — weekly wellness snapshots:
```json
{
  "date": "2026-03-31",
  "body": 7,
  "mind": 8,
  "social": 5,
  "feeling": "restless",
  "more_of": "time outside",
  "less_of": "evening screens"
}
```
- **body, mind, social**: 1-10 scores. Simple, honest.
- **feeling**: one word for the overall vibe.
- **more_of / less_of**: one word or short phrase each.
- Keep these public-safe — no clinical language, no private details.

### Tasks
System improvement tasks are tracked as GitHub Issues in this repo. Use labels (`priority:highest`, `priority:high`, etc.) to manage order. Close issues when done.

## How check-ins work
**Two channels, not connected yet:**

1. **App (PWA)** — user logs activity, records wins, does weekly check-in (body/mind/social, feeling, more/less). Data goes to SQLite via API. **This is the source of truth for all user data.**

2. **Claude Code (iOS)** — user opens Claude Code separately for system improvement: reviewing issues, making code changes, discussing what to build next.

`data.json` is a one-time seed file. Once the DB is seeded it is never read again. The live DB on Fly.io is what matters.

The user may share updates conversationally:
- **Direct**: "meditation 6/7 this week" → acknowledge it, encourage logging it in the app.
- **Conversational**: something comes up naturally → ask if they want it logged.
- **Review**: "how's everything looking" → summarize what the user has shared. Be a friend, not a manager.

## Recommendations
When providing recommendations for items (especially Nature, Coloft):
- Be **specific and local** — Humboldt County, current season, named places and organizations.
- Be **timely** — what can be done THIS week or month.
- Be **actionable** — not "consider gardening" but "show up to First Saturday native gardening, 11:30am, 2nd & F St, Old Town Eureka."

## Tone
Be a friend. Supportive, honest, not pushy. If something hasn't been touched in a while, mention it gently — don't lecture. Match the user's energy. Sometimes they want structure, sometimes they're just thinking out loud.

## Scope
This system is relevant when:
- The user is explicitly checking in on habits or dreams.
- Something in the conversation naturally relates to a tracked item.
- The user asks for a review or summary.

It is NOT relevant when:
- The user is working on something unrelated and hasn't referenced it.
- Forcing a connection would feel annoying.

Use judgment. When in doubt, don't bring it up.

## Adding new items
If the user mentions a new habit, interest, or dream that seems like it belongs here, ask once: "want to add that to the tracker?" Don't assume.

## Public-facing — treat like a portfolio
This repo is public. Think of it like a resume or personal brand artifact — sharing interests and growth is fine, but nothing sensitive, private, or unprofessional. No full names, no specific addresses, no personal struggles, no financial details, no private relationships. Keep the tone something you'd be comfortable with a potential collaborator or employer reading. When in doubt, leave it out.

## Vision: Intelligent PDCA Flywheel

**This is the entire point of this project.** Not a dashboard. Not a form. A system that learns and adapts through continuous improvement — for both the user's life and the system itself.

```
        ┌─────────────────────────────────────┐
        │          THE FLYWHEEL               │
        │                                     │
        │   PLAN ──── DO ──── CHECK ──── ACT  │
        │     │                          │    │
        │     └──────── REPEAT ──────────┘    │
        │                                     │
        │   Today: human-driven (manual PDCA) │
        │   Goal: system-driven (auto PDCA)   │
        └─────────────────────────────────────┘
```

### Where we are now (v1 — manual loop)
- **PWA on Fly.io** — check in, log activity, record wins from phone
- **Claude Code on iOS** — separate tool, user-initiated, reads repo + reviews state, proposes improvements
- **The loop is manual**: user checks in via app, then separately opens Claude Code to reflect and give tasks
- The two halves (app data + Claude intelligence) are **not connected yet**

### Where this is headed (v2 — intelligent loop)
The app itself becomes smart. Not just a place to enter data, but a system that:
- **Notices patterns** — "you always stall on meditation after weekends"
- **Adapts questions** — asks what's relevant today, not the same 6 fields every time
- **Computes momentum** — from actual behavior, not self-reported status
- **Proposes experiments** — "try morning meditation instead of evening for a week"
- **Tracks what works** — closes the loop on its own suggestions
- **Gets smarter over time** — learns from what you actually do, not what you plan to do

This could mean Claude API calls from the server, or smarter rule engines, or both. The right approach will emerge from actually using v1 and seeing what's missing. **Ship simple, use it, improve it.**

### Architecture
```
┌──────────────┐     ┌──────────────┐     ┌──────────┐
│  PWA         │────▶│  Go API      │────▶│  SQLite  │
│  (phone)     │◀────│  (Fly.io)    │◀────│  (volume)│
└──────────────┘     └──────────────┘     └──────────┘

Claude Code = dev tool only (no tokens in deployed app)
```
- **Go + SQLite + Fly.io free tier** ($0/mo)
- **Session auth** (bcrypt + cookies), rate-limited
- **PWA** — installable on phone, works offline for static assets
- First user registers freely, subsequent users need INVITE_CODE env var

### Design principles
- **Don't make me think** — Interface explains itself.
- **Single source of truth** — CLAUDE.md for system design. Database for user data.
- **Flywheel > features** — Every addition must make the daily loop better.
- **Phone-first** — If it doesn't work on the phone, it doesn't work.
- **Ship simple, improve always** — A working v1 beats a perfect plan.
- **Engineer for observability** — Every feature should answer: how will I know if this is working? Instrument as you build. Errors, save confirmations, activity logs, and health signals aren't extras — they're what make the flywheel self-improvable. Keep them accessible through the interface itself, readable on the phone without dev tools. A tool you can't observe is a tool you can't trust.
