# Deploy Pulse

Deploy Pulse is a single-workspace deployment observability MVP. It accepts signed CI/CD webhooks, normalizes provider states, stores sanitized deployment context, and presents a developer-focused release console.

## Run locally

```bash
go run ./cmd/deploypulse
```

Open [http://localhost:8080](http://localhost:8080). Demo deployments are inserted by default. Disable them with `DEMO_DATA=false`.

## Configure webhooks

Set a separate HMAC secret for each provider, or use `DEV_WEBHOOK_SECRET` during local development.

```bash
export DEV_WEBHOOK_SECRET=local-secret
export SECRET_NAMES=TOKEN,PASSWORD,API_KEY,DEPLOY_TOKEN
go run ./cmd/deploypulse
```

`POST /webhooks/{provider}` accepts `github`, `gitlab`, `circleci`, `vercel`, `netlify`, and `aws-codepipeline`. GitLab uses `X-Gitlab-Token`; the remaining adapters accept an SHA-256 HMAC in their provider header or `X-DeployPulse-Signature`.

## API

- `GET /api/deployments` supports `repository`, `branch`, `environment`, `provider`, `status`, `actor`, `start`, `end`, `cursor`, and `limit`.
- `GET /api/deployments/:id` includes ordered checks and no more than 200 sanitized log lines.
- `GET|POST /api/notification-rules`, `GET|POST /api/provider-connections`, and `GET /api/dead-letter-events` use `X-Role: admin` in this local MVP.
- `POST /api/dead-letter-events/:id/reprocess` requeues one parsing failure.

All API calls are scoped by `X-Workspace-ID`, which defaults to `demo`. Local SQLite data is written to `deploy-pulse.db`.

## Deliberate MVP boundaries

The local worker queue is in-process and notifications are recorded in the outbox as delivered. Redis-backed queues, actual Slack/email delivery, PostgreSQL runtime support, authenticated RBAC, and encrypted-key rotation are the production integrations to add next; the API contracts, tables, idempotency boundary, encrypted provider-secret storage, and fixtures are already in place.

## Verify

```bash
go test ./...
```
