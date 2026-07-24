# Playground

A personal flywheel for habits, dreams, goals, and projects — one system for tracking daily habits and long-term goals, instead of two disconnected ones.

**Live demo:** https://playground-flywheel.fly.dev
**Login:** `demo@playground.app` / `demo1234` (shared, rate-limited demo account — feel free to click around)

## What it does

Every habit and goal is the same underlying shape: a boolean toggle, a counter, or a slider, each with its own target and period. Log a rep and progress rolls up automatically.

- **Flexible items** — boolean, counter, or slider inputs with configurable targets and periods
- **Groups** — cluster habits by time of day or theme
- **Weekly check-ins** — quick body/mind/social snapshot
- **Activity feed** — a running log of everything you've done
- **PWA** — installable, phone-first, works offline

## Stack

Go API · SQLite · vanilla JS PWA (no framework) · deployed on Fly.io via GitHub Actions on every push to `main`

The UI shows the live deployed git SHA (from a `/api/health` endpoint), so it's always obvious whether a deploy landed.

## Local dev

```bash
cd api
go mod tidy
go run .
```

Open `http://localhost:8080` — the server serves the PWA directly from `static/`.
