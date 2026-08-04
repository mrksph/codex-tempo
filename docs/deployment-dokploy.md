# Dokploy deployment

Production runs as a Dokploy Docker Compose workload on the `atlas` server.
Dokploy clones the private GitHub repository and builds both application images
from `deploy/compose/docker-compose.yml`.

## Runtime configuration

The production Compose environment must define:

- `POSTGRES_PASSWORD`
- `AGENT_SETUP_KEY`
- `INTERNAL_API_TOKEN`
- `WEB_PASSWORD`
- `AUTH_SECRET`
- `PUBLIC_API_URL`
- `TZ`
- `WEB_PORT`
- `API_PORT`

The PostgreSQL data directory is stored in the persistent
`postgres-data` Docker volume.

## Delivery flow

1. Every push and pull request runs `.github/workflows/ci.yml`.
2. A successful `CI` run on `master` starts `.github/workflows/deploy.yml`.
3. The deploy workflow calls the Dokploy API through its CLI.
4. Dokploy checks out `master`, builds the Compose services and performs the
  deployment on Atlas.

The GitHub repository stores only the Dokploy endpoint and resource identifier
as Actions secrets. Runtime application secrets remain in Dokploy.

## GitHub Actions secrets

- `DOKPLOY_URL`
- `DOKPLOY_AUTH_TOKEN`
- `DOKPLOY_COMPOSE_ID`

Production can also be redeployed manually from the `Deploy` workflow or with:

```bash
dokploy compose deploy --composeId "$DOKPLOY_COMPOSE_ID"
```
