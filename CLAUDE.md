# Playground

A personal flywheel for habits, dreams, goals, and projects. Source of truth: `data.json` and `tasks.json`. Rendered at `index.html`. Hosted on GitHub Pages.

## Git
- Push directly to `main`, no branches.
- Commit and push after updating data.

## Local preview
The page fetches JSON via JS, which won't work from `file://`. To preview locally:
```
python3 -m http.server 8000
```
Then open `http://localhost:8000`. GitHub Pages serves it fine — this is only needed for local dev.

## Data model

### Items (`data.json`)
Each item has:
- **Types**: Core (daily non-negotiables), Habit (daily practices being built), Dream (bigger aspirations), Goal (SMART goals with target dates).
- **Momentum**: `rising`, `steady`, `stalling`, `dormant` — updated based on log activity, not guesswork.
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

Goals are worked through weekly PDCA cycles. Milestones are the checkpoints that prove momentum. The weekly check-in is where SMART targets and PDCA loops meet.

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
Persisted action items that carry across sessions:
```json
{
  "id": 1,
  "task": "Description of what needs to happen",
  "status": "pending",
  "related": "Item name or null",
  "created": "2026-03-31"
}
```
- Status: `pending` or `done`
- Related: links to a tracker item by name, or null
- When completing a task, set status to `done` — don't delete it
- When a task generates a result, also log it on the related item

## How check-ins work
The user may share updates in different ways:
- **Direct**: "meditation 6/7 this week" → update focus, momentum, append log, push.
- **Conversational**: something comes up naturally in another topic → ask if they want it logged before adding.
- **Review**: "how's everything looking" → summarize the state of all items. Mention what's active, what's stalling, what hasn't been touched. Be a friend, not a manager.
- **Weekly check-in**: prompted by scheduled agent or user. Cover: what worked, what didn't, body/mind/social scores, feeling word, more_of/less_of. Review goal progress against targets. Update everything and push.

When updating:
1. Read `data.json` and `tasks.json` first.
2. Update focus and momentum based on what was shared.
3. Append a dated log entry.
4. Add milestones for any wins or achievements.
5. Update `last_updated` to today's date.
6. If it's a weekly check-in, append to `check_ins` array too.
7. Mark any completed tasks in `tasks.json`.
8. Commit and push.

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
