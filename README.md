# Deploy Pulse

Deploy Pulse ingests signed deployment webhooks, stores the raw event durably, then processes it through Redis Streams into a sanitized deployment timeline. PostgreSQL is the production store; SQLite is retained for local development and tests.

## Run production locally

```bash
cp .env.compose.example .env
# Replace every change-me value with a unique secret.
docker compose up --build
```

The API is at [http://localhost:8080](http://localhost:8080). Compose starts PostgreSQL and Redis first, runs idempotent migrations, then starts the API and worker. Check liveness with `/healthz` and dependency readiness with `/readyz`.

The runtime roles are separate:

- `deploypulse api` serves the UI and HTTP endpoints.
- `deploypulse worker` consumes `deploypulse:events` with consumer group `deploypulse-workers`.
- `deploypulse migrate` applies entries in `schema_migrations` and can be rerun safely.

For SQLite-only parser and API tests, use `go test ./...`. The production API and worker require `REDIS_URL`; use Compose for an end-to-end local runtime.

## Webhooks

Configure only the provider secrets you use. Each endpoint is `POST /webhooks/{provider}` and rejects a missing or invalid provider-specific signature before persistence.

| Provider | Environment variable | Verification |
| --- | --- | --- |
| GitHub Actions | `WEBHOOK_SECRET_GITHUB` | `X-Hub-Signature-256`, HMAC-SHA256 |
| GitLab CI | `WEBHOOK_SECRET_GITLAB` | GitLab signing token (`Webhook-Signature`) or legacy `X-Gitlab-Token` |
| CircleCI | `WEBHOOK_SECRET_CIRCLECI` | `Circleci-Signature` v1, HMAC-SHA256 |
| Vercel | `WEBHOOK_SECRET_VERCEL` | `X-Vercel-Signature`, HMAC-SHA1 |
| Netlify | `WEBHOOK_SECRET_NETLIFY` | `X-Webhook-Signature` HS256 JWS |
| AWS CodePipeline | `AWS_RELAY_SECRET` | internal `X-DeployPulse-Relay-Signature`, HMAC-SHA256 |

Webhook writes are idempotent by `(workspace_id, provider, provider_event_id)`. Redis is at-least-once: the worker calls `XACK` only after the database transaction for the deployment and webhook status commits. Pending deliveries are reclaimed by `XAUTOCLAIM`; after five attempts, the worker writes both the Redis DLQ stream and the database DLQ.

`DEFAULT_WORKSPACE_ID` scopes this release to one workspace. Request-provided workspace headers are intentionally ignored.

## Administration

`ADMIN_API_TOKEN` is required for provider connections, notification rules, DLQ inspection, and DLQ reprocessing. Send it only from an administrative deployment or client:

```bash
curl -H "Authorization: Bearer $ADMIN_API_TOKEN" \
  http://localhost:8080/api/dead-letter-events
```

The browser dashboard never stores or sends this token. It displays operational health, connected providers, and DLQ count from `GET /api/health`.

Provider secrets saved through the API are encrypted with `ENCRYPTION_KEY`; the API never returns them. `ENCRYPTION_KEY` is mandatory whenever `APP_ENV=production`.

## Operate

Back up PostgreSQL before upgrades or token rotations:

```bash
docker compose exec -T postgres pg_dump -U deploypulse deploypulse > deploypulse-$(date +%F).sql
```

Restore into a stopped/replacement database with `psql -U deploypulse deploypulse < backup.sql`, then start Compose and let the migrator run.

To rotate `ADMIN_API_TOKEN`, update `.env`/the secret manager and restart API containers; all previous tokens stop working immediately. To rotate `ENCRYPTION_KEY`, first re-encrypt saved provider secrets with a dedicated migration or re-enter each provider secret—do not simply change the variable. Rotate a webhook secret at its provider and in the deployment together, accepting both only during a separately planned transition.

To reprocess a dead letter after fixing the cause:

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_API_TOKEN" \
  http://localhost:8080/api/dead-letter-events/<dlq-id>/reprocess
```

The endpoint marks the event queued, publishes it, and only then clears its DLQ record. A duplicate is harmless because the deployment unique key remains enforced by PostgreSQL.

## AWS CodePipeline relay

Deploy `infra/aws/eventbridge-relay` with SAM. Its EventBridge rule accepts only CodePipeline pipeline-execution state events; the Lambda validates that source and detail type again before forwarding the untouched event JSON to `/webhooks/aws-codepipeline` with the internal HMAC header.

```bash
cd infra/aws/eventbridge-relay
sam build
sam deploy --guided
```

Set `DeployPulseWebhookURL` to the public AWS webhook endpoint and use the exact same value for `RelaySharedSecret` and `AWS_RELAY_SECRET`.

## Verify

```bash
go test ./...
docker compose config
```

Run the included local load check after Compose is healthy:

```bash
WEBHOOK_SECRET_GITHUB="$WEBHOOK_SECRET_GITHUB" scripts/load-webhooks.sh
```

It sends 100 parallel signed GitHub fixture requests and reports acknowledgement p95. The intended local Compose target is below one second, with the worker showing the deployment in under ten seconds.
