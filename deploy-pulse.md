# DeployPulse - Product Requirements Document

**Status:** Ready for implementation  
**Owner:** Platform Engineering  
**Version:** 1.0  
**Target:** Single workspace deployment observability MVP

## Product statement
Deploy Pulse receives CI/CD provider events and presents one normalized deployment timeline with status, checks, failure context, and alerting. It observes deployments; it never starts, retries, or changes a provider pipeline.

## Users, problem, and goals
Developers lose time switching between provider dashboards during a failed release. A **developer** investigates deployments, an **engineering manager** monitors delivery health, and an **admin** configures provider connections and alerts. The MVP target is p95 webhook acknowledgement below one second and p95 event-to-dashboard delay below ten seconds.

## Scope
Support GitHub Actions, GitLab CI, CircleCI, Vercel, Netlify, and AWS CodePipeline. Include webhook ingestion, status normalization, deployment detail, filtered history, sanitized log excerpts, and Slack/email notifications. Exclude credential-based polling, triggering deployments, full log archival, and incident management workflows.

## Functional requirements
- **DEP-01:** `POST /webhooks/:provider` validates the provider signature before accepting an event; invalid signatures return 401 and are not persisted as deployments.
- **DEP-02:** The handler records raw payload metadata and enqueues a job using `{provider, providerEventId}` as an idempotency key. Duplicate delivery must produce exactly one normalized event.
- **DEP-03:** Normalize provider states into `queued`, `running`, `success`, `failed`, `cancelled`, or `unknown`. Preserve original state and provider URL.
- **DEP-04:** A deployment contains repository, branch, commit SHA, environment, actor, started/finished timestamps, duration, status, and a provider deep link.
- **DEP-05:** `GET /api/deployments` filters by repository, branch, environment, provider, status, actor, and time range; results paginate by cursor.
- **DEP-06:** `GET /api/deployments/:id` returns deployment details, ordered checks, sanitized failure summary, and up to 200 most recent log lines.
- **DEP-07:** Redact values matching configured secret names and token/password/private-key patterns before logs are stored or returned. Raw logs are not retained in MVP.
- **DEP-08:** Admins create notification rules by repository/environment/status. A failed event sends one alert, while a later success sends a recovery notification.
- **DEP-09:** Failed parsing enters a dead-letter table with error, retry count, and raw payload reference; admins can reprocess a single item.

## Architecture and data
Run an HTTP ingestion service plus a Go worker. SQLite is acceptable locally; production schema targets PostgreSQL. Tables: `provider_connections`, `repositories`, `webhook_events`, `deployments`, `deployment_checks`, `log_excerpts`, `notification_rules`, `notification_deliveries`, and `dead_letter_events`. Redis backs the job queue and deduplication locks.

Each adapter implements `verifySignature`, `parseEvent`, and `normalizeState`; adapter fixtures are mandatory. Persist a structured `failure_category` such as `test_failure`, `build_failure`, `deploy_failure`, `cancelled`, or `unknown`.

## Quality, acceptance, and delivery
Secrets are encrypted, RBAC is workspace-scoped, and all logs contain a correlation ID. Retain event metadata 90 days and sanitized excerpts 30 days. The MVP is accepted when fixtures for all six providers pass, duplicate webhooks do not duplicate deployments, a failed provider adapter cannot block another provider, redaction tests cover representative secrets, and a recovery alert is sent once.

Deliver GitHub ingestion and canonical model first, then remaining adapters and list UI/API, then logs/notifications, and finally DLQ, load tests, and security review.

## Stitch Prompt 
