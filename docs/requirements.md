# Requirements

Captured from interview on 2026-05-24.

## Context

A group of friends (including Goodall) have placed several shared accumulator bets on the FIFA World Cup 2026. They want a website to watch those bets unfold in real time during the tournament — no admin, no per-user tracking, just a shared dashboard everyone can open.

## Functional requirements

### Bets

- Multiple accumulator bets, each tracked independently.
- Each accumulator consists of several legs; each leg predicts which team wins a specific group.
- 2026 World Cup has 12 groups (A–L), so up to 12 legs per accumulator.
- Bets are fixed before the tournament starts. They are defined in `config/bets.yaml` and do not change at runtime.
- Each bet has at minimum: a name and an ordered list of (group, predicted_winner) pairs.
- Optionally: a stake amount for display purposes.

### Display

- A grid showing all bets and their current status.
- Each leg cell is:
  - **Grey / neutral** — group stage hasn't started yet for that group.
  - **Green** — leg is still mathematically possible.
  - **Red** — leg is mathematically dead (predicted team cannot finish 1st).
  - **Gold / won** — tournament over, that leg came in.
- An accumulator row is marked dead the moment any leg dies.
- Show the current group standings for context (teams, points, GD, GF, games played).
- Show a "last updated" timestamp so users know how fresh the data is.
- Visual style: World Cup 2026 branding (official colour palette and feel).

### Data freshness

- During live match windows: data refreshes every 2–3 minutes.
- Outside match windows: data refreshes every 30–60 minutes.
- The frontend polls `data/state.json` on a short interval (e.g., 60 s) and re-renders if the file has changed (compare `updated_at` timestamp).

## Non-functional requirements

- Static hosting only — no server-side logic at request time.
- Fully public, no authentication.
- Site must load fast on mobile (friends will check from phones at the pub).
- The JSON data file must be small enough to fetch cheaply on every poll.

## Out of scope

- Per-user accounts or individual bet tracking.
- Knock-out round or top-scorer bets (group winners only for now).
- Editing bets at runtime.
- Push notifications.
- Historical results or bet history beyond the current tournament.
