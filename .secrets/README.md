# Secrets

This directory holds **gitignored** secret files. Only `*.example` files and
this `README.md` are tracked — see `../.gitignore` (`git check-ignore` confirms).

## DeepSeek API key

1. Create a key at <https://platform.deepseek.com> (it starts with `sk-`).
2. Copy the template and edit it:

   ```bash
   cp deepseek_api_key.example deepseek_api_key
   $EDITOR deepseek_api_key          # paste the key — no quotes, no spaces
   ```

3. Prove it's safe:

   ```bash
   make secrets-check
   ```

The key is mounted into the `llm-gateway` container as a Docker **secret** at
`/run/secrets/deepseek_api_key`. It is **not** an environment variable, so it
will not show up in `docker inspect` or `docker compose config` output.

### When is it needed?
Not until **M3** (first personas / self-build). The infrastructure and the stub
services (`make up`) run fine without it.

### Production
In production, fetch the key from **Vault** (the `vault` service in
`docker-compose.yml`) rather than a file on disk.

### If a key leaks
Revoke it immediately at <https://platform.deepseek.com>, generate a new one,
and update `deepseek_api_key`. If it was committed to git, rewrite history
(`git filter-repo`) and **rotate** — assume anything in git history is compromised.
