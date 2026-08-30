# GoBank

Banking system developed using Golang!

**A minimal, REST-only Gin server** — register, login, create accounts, list accounts, transfer money, all behind JWT/PASETO bearer auth. No gRPC, no Redis, no email worker, no frontend.

> This repo used to also have a gRPC/gateway layer, a Redis-backed async email worker, and a Vue frontend calling both. That code (`gapi/`, `pb/`, `proto/`, `worker/`, `mail/`, `doc/`, `frontend/`, `val/`) has been removed, along with the now-unused dependencies in `go.mod` (`go mod tidy`'d down accordingly) — this repo is REST-API-only now. See "Restoring the full-featured version" at the bottom if you want any of it back; it's recoverable from git history, not gone for good.

## Stack

- **Backend:** Go, Gin, [sqlc](https://sqlc.dev/), PostgreSQL, PASETO (via [o1egl/paseto](https://github.com/o1egl/paseto)) / JWT auth

## Prerequisites

| Tool | Check | Install |
|---|---|---|
| Go 1.22+ | `go version` | https://go.dev/dl/ |
| Docker | `docker --version` | https://www.docker.com/ |
| [golang-migrate](https://github.com/golang-migrate/migrate) CLI | `migrate -version` | `brew install golang-migrate` |

## 1. Start Postgres

```bash
docker run --name postgres -p 5432:5432 \
  -e POSTGRES_USER=root -e POSTGRES_PASSWORD=mysecretpassword \
  -d postgres:14-alpine
docker exec -it postgres createdb --username=root --owner=root gobank
```

If a Postgres container is already running on `:5432` (check `docker ps`), skip this — just make sure `DB_SOURCE` in `app.env` matches its credentials and db name.

## 2. Run migrations

```bash
make migrateup
```

(`main.go` also runs pending migrations automatically on startup, so this step is optional — but running it explicitly first means startup failures are only ever about the app, not the schema.)

## 3. Run the server

```bash
go run main.go
```

You should see:

```
db migrated successfully
[GIN-debug] POST   /users                    --> ...
[GIN-debug] POST   /users/login              --> ...
[GIN-debug] POST   /tokens/renew_access      --> ...
[GIN-debug] POST   /accounts                 --> ...
[GIN-debug] GET    /accounts/:id             --> ...
[GIN-debug] GET    /accounts                 --> ...
[GIN-debug] GET    /accounts/lookup          --> ...
[GIN-debug] POST   /transfers                --> ...
[GIN-debug] Listening and serving HTTP on 0.0.0.0:8080
```

**Listening on `http://localhost:8080`.** Stop with Ctrl-C, or `lsof -ti:8080 -sTCP:LISTEN | xargs -r kill`.

## API reference (curl / Insomnia)

All bodies are JSON. Authenticated routes need `Authorization: bearer <access_token>` (lowercase `bearer` — matches how it's checked, case-insensitively, but keep it lowercase to match every other example in this codebase).

| Method | Path | Auth | Body |
|---|---|---|---|
| POST | `/users` | — | `{"username","password","full_name","email"}` |
| POST | `/users/login` | — | `{"username","password"}` |
| POST | `/tokens/renew_access` | — | `{"refresh_token"}` |
| POST | `/accounts` | Bearer | `{"currency"}` (`USD`, `EUR`, or `CAD`) |
| GET | `/accounts/:id` | Bearer | — |
| GET | `/accounts?page_id=&page_size=` | Bearer | — (`page_size` 5–10) |
| GET | `/accounts/lookup?owner=&currency=` | Bearer | — resolves a username to their account id in that currency, e.g. for addressing a transfer |
| POST | `/transfers` | Bearer | `{"from_account_id","to_account_id","amount","currency"}` |

### Full walkthrough

```bash
# 1. register
curl -X POST http://localhost:8080/users -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123","full_name":"Alice","email":"alice@example.com"}'

# 2. log in — grab the access token
TOKEN=$(curl -s -X POST http://localhost:8080/users/login -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])")

# 3. create an account
curl -X POST http://localhost:8080/accounts -H "Authorization: bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"currency":"USD"}'

# 4. list your accounts
curl "http://localhost:8080/accounts?page_id=1&page_size=10" -H "Authorization: bearer $TOKEN"

# 5. register a second user + account, so there's someone to transfer to
curl -X POST http://localhost:8080/users -H "Content-Type: application/json" \
  -d '{"username":"bob","password":"secret123","full_name":"Bob","email":"bob@example.com"}'
TOKEN_BOB=$(curl -s -X POST http://localhost:8080/users/login -H "Content-Type: application/json" \
  -d '{"username":"bob","password":"secret123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])")
curl -X POST http://localhost:8080/accounts -H "Authorization: bearer $TOKEN_BOB" -H "Content-Type: application/json" \
  -d '{"currency":"USD"}'

# 6. look up bob's account id (optional — you can also just use the id from step 5's response)
curl "http://localhost:8080/accounts/lookup?owner=bob&currency=USD" -H "Authorization: bearer $TOKEN"

# 7. transfer (account ids below assume alice=1, bob=2 — adjust to what steps 3/5 actually returned)
curl -X POST http://localhost:8080/transfers -H "Authorization: bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"from_account_id":1,"to_account_id":2,"amount":50,"currency":"USD"}'
```

**Step 7 will correctly return `422 {"error":"account balance is insufficient for this transfer"}`** — a brand-new account starts at balance `0` and there's no deposit endpoint, so alice has nothing to send yet. `TransferTx` (`db/sqlc/tx_transfer.go`) locks both accounts and checks the sender's real balance before writing anything; that rejection is the app working correctly, not a bug. To see a transfer actually succeed, fund an account directly for demo purposes:

```bash
docker exec postgres12 psql -U root -d gobank -c "UPDATE accounts SET balance = 100 WHERE id = 1;"
```

(Swap `postgres12` for your container's actual name if different — check `docker ps`.) Then retry step 7.

### Using Insomnia instead of curl

Import nothing special needed — just create requests matching the table above: set the method/URL, add a JSON body where listed, and for Bearer rows add a header `Authorization` with value `bearer <token>` (paste the `access_token` from the login response). Insomnia's environment variables are a natural fit for `{{ base_url }}` (`http://localhost:8080`) and `{{ token }}` if you want to chain requests without re-pasting.

## Reset to a clean slate

```bash
docker exec postgres12 psql -U root -d gobank -c "TRUNCATE TABLE transfers, entries, sessions, accounts, users RESTART IDENTITY CASCADE;"
```

## Tests

```bash
go test ./...
```

Requires Postgres reachable via `DB_SOURCE` in `app.env`. All packages pass clean — `go test ./...` -> `GoBank/api`, `GoBank/db/sqlc`, `GoBank/token`, `GoBank/util` all `ok`.

## Restoring the full-featured version

Before this REST-only demo, `main.go` started a gRPC server (`:9090`), an HTTP gateway with Swagger docs (`:8080`), this same REST API on a separate port (`:8081`), and a Redis-backed async email worker — all four in one process via `errgroup` — with a Vue frontend (`frontend/`) as a separate process calling both HTTP ports. Everything for that (`gapi/`, `pb/`, `proto/`, `worker/`, `mail/`, `doc/`, `frontend/`, `val/`, plus the fuller `main.go` and `go.mod`) is recoverable from commit `240018f` — e.g. `git show 240018f:main.go` to see it, or `git checkout 240018f -- gapi pb proto worker mail doc frontend val main.go go.mod go.sum` to bring it all back.
