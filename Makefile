# Agentic SDLC Platform — developer Makefile.
# Thin wrapper around docker compose. See SETUP.md.
#
# App services live behind the "app" profile until their milestone lands
# (ROADMAP.md). `make up` starts infrastructure only — the M0 baseline
# ("healthy Grafana + NATS; nothing else").

COMPOSE := docker compose
SVC     ?=

.DEFAULT_GOAL := help

.PHONY: help up up-app down restart logs ps build config test dev nuke ensure-secrets secrets-check

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	| awk 'BEGIN { FS = ":.*?## " } { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }'

up: ensure-secrets ## Start infrastructure only (M0: DBs, NATS, Redis, Vault, OTel, Grafana)
	$(COMPOSE) up -d

up-app: ensure-secrets ## Start infrastructure + the (stubbed) app services
	@grep -qE 'sk-[A-Za-z0-9_-]{8,}' .secrets/deepseek_api_key || echo "⚠️  No DeepSeek key in .secrets/deepseek_api_key — fine for M0–M2; needed at M3+."
	$(COMPOSE) --profile app up -d

ensure-secrets: ## Create placeholder secret files if missing (gitignored)
	@mkdir -p .secrets && test -f .secrets/deepseek_api_key || cp .secrets/deepseek_api_key.example .secrets/deepseek_api_key

secrets-check: ## Prove secrets are gitignored and report key presence
	@echo "== git-ignore check ==" && git check-ignore .env .secrets/deepseek_api_key >/dev/null && echo "  ok: .env and .secrets/deepseek_api_key are gitignored" || { echo "  ⚠️  NOT ignored — fix .gitignore before adding a real key!"; exit 1; }
	@echo "== key presence ==" && grep -qE 'sk-[A-Za-z0-9_-]{8,}' .secrets/deepseek_api_key && echo "  ok: DeepSeek key present" || echo "  ℹ️  DeepSeek key is empty/placeholder (ok for M0–M2; set it at M3+)"

build: config ## Build all images, including the app profile
	$(COMPOSE) --profile app build

config: ## Validate & print the resolved compose config
	@$(COMPOSE) --profile app config >/dev/null && echo "compose config OK"

down: ## Stop all services, keep volumes
	$(COMPOSE) down

restart: ## Restart all, or SVC=portal
	$(COMPOSE) restart $(SVC)

logs: ## Tail logs, optionally SVC=portal
	$(COMPOSE) logs -f --tail=200 $(SVC)

ps: ## Show container/service status
	$(COMPOSE) ps

test: ## Validate shared proto + compile-check the SDKs
	@python3 -c "import json; [json.load(open(f)) for f in ['shared/proto/envelope.schema.json','shared/proto/events.json']]" && echo "proto JSON OK"
	@python3 -m py_compile shared/sdk-py/aisdlc/*.py && echo "sdk-py OK"
	@if command -v go >/dev/null 2>&1; then (cd shared/sdk-go && go build ./...) && echo "sdk-go OK"; else echo "(go not installed — skip sdk-go)"; fi
	@if [ -x shared/sdk-ts/node_modules/.bin/tsc ]; then (cd shared/sdk-ts && ./node_modules/.bin/tsc --noEmit) && echo "sdk-ts OK"; elif command -v tsc >/dev/null 2>&1; then (cd shared/sdk-ts && tsc --noEmit) && echo "sdk-ts OK"; else echo "(tsc not installed — run 'npm install' in shared/sdk-ts; skip sdk-ts)"; fi

dev: ensure-secrets ## Hot-reload one service (SVC=portal) via docker-compose.dev.yml
	$(COMPOSE) --profile app -f docker-compose.yml -f docker-compose.dev.yml up $(SVC)

nuke: ## Stop everything AND delete all volumes (data loss!)
	@read -p "Are you sure? This destroys ALL data volumes! [y/N] " -n 1 && echo "" && [ "$$REPLY" = "y" ] && $(COMPOSE) down -v --remove-orphans || echo "aborted."
