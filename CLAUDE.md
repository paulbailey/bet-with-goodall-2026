# bet-with-goodall

A static website for a group of friends to track shared FIFA World Cup 2026 accumulator bets.

## What it is

The group places several accumulator bets before the tournament (group-winner predictions). The site shows a live grid of those bets — green while still mathematically possible, red once a leg is bust.

## Repo layout

```
web/        Svelte 5 + Vite frontend
builder/    Go binary that polls for results and regenerates the JSON data files
infra/      Terraform for AWS S3 + CloudFront
config/     bets.yaml — the bet definitions, edited once before the tournament
```

## How the system works

1. `config/bets.yaml` defines every accumulator and its legs (which team wins which group).
2. The Go builder runs as a k8s Deployment on a homelab cluster. It polls a football results API, calculates which legs are still alive, and writes `data/state.json` to the S3 bucket.
3. CloudFront serves the static site. The Svelte app fetches `data/state.json` and renders the bet grid.
4. After each day's matches finish, the builder also writes `data/daily-summary.json` — a natural-language recap (Claude API, with a templated fallback) plus the day's biggest risers/fallers in bet likelihood and any bets that landed or went bust. The site shows the latest on the dashboard and keeps a browsable archive at `/daily-summary`. See `builder/README.md` → Daily summary.
5. The site is an installable **PWA** (vite-plugin-pwa, custom service worker in `web/src/sw.ts`) that works offline. Users can opt in to **push notifications**: when a match finishes the builder sends a Web Push showing the score, and tapping it deep-links to a `/match/<id>` page detailing how the result moved the pending bets (rendered from `data/match-results.json`, which the builder writes per finished match). Subscriptions are stored in DynamoDB behind a Lambda function URL (`infra/push.tf`, exposed same-origin at `/api/*` via CloudFront); the builder reads that table and signs pushes with a VAPID key. See `builder/README.md` → Push notifications.

## Key decisions

- Frontend: Svelte 5 + Vite, no SSR. Components use Svelte 5 runes (`$state`, `$derived`, `$props`).
- Builder: Go, long-running Deployment (smart polling — frequent near match times, slow otherwise).
- Auth to AWS: IRSA (k8s ServiceAccount annotation).
- Infrastructure: Terraform, user has Terraform Cloud configured.
- Data feed: TBD — football-data.org (free) or API-Football (paid). Pluggable behind an interface.
- Bet config: `config/bets.yaml`, committed before the tournament, not editable at runtime.
- Access: fully public, no auth.

## 2026 World Cup format

48 teams, 12 groups of 4. Each team plays 3 group-stage games. The group winner is the team that finishes 1st in their group on points (then goal difference, goals scored, head-to-head).

## What "mathematically impossible" means

For a group-winner leg: the predicted team can no longer finish 1st in their group regardless of remaining results. The builder calculates this after every result update.

An accumulator is marked dead the moment any one of its legs is dead.
