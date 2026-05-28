# builder

Go binary that polls football results, recalculates which bets are still alive,
computes a maximum possible payout, and — when a Betfair odds source is
configured — a win probability and expected payout for every bet. It writes
`state.json` (to a local file or S3) for the Svelte frontend to render.

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

## Tests

```bash
go test ./...
```
