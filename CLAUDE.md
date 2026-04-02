# Playground

A personal flywheel for habits, dreams, goals, and projects. Go API + SQLite + PWA. Deployed on Fly.io.

## Git
- Push directly to `main`, no branches.

## Local dev
```bash
cd api
go mod tidy
go run .
```
Open `http://localhost:8080`. The server seeds SQLite from `data.json` + `tasks.json` on first run and serves the PWA from `static/`.

## Deploy
```bash
fly deploy
```

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

### Tasks (`tasks.json`)
Agent backlog — things for the system to work on, not user to-dos:
```json
{
  "id": 1,
  "task": "Description of what needs to happen",
  "status": "pending",
  "created": "2026-04-01"
}
```
- Status: `pending` or `done`
- When a task is done, remove it. Keep the list lean.
- These are system improvement tasks, not item-level actions (those live in each item's `next` field).

## How check-ins work
**Daily check-in at noon PT** (scheduled via remote agent). Covers yesterday evening, this morning, and heading into tonight. On Sundays, adds weekly reflection questions (body/mind/social scores, feeling, more_of/less_of).

Each daily check-in also includes a **system flywheel** component:
- **Engagement report**: check-in streak, % items updated in last 7 days, goals on pace vs behind. Updates the derived "Stick with the process" momentum.
- **Task progress**: status update on each pending agent task.
- **One innovation**: a concrete proposal for improving the system — data model, interface, workflow, architecture. Builds on the task backlog.

The agent presents all of this alongside the check-in questions, then waits for the user to respond before making any changes.

The user may also share updates conversationally:
- **Direct**: "meditation 6/7 this week" → update focus, momentum, append log.
- **Conversational**: something comes up naturally → ask if they want it logged before adding.
- **Review**: "how's everything looking" → summarize the state. Be a friend, not a manager.

When updating:
1. Read `data.json` and `tasks.json` first.
2. Update focus and momentum based on what was shared. **Evaluate momentum relative to the item's expected cadence** — a daily habit stalling after 3 missed days is different from a 1-year goal with no update in a week.
3. Append a dated log entry.
4. Add milestones for any wins or achievements.
5. Update `last_updated` to today's date.
6. If it's a weekly check-in (Sunday), append to `check_ins` array too.
7. **Show the user all changes and wait for their OK before committing and pushing.**

## Recommendations
When providing recommendations for items (especially Nature, Coloft):
- Be **specific and local** — Humboldt County, current season, named places and organizations.
- Be **timely** — what can be done THIS week or month.
- Be **actionable** — not "consider gardening" but "show up to First Saturday native gardening, 11:30am, 2nd & F St, Old Town Eureka."
- Store as log entries with `type: "recommendation"` so they persist and can be refreshed.

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

## Architecture — Cloud Migration Path

This section is the single source of truth for where the system is headed. The current static-JSON + GitHub Pages setup is Phase 0. The goal is a real app that reliably reaches the user, persists data properly, and could serve others.

### Phase 0: Now (static files)
```
GitHub Pages: index.html ← data.json + tasks.json
Scheduled Claude agent (cron) → reads/writes JSON via git
```
**Works:** Data model, UI, check-in structure, flywheel loop.
**Broken:** Can't reach the user's phone. No notifications. No real-time interaction.

### Phase 1: Phone-first app (target: deploy the full-stack project)
The core need is a **standalone PWA that pings you and lets you check in**. No Claude integration in the running app. No tokens. No third-party AI dependency.
```
┌──────────────┐     ┌──────────────┐     ┌──────────┐
│  PWA         │────▶│  Go API      │────▶│  SQLite  │
│  (installable│◀────│  + Web Push  │◀────│  (Fly    │
│  mobile web) │     │  + Cron      │     │  volume) │
└──────────────┘     └──────────────┘     └──────────┘

Claude Code = dev tool (build, iterate, manage)
The app itself = standalone, no AI dependency
```
**What it does:**
- PWA on phone home screen. Push notification at noon. Tap → check-in form in the app.
- Go API serves data, receives check-in responses, computes engagement metrics.
- Built-in cron sends Web Push at noon PT. No external scheduler needed.
- SQLite for one user. Postgres when/if multi-user matters.

**What Claude Code does (dev time only):**
- Build and iterate on features with the user
- System flywheel: review engagement, propose improvements, manage tasks
- No Claude tokens or API keys in the deployed app

**Security before deploy:**
- API key auth (env var on Fly, reject requests without it)
- Spending limit $0 in Fly dashboard
- Rate limiting on endpoints

**Hosting research (completed):**

| Platform | Free tier | 1 user cost | 100 users | Go/Rust | DB | Notes |
|----------|-----------|-------------|-----------|---------|-----|-------|
| **Fly.io** | 3 shared VMs, 1GB vol | $0 | $5-10/mo | Both native | SQLite (volume), Postgres (built-in) | Best fit. Real binary, scales simply. Machines sleep on free tier. |
| Cloudflare Workers+D1+Pages | 100K req/day, 5GB D1 | $0 | $0-5/mo | Rust→WASM only, no Go | D1 (SQLite-compat) | Cheapest at scale but no Go, WASM-only Rust, D1 still maturing. |
| Railway | $5 trial credit (one-time) | ~$5/mo | $10-20/mo | Both via Docker | Postgres add-on | No real free tier. |
| Render | 1 free service (sleeps), 90-day free Postgres | $0-7/mo | $7-25/mo | Both native | Postgres (managed) | Free Postgres expires. 30s+ cold starts on free. |
| Vercel | Generous frontend, 100GB-hrs functions | $0 frontend | $20/mo Pro | Go yes, Rust limited | None (external) | Wrong paradigm for Go/Rust API server. |
| Firebase | 125K invocations/mo, 1GB Firestore | $0 | $5-25/mo | Go (2nd gen only), no Rust | Firestore (NoSQL) | Best push (FCM), but vendor lock-in, NoSQL only. |

**Recommendation: Fly.io.** Native Go/Rust, SQLite on a volume for one user, built-in Postgres for growth, $0 to start. Web Push is just HTTP calls from the backend.

**Decided:**
- Language: **Go** (ship fast, learn new language, Rust for OSS separately)
- Push: **Web Push** (free, no external service needed)
- Hosting: **Fly.io free tier** ($0/mo, credit card required, set spending limit to $0)
- Auth: **API key** (env var on Fly, no user accounts yet)
- Claude: **Dev tool only** (no tokens in deployed app, no runtime AI dependency)

### Phase 2: Multi-user + Open Source (future)
- Others run their own flywheel
- Coloft community connection?
- OSS contributions come from others building on the framework

### Design principles
- **Don't make me think** — Interface explains itself.
- **Single source of truth** — CLAUDE.md for system design. Database for user data.
- **Flywheel > features** — Every addition must make the daily loop better.
- **Phone-first** — If it doesn't work on the phone, it doesn't work.
