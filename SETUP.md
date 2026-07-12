# Local Developer Setup — Agentic SDLC Platform

Everything runs via Docker Compose. No host installs beyond Docker + make.

## Prerequisites
- Docker 24+ with Compose v2
- `make`
- (optional) `jq`, `httpie` for poking the API

## Quick start
```bash
cp .env.example .env          # set LLM keys, secrets
make up                       # docker compose up -d (infra + services)
make logs                     # tail everything
make ps                       # service status
open http://localhost:3000    # portal
open http://localhost:3001    # grafana (admin/admin)
```

## Teardown
```bash
make down                     # stop, keep volumes
make nuke                     # stop + delete volumes (⚠️ wipes data)
```

## Per-service dev loop
Each service has hot reload via `docker-compose.dev.yml`:
```bash
make dev SVC=portal           # portal with file watch + source mount
```

## Running a single agent locally
```bash
docker compose run --rm agent-requirements \
    python -m agent.run --ticket 42 --dry-run
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
| Mailhog | http://localhost:8025 |

## Layout
See [PLAN.md §8](./PLAN.md). Infra in `infra/`, services in `services/`,
shared code in `shared/`.
