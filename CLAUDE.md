# Playground

A personal flywheel for habits, dreams, and projects. The source of truth is `data.json`. The page at `index.html` renders it. This repo is hosted on GitHub Pages.

## Git
- Push directly to `main`, no branches.
- Commit and push after updating data.

## Local preview
The page fetches `data.json` via JS, which won't work from `file://`. To preview locally:
```
python3 -m http.server 8000
```
Then open `http://localhost:8000`. GitHub Pages serves it fine — this is only needed for local dev.

## Data model
- `data.json` contains an array of items, each with: name, type, momentum, focus, next, and a log array.
- **Types**: Core (daily non-negotiables), Habit (daily practices being built), Dream (bigger aspirations being explored).
- **Momentum**: `rising`, `steady`, `stalling`, `dormant` — updated based on log activity, not guesswork.
- **Focus**: one honest sentence about where this actually stands right now.
- **Next**: one specific, concrete action.
- **Log**: append-only. Each entry has a date and a short note. This is the history.

## How check-ins work
The user may share updates in different ways:
- **Direct**: "meditation 6/7 this week" → update focus, momentum, append log, push.
- **Conversational**: something comes up naturally in another topic → ask if they want it logged before adding.
- **Review**: "how's everything looking" → summarize the state of all items. Mention what's active, what's stalling, what hasn't been touched. Be a friend, not a manager.

When updating:
1. Read `data.json` first.
2. Update focus and momentum based on what was shared.
3. Append a dated log entry.
4. Update `last_updated` to today's date.
5. Commit and push.

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
