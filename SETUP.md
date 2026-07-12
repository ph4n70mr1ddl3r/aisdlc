# Local Developer Setup — Agentic SDLC Platform

Everything runs via Docker Compose. No host installs beyond Docker + make.

## Prerequisites
- Docker 24+ with Compose v2
- `make`
- (optional) `jq`, `httpie` for poking the API

## Quick start
```bash
cp .env.example .env          # set LLM keys, secrets
make up                       # infra only today (M0): DBs, NATS, OTel, Grafana
# make up-app                 # also boot the (stubbed) app services
make logs                     # tail everything
make ps                       # service status
# open http://localhost:3000  # portal — ships in M1 (model-driven core)
open http://localhost:3001    # grafana (admin/admin)
```

## Secrets (API keys)
Real secrets are **never** committed. Two layers keep them safe:

1. `.gitignore` blocks `.env` and `.secrets/*` (only `*.example` + `README.md`
   are tracked). Verify any time with `make secrets-check`.
2. The DeepSeek API key is a Docker Compose **secret** (file-based), so it is
   **not** an env var and never appears in `docker inspect` or
   `docker compose config`.

Add the DeepSeek key (needed at **M3+** only; infra/stubs run fine without it):

```bash
cp .secrets/deepseek_api_key.example .secrets/deepseek_api_key
$EDITOR .secrets/deepseek_api_key     # paste your sk-... key
make secrets-check                    # prove it's gitignored + present
```

The key is mounted into `llm-gateway` at `/run/secrets/deepseek_api_key`.
In production, source it from Vault (the `vault` service in `docker-compose.yml`)
instead of a file.
If a key is ever committed, rotate it immediately and rewrite git history.

## Teardown
```bash
make down                     # stop, keep volumes
make nuke                     # stop + delete volumes (⚠️ wipes data)
```

## Per-service dev loop
Each service has hot reload via `docker-compose.dev.yml` (overrides apply once
real source lands under `services/`):
```bash
make dev SVC=portal           # portal with file watch + source mount
```

## Running a single agent locally
The worker pool + personas land in **M3**. Once they exist you can drive one
task by hand:
```bash
docker compose --profile app run --rm agent-runtime \
    python -m agent.run --task 42 --dry-run
```

## Useful endpoints (dev defaults)
| Service | URL |
|---|---|
| Portal | http://localhost:3000 |
| Gateway API | http://localhost:8080 |
| NATS monitor | http://localhost:8222 |
| Grafana | http://localhost:3001 |
| Loki | http://localhost:3100 |
| Qdrant dashboard | http://localhost:6333/dashboard |
| Vault UI | http://localhost:8200 |

## Layout
See [PLAN.md §8](./PLAN.md). Infra in `infra/`, services in `services/`,
shared code in `shared/`.
