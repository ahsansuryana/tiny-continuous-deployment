# DeployHub

Lightweight self-hosted deployment manager. Minimal alternative to Dokploy/Coolify for homelabs and small servers.

## Features

- **Docker Compose** based deployments
- **Webhook** endpoint for GitHub Actions / CI/CD
- **Real-time** deployment logs with HTMX streaming
- **Single binary** — Go + SQLite, no external dependencies
- **Dense UI** — GitHub/Vercel-inspired dark theme
- **Traefik** example configuration generator

## Quick Start

```bash
# Clone and configure
cp .env.example .env
# Edit .env with your admin credentials

# Start with Docker Compose
docker compose up -d

# Open http://localhost:8080
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_USERNAME` | — | Admin login username |
| `ADMIN_PASSWORD` | — | Admin login password |
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `DATA_DIR` | `/data` | Data directory for SQLite DB |

### Docker Compose

```yaml
services:
  deployhub:
    build: .
    ports:
      - "8080:8080"
    env_file: .env
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock
```

## GitHub Actions Integration

```yaml
deploy:
  steps:
    - name: Deploy to DeployHub
      run: |
        curl -X POST \
          -H "Authorization: Bearer ${{ secrets.DEPLOY_TOKEN }}" \
          https://deploy.example.com/api/deploy/frontend
```

Add `DEPLOY_TOKEN` as a repository secret — it must match the webhook token configured in your project.

## Webhook API

```http
POST /api/deploy/{project-name}
Authorization: Bearer {webhook-token}
```

Response:

```json
{
  "success": true,
  "deployment_id": 123
}
```

## Traefik Configuration

Set your hostname in Settings → Traefik and click "Show Example" for a ready-to-use configuration.

## Development

```bash
# Install dependencies
npm install
go mod download

# Build CSS
npm run css

# Build and run
go build -o bin/deployhub ./cmd/server
./bin/deployhub
```

## Architecture

```
┌─────────┐     ┌──────────────┐     ┌──────────┐
│ Browser │────▶│  Go Server   │────▶│  SQLite  │
│ (HTMX)  │     │  (net/http)  │     │          │
└─────────┘     │              │     └──────────┘
                │  /api/deploy │──┐  ┌──────────┐
┌─────────┐     │  (webhook)   │  └─▶│  Docker  │
│ GitHub  │────▶│              │     │  Compose │
│ Actions │     └──────────────┘     └──────────┘
```

## Performance Targets

- Idle RAM: < 30 MB
- Startup: < 1 second
- Single binary, no runtime dependencies
- SQLite only — no PostgreSQL, no Redis

## License

MIT
# tiny-continuous-deployment
