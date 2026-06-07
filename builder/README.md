# builder

Go binary that polls football results, recalculates which bets are still alive,
computes a maximum possible payout, and — when a Betfair odds source is
configured — a win probability and expected payout for every bet. It writes
`state.json` (to a local file or S3) for the Svelte frontend to render.

It also produces a **daily summary** after each day's matches finish: a short
natural-language recap plus the biggest risers/fallers in bet likelihood and any
bets that landed or went bust. See [Daily summary](#daily-summary).

## Running locally

From this directory:

```bash
export FDB_API_KEY=...          # football-data.org API key (required)
export LOCAL_OUTPUT=./state.json # write to a file instead of S3
go run .
```

`BETS_FILE` defaults to `/config/bets.yaml`; when you run from `builder/` it
automatically falls back to `../config/bets.yaml`, so no flag is needed.

The process runs an infinite poll loop but writes on the **first** cycle. Watch
for the `wrote state file` log line, then Ctrl-C. Inspect the output with:

```bash
jq '.expected, (.bets[]|{id,status,probability,expected_return})' state.json
```

### Environment variables

| Var | Required | Default | Purpose |
|---|---|---|---|
| `FDB_API_KEY` | yes | — | football-data.org key (matches/standings/scorers) |
| `LOCAL_OUTPUT` | no | — | write `state.json` here instead of S3 |
| `BETS_FILE` | no | `/config/bets.yaml` | bet definitions (auto-falls-back to `../config/bets.yaml`) |
| `S3_BUCKET` | if not local | — | destination bucket |
| `S3_KEY` | no | `data/state.json` | object key |
| `AWS_REGION` | no | `eu-west-1` | bucket region |
| `BETFAIR_APP_KEY` | for odds | — | enables likelihoods; absent ⇒ results-only |
| `BETFAIR_SESSION_TOKEN` | — | — | pre-minted session token |
| `BETFAIR_USERNAME` / `BETFAIR_PASSWORD` | — | — | login credentials |
| `BETFAIR_CERT_FILE` / `BETFAIR_KEY_FILE` | — | — | PEM client cert + key for cert login |
| `ANTHROPIC_API_KEY` | no | — | enables Claude-written daily-summary recap; absent ⇒ templated sentence |
| `ANTHROPIC_MODEL` | no | `claude-opus-4-8` | model for the recap paragraph |
| `SUMMARY_TZ` | no | `America/New_York` | timezone used to group fixtures into a "tournament day" |
| `SUMMARY_KEY` | no | `data/daily-summary.json` | public summary object the frontend reads |
| `SUMMARY_STATE_KEY` | no | `data/summary-state.json` | builder-private probability snapshot store |
| `VAPID_PRIVATE_KEY` | for push | — | VAPID private key; enables match-result push notifications |
| `VAPID_PUBLIC_KEY` | for push | — | VAPID public key (must match the one baked into the frontend) |
| `VAPID_SUBJECT` | no | `https://betwithgoodall.com` | VAPID `sub` claim (a `mailto:` or `https:` URL) |
| `PUSH_TABLE` | for push | — | DynamoDB table of Web Push subscriptions |

Without `BETFAIR_APP_KEY` the builder degrades gracefully: it still tracks bet
status and max payout, just without probabilities or expected payout.

## Betfair odds

When `BETFAIR_APP_KEY` is set, the builder fetches de-vigged Exchange markets
(match 1X2, tournament winner, top goalscorer, group winners, correct score) on
a 15-minute throttle, calibrates team strengths to the WINNER market, and runs a
Monte Carlo tournament simulator each cycle to price every bet.

### Authentication

Auth is resolved in this order (first match wins):

1. **Session token** — `BETFAIR_SESSION_TOKEN` (mint it out-of-band).
2. **Certificate (bot) login** — `BETFAIR_CERT_FILE` + `BETFAIR_KEY_FILE` +
   `BETFAIR_USERNAME` + `BETFAIR_PASSWORD`. This is the right choice for
   unattended runs and for accounts with two-factor auth, which the interactive
   flow rejects. **The username and password are still required** — the
   certificate is an additional factor, not a replacement for credentials.
3. **Interactive login** — `BETFAIR_USERNAME` + `BETFAIR_PASSWORD` only.

### Creating the cert-login keypair

A self-signed certificate is sufficient; Betfair only needs the public cert on
file to match the key you present during the TLS handshake.

```bash
# 1. Generate an unencrypted 2048-bit RSA private key
openssl genrsa -out betfair.key 2048

# 2. Create a self-signed certificate from it (valid 10 years, no prompts)
openssl req -new -x509 -sha256 -days 3650 \
  -key betfair.key -out betfair.crt \
  -subj "/CN=bet-with-goodall"
```

Then:

1. Upload `betfair.crt` to Betfair: **My Account → Security → Automated Betting
   Program Access**. Allow a little time for it to propagate.
2. Point the builder at the files and run:

   ```bash
   export BETFAIR_CERT_FILE=/abs/path/betfair.crt
   export BETFAIR_KEY_FILE=/abs/path/betfair.key
   ```

On success you'll see `betfair auth: certificate login` → `betfair odds enabled`.

**Notes**
- The key must be **unencrypted** (no passphrase) — Go's `tls.LoadX509KeyPair`
  cannot decrypt a passphrase-protected key. `openssl genrsa` without
  `-aes256`/`-des3` produces an unencrypted key.
- Keep the `.key` private (`chmod 600`) and out of the repo. `*.key`/`*.crt`/
  `*.pem` are gitignored as a safeguard.
- The `CN` value is an arbitrary label for a self-signed cert.

## Daily summary

After every poll the builder checks whether a tournament day's fixtures have all
finished (fixtures are grouped by local date in `SUMMARY_TZ`, so a late-evening
North-American card stays on one day rather than splitting across UTC midnight).
The first time it sees a newly-completed day it:

1. snapshots every priced bet's current win probability,
2. diffs it against the previous day's close (or a pre-tournament baseline on
   day one), kept in `summary-state.json`,
3. ranks the biggest movers by **relative** change — so long-shot accumulators
   that swing in relative terms aren't buried under shorter-priced bets — while
   carrying the absolute percentage-point move too,
4. notes any bets that were alive at the previous close and are now won/bust,
5. renders a recap paragraph (Claude when `ANTHROPIC_API_KEY` is set, otherwise a
   templated sentence), and
6. prepends the result to a rolling archive in `daily-summary.json`.

The public `daily-summary.json` is what the frontend's Daily Summary card and
`/daily-summary` archive page read; it's a separate object from `state.json`
because it changes about once a day rather than every poll. A notification
channel (email/Slack/push) can hang off the same file or off the
`daily summary generated` log line later — none is wired up yet.

Run locally with a baseline that already has completed days by pointing
`FDB_API_KEY` at the live feed once the group stage is under way; with no
completed day yet it just records the pre-tournament baseline and writes nothing
to `daily-summary.json`.

## Push notifications

The site is an installable PWA, and the builder sends a Web Push notification
each time a match finishes — the score plus a one-line summary of how the
result moved the group's pending bets (Claude when `ANTHROPIC_API_KEY` is set,
otherwise a templated sentence built from the same risers/fallers/settled data
the daily summary uses).

**How it fits together:**

1. The frontend asks the browser to subscribe (with the VAPID *public* key) and
   POSTs the subscription to `/api/subscribe`, a CloudFront → API Gateway →
   Lambda → DynamoDB path provisioned in `infra/push.tf`.
2. Each poll cycle the builder diffs fixture statuses against the previous
   cycle. For any match that just turned `FINISHED`, it composes a notification
   and signs it with the VAPID *private* key, sending to every subscription in
   the DynamoDB table (`PUSH_TABLE`). Subscriptions the push service reports as
   gone (HTTP 404/410) are pruned.
3. The service worker (`web/src/sw.ts`) shows the notification and focuses/opens
   the site when it's tapped.

On a fresh start the builder seeds its fixture-status baseline without sending
anything, so a restart never replays already-finished matches.

**Generating the VAPID keypair** (once):

```bash
# With the web-push CLI (npx, no install):
npx web-push generate-vapid-keys
# → Public Key:  B....   Private Key:  ....
```

Then:

- Put the **private** key in SSM at `/homelab/bet-with-goodall/builder/vapid_private`
  and the **public** key at `/homelab/bet-with-goodall/builder/vapid_public`
  (both parameters are created by `infra/push.tf`). The ExternalSecret syncs
  them into the `bet-builder` Secret.
- Set the **public** key as the `VITE_VAPID_PUBLIC_KEY` GitHub Actions variable
  so it's baked into the frontend build. The public key on both sides must match.

Without `VAPID_PRIVATE_KEY` / `PUSH_TABLE` the builder skips pushes entirely and
everything else runs unchanged. Push is also disabled in `LOCAL_OUTPUT` mode.

## Tests

```bash
go test ./...
```
