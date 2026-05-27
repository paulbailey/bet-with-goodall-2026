# Architecture

## System overview

```
                                 ┌──────────────────────────────┐
                                 │        AWS (Terraform)        │
                                 │                              │
 homelab k8s cluster             │   ┌────────┐   ┌──────────┐ │
 ┌─────────────────────────┐     │   │   S3   │   │CloudFront│ │
 │  builder Deployment     │────────▶│ bucket │──▶│  distrib │ │
 │  (Go binary)            │     │   │        │   └──────────┘ │
 │  - polls results API    │     │   │ web/*  │         │      │
 │  - evaluates bet legs   │     │   │ data/  │         │      │
 │  - writes state.json    │     │   │state.  │         ▼      │
 └─────────────────────────┘     │   │ json   │     browsers   │
          │                      │   └────────┘               │
          │ IRSA                 └──────────────────────────────┘
          ▼
    AWS IAM role
    (s3:PutObject on bucket)
```

## Components

### 1. Config (`config/bets.yaml`)

Committed to the repo, deployed as a ConfigMap or bundled into the builder image. Defines all bets. Never changes during the tournament. See [Data formats](#data-formats) below.

### 2. Go builder (`builder/`)

A long-running Go binary deployed as a k8s **Deployment** (1 replica).

**Why Deployment over CronJob**: the builder maintains in-memory state about the match schedule so it can self-regulate its polling frequency without re-fetching the schedule on every wake. A Deployment keeps this alive; a CronJob would require persisting state externally.

**Responsibilities:**
1. On startup, load `config/bets.yaml` and fetch the full match schedule from the results API.
2. Enter a poll loop:
   - Fetch current group standings and completed match results.
   - For each bet leg, evaluate whether the predicted team can still mathematically win their group.
   - Serialize the full state to `data/state.json`.
   - Upload `data/state.json` to S3.
   - Sleep until next poll (short during match windows, long otherwise).
3. Refresh the match schedule periodically (once per day is sufficient).

**Smart polling:**
- Define a "match window" as: any match within the past 3 hours or the next 30 minutes.
- During a match window: poll every 2 minutes.
- Outside a match window: poll every 30 minutes.
- Compute time-to-next-match on each iteration to set the sleep duration precisely.

**AWS auth:** IRSA via a k8s ServiceAccount annotated with the IAM role ARN. No credentials in the image or environment.

**Data feed:** Abstracted behind a `ResultsProvider` interface so the implementation can be swapped. Initial implementation will target **football-data.org** (free tier, 10-second delay — acceptable for a bet tracker). API-Football (RapidAPI) is the paid fallback.

```go
type ResultsProvider interface {
    GetGroups(ctx context.Context) ([]Group, error)
    GetMatches(ctx context.Context) ([]Match, error)
}
```

**Bet evaluation logic (group-winner legs):**

After fetching the latest standings for each group, for each predicted team:
1. Count games remaining for every team in the group.
2. Compute each team's maximum achievable points: `current_points + 3 * games_remaining`.
3. If the predicted team's max_points < any other team's current_points, the leg is **dead** (the other team already can't be caught in points — ignoring GD tiebreakers for simplicity at this stage).
4. Once a group is complete (all 3 games played by all teams), the winner is known definitively; mark the leg won or lost.

An accumulator's `status` is:
- `won` — all legs won (all groups complete and correct).
- `lost` — at least one leg is dead.
- `alive` — no legs dead, tournament ongoing.

### 3. React frontend (`web/`)

Built with **Vite + React + TypeScript**.

**No build-time data** — the app fetches `data/state.json` at runtime from the same CloudFront origin. This keeps the build/deploy pipeline simple: Terraform/CI pushes the built `web/dist/` to S3 once; the builder updates `data/state.json` independently.

**Polling:** `useEffect` with a `setInterval` (60 s). On each tick, fetch `state.json` and compare `updated_at` to avoid unnecessary re-renders.

**Key views:**
- **Bet grid** — rows = accumulators, columns = groups A–L. Cell shows predicted team name + status colour. Row header shows overall accumulator status.
- **Group standings panel** — expandable or side-by-side; shows live group tables for context.
- **Last updated** — footer timestamp.

**Visual style:** World Cup 2026 official branding. Use the official colours (deep red `#C8102E`, dark navy `#003087`, gold `#FDB913`) and typography where possible.

### 4. Terraform (`infra/`)

Manages:
- S3 bucket (private, versioning on, no public access).
- CloudFront distribution (OAC → S3, HTTPS only, default TTL 60 s so data refreshes are visible promptly).
- IAM role for the builder (trust policy: `sts:AssumeRoleWithWebIdentity` from the homelab cluster's OIDC provider, limited to `s3:PutObject` and `s3:GetObject` on the bucket).
- Optional: Route 53 record + ACM certificate for a custom domain.

State backend: Terraform Cloud.

### 5. Kubernetes manifests (`builder/k8s/` + root `kustomization.yaml`)

- `Deployment` for the builder.
- `ServiceAccount` annotated with the IAM role ARN.
- `ExternalSecret` for the results API key (referenced as env var in the Deployment).
- `ConfigMap` for `bets.yaml` — generated by Kustomize from `config/bets.yaml` at the repo root, so the bet definitions live in exactly one file. ArgoCD syncs the kustomization (Application `path: .`).

---

## Data formats

### `config/bets.yaml`

```yaml
bets:
  - id: bet-1
    stake: 10.00             # optional, £
    potential_return: 847.50 # optional, bookmaker's quoted payout
    legs:
      - group: A
        team: USA
      - group: B
        team: England
      # ...

  - id: bet-2
    stake: 5.00
    legs:
      - group: C
        team: Argentina
      # ...

top_scorer_bets:
  - id: tsb-1
    player: "Kylian Mbappé"
    team: France
    stake: 5.00
    potential_return: 37.50
```

### `data/state.json`

Written by the builder, read by the frontend. Schema:

```jsonc
{
  "updated_at": "2026-06-15T14:32:00Z",    // RFC3339
  "tournament_phase": "group_stage",        // pre_tournament | group_stage | knockout | complete
  "groups": {
    "A": {
      "standings": [
        {
          "team": "USA",
          "played": 2, "won": 1, "drawn": 0, "lost": 1,
          "gf": 2, "ga": 1, "gd": 1, "points": 3
        }
        // ...
      ],
      "complete": false,   // true once all 3 matchdays played
      "winner": null       // team name once complete, else null
    }
    // groups B–L
  },
  "bets": [
    {
      "id": "bet-1",
      "stake": 10.00,              // optional
      "potential_return": 847.50,  // optional, bookmaker's quoted payout
      "status": "alive",  // alive | lost | won
      "legs": [
        {
          "group": "A",
          "team": "USA",
          "status": "alive"  // alive | lost | won | pending
        }
        // ...
      ]
    }
  ],
  "top_scorer_bets": [
    {
      "id": "tsb-1",
      "player": "Kylian Mbappé",
      "team": "France",
      "stake": 5.00,              // optional
      "potential_return": 37.50,  // optional
      "status": "alive"           // alive | lost | won
    }
  ],
  "top_scorers": [
    {
      "player": "Kylian Mbappé",
      "team": "France",
      "goals": 4,
      "games": 2,
      "team_eliminated": false  // true once team is out; locks in their goal tally
    }
  ]
}
```

---

## Deployment flow

1. **One-time infra setup:** `cd infra && terraform apply` — creates bucket, CloudFront, IAM role.
2. **Before tournament:** Edit `config/bets.yaml` with final bets. Commit and push.
3. **Build frontend:** `cd web && npm run build` → upload `dist/` to S3 (CI or manual).
4. **Start builder:** ArgoCD syncs the root kustomization (`path: .`), which generates the ConfigMap from `config/bets.yaml` and applies the rest of `builder/k8s/`. The builder starts polling and writes `data/state.json`.
5. **During tournament:** Builder runs continuously. Friends open the CloudFront URL and see live updates.

---

## Open questions / decisions deferred

| Question | Status |
|---|---|
| Which football results API? | TBD — football-data.org is the default plan |
| Custom domain for the site? | TBD |
| CI/CD for frontend deploys? | TBD — GitHub Actions is the obvious choice |
| ConfigMap vs image-bundled bets.yaml? | Resolved — Kustomize `configMapGenerator` reads `config/bets.yaml` so it's the single source of truth (no rebuild to change bets pre-tournament). |
| GD/GH tiebreaker in impossibility calc? | Deferred — points-only check is good enough for early group stage |
