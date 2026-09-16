package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type Store struct {
	db       *sql.DB
	postgres bool
}

// OpenStore accepts a PostgreSQL URL in production and a SQLite path locally.
func OpenStore(ctx context.Context, databaseURL string) (*Store, error) {
	postgres := strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://")
	driver, dsn := "sqlite", databaseURL+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	if postgres {
		driver, dsn = "postgres", databaseURL
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db, postgres: postgres}
	if err := store.db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error                      { return s.db.Close() }
func (s *Store) Healthy(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) rebind(query string) string {
	if !s.postgres {
		return query
	}
	var result strings.Builder
	result.Grow(len(query) + 8)
	index := 0
	for _, char := range query {
		if char == '?' {
			index++
			fmt.Fprintf(&result, "$%d", index)
			continue
		}
		result.WriteRune(char)
	}
	return result.String()
}

func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, s.rebind(query), args...)
}
func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, s.rebind(query), args...)
}
func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, s.rebind(query), args...)
}
func (s *Store) txExec(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(ctx, s.rebind(query), args...)
}

type migration struct {
	version string
	sql     string
}

var migrations = []migration{{version: "001_initial", sql: `
CREATE TABLE IF NOT EXISTS provider_connections (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, provider TEXT NOT NULL, name TEXT NOT NULL,
  secret_encrypted TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(workspace_id, provider)
);
CREATE TABLE IF NOT EXISTS repositories (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, name TEXT NOT NULL, created_at TEXT NOT NULL,
  UNIQUE(workspace_id, name)
);
CREATE TABLE IF NOT EXISTS webhook_events (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, provider TEXT NOT NULL, provider_event_id TEXT NOT NULL,
  payload BYTEA NOT NULL, payload_hash TEXT NOT NULL, correlation_id TEXT NOT NULL, status TEXT NOT NULL,
  received_at TEXT NOT NULL, processed_at TEXT, UNIQUE(workspace_id, provider, provider_event_id)
);
CREATE TABLE IF NOT EXISTS deployments (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, provider TEXT NOT NULL, provider_event_id TEXT NOT NULL,
  provider_url TEXT NOT NULL, repository TEXT NOT NULL, branch TEXT NOT NULL, commit_sha TEXT NOT NULL,
  environment TEXT NOT NULL, actor TEXT NOT NULL, status TEXT NOT NULL, original_status TEXT NOT NULL,
  failure_category TEXT NOT NULL, failure_summary TEXT NOT NULL, started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL, duration TEXT NOT NULL, created_at TEXT NOT NULL,
  UNIQUE(workspace_id, provider, provider_event_id)
);
CREATE TABLE IF NOT EXISTS deployment_checks (
  id TEXT PRIMARY KEY, deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  name TEXT NOT NULL, status TEXT NOT NULL, summary TEXT NOT NULL, started_at TEXT NOT NULL, finished_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS log_excerpts (
  id TEXT PRIMARY KEY, deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  line_number INTEGER NOT NULL, content TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notification_rules (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, repository TEXT NOT NULL, environment TEXT NOT NULL,
  status TEXT NOT NULL, channel TEXT NOT NULL, target TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS notification_deliveries (
  id TEXT PRIMARY KEY, deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  rule_id TEXT NOT NULL REFERENCES notification_rules(id) ON DELETE CASCADE, kind TEXT NOT NULL,
  status TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(deployment_id, rule_id, kind)
);
CREATE TABLE IF NOT EXISTS dead_letter_events (
  id TEXT PRIMARY KEY, webhook_event_id TEXT NOT NULL UNIQUE REFERENCES webhook_events(id) ON DELETE CASCADE,
  provider TEXT NOT NULL, error TEXT NOT NULL, retry_count INTEGER NOT NULL, raw_payload_ref TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS deployments_workspace_started ON deployments(workspace_id, started_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS webhook_events_status ON webhook_events(status, received_at);
`}}

// Migrate is safe to run repeatedly and is also exposed for the Compose migrator.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	for _, migration := range migrations {
		var applied string
		err := s.queryRow(ctx, `SELECT version FROM schema_migrations WHERE version=?`, migration.version).Scan(&applied)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, migration.sql); err == nil {
			_, err = s.txExec(ctx, tx, `INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`, migration.version, now())
		}
		if err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func (s *Store) PruneRetained(ctx context.Context) error {
	if _, err := s.exec(ctx, `DELETE FROM log_excerpts WHERE created_at < ?`, time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	_, err := s.exec(ctx, `DELETE FROM webhook_events WHERE received_at < ?`, time.Now().UTC().AddDate(0, 0, -90).Format(time.RFC3339Nano))
	return err
}

func (s *Store) RecordWebhook(ctx context.Context, event WebhookEvent) (bool, error) {
	result, err := s.exec(ctx, `INSERT INTO webhook_events
  (id,workspace_id,provider,provider_event_id,payload,payload_hash,correlation_id,status,received_at)
  VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,provider,provider_event_id) DO NOTHING`,
		event.ID, event.WorkspaceID, event.Provider, event.ProviderEventID, event.Payload, event.PayloadHash,
		event.CorrelationID, "queued", event.ReceivedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) Webhook(ctx context.Context, id string) (WebhookEvent, error) {
	var event WebhookEvent
	var received string
	err := s.queryRow(ctx, `SELECT id,workspace_id,provider,provider_event_id,payload,payload_hash,correlation_id,status,received_at FROM webhook_events WHERE id=?`, id).
		Scan(&event.ID, &event.WorkspaceID, &event.Provider, &event.ProviderEventID, &event.Payload, &event.PayloadHash, &event.CorrelationID, &event.Status, &received)
	if err != nil {
		return event, err
	}
	event.ReceivedAt, _ = time.Parse(time.RFC3339Nano, received)
	return event, nil
}

func (s *Store) MarkWebhook(ctx context.Context, id, status string) error {
	_, err := s.exec(ctx, `UPDATE webhook_events SET status=?, processed_at=? WHERE id=?`, status, now(), id)
	return err
}

func (s *Store) SaveDeployment(ctx context.Context, deployment Deployment, checks []DeploymentCheck, logs []string) (bool, error) {
	return s.saveDeployment(ctx, "", deployment, checks, logs)
}

// SaveProcessedDeployment commits a deployment and its webhook state together.
func (s *Store) SaveProcessedDeployment(ctx context.Context, webhookID string, deployment Deployment, checks []DeploymentCheck, logs []string) (bool, error) {
	return s.saveDeployment(ctx, webhookID, deployment, checks, logs)
}

func (s *Store) saveDeployment(ctx context.Context, webhookID string, deployment Deployment, checks []DeploymentCheck, logs []string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := s.txExec(ctx, tx, `INSERT INTO deployments
  (id,workspace_id,provider,provider_event_id,provider_url,repository,branch,commit_sha,environment,actor,status,original_status,failure_category,failure_summary,started_at,finished_at,duration,created_at)
  VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,provider,provider_event_id) DO NOTHING`,
		deployment.ID, deployment.WorkspaceID, deployment.Provider, deployment.ProviderEventID, deployment.ProviderURL,
		deployment.Repository, deployment.Branch, deployment.CommitSHA, deployment.Environment, deployment.Actor,
		deployment.Status, deployment.OriginalStatus, deployment.FailureCategory, deployment.FailureSummary,
		deployment.StartedAt.UTC().Format(time.RFC3339Nano), deployment.FinishedAt.UTC().Format(time.RFC3339Nano), deployment.Duration, deployment.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	created, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if created == 1 {
		for _, check := range checks {
			if _, err := s.txExec(ctx, tx, `INSERT INTO deployment_checks(id,deployment_id,name,status,summary,started_at,finished_at) VALUES(?,?,?,?,?,?,?)`,
				check.ID, deployment.ID, check.Name, check.Status, check.Summary, check.StartedAt, check.FinishedAt); err != nil {
				return false, err
			}
		}
		for i, line := range logs {
			if _, err := s.txExec(ctx, tx, `INSERT INTO log_excerpts(id,deployment_id,line_number,content,created_at) VALUES(?,?,?,?,?)`, uuid.NewString(), deployment.ID, i+1, line, now()); err != nil {
				return false, err
			}
		}
		if _, err := s.txExec(ctx, tx, `INSERT INTO repositories(id,workspace_id,name,created_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,name) DO NOTHING`, uuid.NewString(), deployment.WorkspaceID, deployment.Repository, now()); err != nil {
			return false, err
		}
	}
	if webhookID != "" {
		if _, err := s.txExec(ctx, tx, `UPDATE webhook_events SET status='processed', processed_at=? WHERE id=?`, now(), webhookID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return created == 1, nil
}

func (s *Store) ListDeployments(ctx context.Context, workspaceID string, f ListFilter) ([]Deployment, string, error) {
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	clauses, args := []string{"workspace_id=?"}, []any{workspaceID}
	for _, filter := range []struct{ column, value string }{{"repository", f.Repository}, {"branch", f.Branch}, {"environment", f.Environment}, {"provider", f.Provider}, {"status", f.Status}, {"actor", f.Actor}} {
		if filter.value != "" {
			clauses, args = append(clauses, filter.column+"=?"), append(args, filter.value)
		}
	}
	if !f.Start.IsZero() {
		clauses, args = append(clauses, "started_at>=?"), append(args, f.Start.UTC().Format(time.RFC3339Nano))
	}
	if !f.End.IsZero() {
		clauses, args = append(clauses, "started_at<=?"), append(args, f.End.UTC().Format(time.RFC3339Nano))
	}
	if at, id, ok := decodeCursor(f.Cursor); ok {
		clauses, args = append(clauses, "(started_at < ? OR (started_at = ? AND id < ?))"), append(args, at, at, id)
	}
	args = append(args, f.Limit+1)
	rows, err := s.query(ctx, `SELECT id,workspace_id,provider,provider_event_id,provider_url,repository,branch,commit_sha,environment,actor,status,original_status,failure_category,failure_summary,started_at,finished_at,duration,created_at FROM deployments WHERE `+strings.Join(clauses, " AND ")+` ORDER BY started_at DESC,id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]Deployment, 0)
	for rows.Next() {
		d, err := scanDeployment(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > f.Limit {
		last := items[f.Limit-1]
		next, items = encodeCursor(last.StartedAt, last.ID), items[:f.Limit]
	}
	return items, next, nil
}

type scanner interface{ Scan(...any) error }

func scanDeployment(row scanner) (Deployment, error) {
	var d Deployment
	var started, finished, created string
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.Provider, &d.ProviderEventID, &d.ProviderURL, &d.Repository, &d.Branch, &d.CommitSHA, &d.Environment, &d.Actor, &d.Status, &d.OriginalStatus, &d.FailureCategory, &d.FailureSummary, &started, &finished, &d.Duration, &created)
	if err != nil {
		return d, err
	}
	d.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	if finished != "" {
		d.FinishedAt, _ = time.Parse(time.RFC3339Nano, finished)
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return d, nil
}

func (s *Store) Deployment(ctx context.Context, workspaceID, id string) (DeploymentDetail, error) {
	var detail DeploymentDetail
	d, err := scanDeployment(s.queryRow(ctx, `SELECT id,workspace_id,provider,provider_event_id,provider_url,repository,branch,commit_sha,environment,actor,status,original_status,failure_category,failure_summary,started_at,finished_at,duration,created_at FROM deployments WHERE workspace_id=? AND id=?`, workspaceID, id))
	if err != nil {
		return detail, err
	}
	detail.Deployment = d
	checks, err := s.query(ctx, `SELECT id,deployment_id,name,status,summary,started_at,finished_at FROM deployment_checks WHERE deployment_id=? ORDER BY started_at,id`, id)
	if err != nil {
		return detail, err
	}
	defer checks.Close()
	for checks.Next() {
		var c DeploymentCheck
		if err := checks.Scan(&c.ID, &c.DeploymentID, &c.Name, &c.Status, &c.Summary, &c.StartedAt, &c.FinishedAt); err != nil {
			return detail, err
		}
		detail.Checks = append(detail.Checks, c)
	}
	if err := checks.Err(); err != nil {
		return detail, err
	}
	logs, err := s.query(ctx, `SELECT content FROM log_excerpts WHERE deployment_id=? ORDER BY line_number DESC LIMIT 200`, id)
	if err != nil {
		return detail, err
	}
	defer logs.Close()
	for logs.Next() {
		var line string
		if err := logs.Scan(&line); err != nil {
			return detail, err
		}
		detail.LogLines = append(detail.LogLines, line)
	}
	for left, right := 0, len(detail.LogLines)-1; left < right; left, right = left+1, right-1 {
		detail.LogLines[left], detail.LogLines[right] = detail.LogLines[right], detail.LogLines[left]
	}
	return detail, logs.Err()
}

func (s *Store) AddDeadLetter(ctx context.Context, event WebhookEvent, processErr error, retryCount int) (DeadLetterEvent, error) {
	d := DeadLetterEvent{ID: uuid.NewString(), WebhookEventID: event.ID, Provider: event.Provider, Error: processErr.Error(), RetryCount: retryCount, RawPayloadRef: event.ID, CreatedAt: time.Now().UTC()}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return d, err
	}
	defer tx.Rollback()
	_, err = s.txExec(ctx, tx, `INSERT INTO dead_letter_events(id,webhook_event_id,provider,error,retry_count,raw_payload_ref,created_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(webhook_event_id) DO UPDATE SET error=excluded.error,retry_count=excluded.retry_count,created_at=excluded.created_at`, d.ID, d.WebhookEventID, d.Provider, d.Error, d.RetryCount, d.RawPayloadRef, d.CreatedAt.Format(time.RFC3339Nano))
	if err == nil {
		_, err = s.txExec(ctx, tx, `UPDATE webhook_events SET status='dead_letter', processed_at=? WHERE id=?`, now(), event.ID)
	}
	if err != nil {
		return d, err
	}
	return d, tx.Commit()
}

func (s *Store) ListDeadLetters(ctx context.Context, workspaceID string) ([]DeadLetterEvent, error) {
	rows, err := s.query(ctx, `SELECT d.id,d.webhook_event_id,d.provider,d.error,d.retry_count,d.raw_payload_ref,d.created_at FROM dead_letter_events d JOIN webhook_events w ON w.id=d.webhook_event_id WHERE w.workspace_id=? ORDER BY d.created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DeadLetterEvent, 0)
	for rows.Next() {
		var d DeadLetterEvent
		var created string
		if err := rows.Scan(&d.ID, &d.WebhookEventID, &d.Provider, &d.Error, &d.RetryCount, &d.RawPayloadRef, &created); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		items = append(items, d)
	}
	return items, rows.Err()
}

func (s *Store) DeadLetterCount(ctx context.Context, workspaceID string) (int, error) {
	var count int
	err := s.queryRow(ctx, `SELECT COUNT(*) FROM dead_letter_events d JOIN webhook_events w ON w.id=d.webhook_event_id WHERE w.workspace_id=?`, workspaceID).Scan(&count)
	return count, err
}

func (s *Store) ConnectedProviders(ctx context.Context, workspaceID string) ([]string, error) {
	rows, err := s.query(ctx, `SELECT provider FROM provider_connections WHERE workspace_id=? ORDER BY provider`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	providers := make([]string, 0)
	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

// PrepareReprocess leaves the DLQ record intact until the event was published successfully.
func (s *Store) PrepareReprocess(ctx context.Context, workspaceID, id string) (string, error) {
	var eventID string
	err := s.queryRow(ctx, `SELECT d.webhook_event_id FROM dead_letter_events d JOIN webhook_events w ON w.id=d.webhook_event_id WHERE d.id=? AND w.workspace_id=?`, id, workspaceID).Scan(&eventID)
	if err != nil {
		return "", err
	}
	if _, err := s.exec(ctx, `UPDATE webhook_events SET status='queued', processed_at=NULL WHERE id=?`, eventID); err != nil {
		return "", err
	}
	return eventID, nil
}

func (s *Store) CompleteReprocess(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM dead_letter_events WHERE id=?`, id)
	return err
}

func (s *Store) ListRules(ctx context.Context, workspaceID string) ([]NotificationRule, error) {
	rows, err := s.query(ctx, `SELECT id,workspace_id,repository,environment,status,channel,target FROM notification_rules WHERE workspace_id=? ORDER BY repository,environment`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rules := make([]NotificationRule, 0)
	for rows.Next() {
		var r NotificationRule
		if err := rows.Scan(&r.ID, &r.WorkspaceID, &r.Repository, &r.Environment, &r.Status, &r.Channel, &r.Target); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *Store) CreateRule(ctx context.Context, rule NotificationRule) (NotificationRule, error) {
	rule.ID = uuid.NewString()
	_, err := s.exec(ctx, `INSERT INTO notification_rules(id,workspace_id,repository,environment,status,channel,target,created_at) VALUES(?,?,?,?,?,?,?,?)`, rule.ID, rule.WorkspaceID, rule.Repository, rule.Environment, rule.Status, rule.Channel, rule.Target, now())
	return rule, err
}

func (s *Store) DeliverNotifications(ctx context.Context, d Deployment) error {
	rules, err := s.ListRules(ctx, d.WorkspaceID)
	if err != nil {
		return err
	}
	kind, shouldDeliver := "failure", d.Status == StatusFailed
	if d.Status == StatusSuccess {
		var count int
		err = s.queryRow(ctx, `SELECT COUNT(*) FROM deployments WHERE workspace_id=? AND repository=? AND environment=? AND status='failed' AND started_at<?`, d.WorkspaceID, d.Repository, d.Environment, d.StartedAt.UTC().Format(time.RFC3339Nano)).Scan(&count)
		if err != nil {
			return err
		}
		shouldDeliver, kind = count > 0, "recovery"
	}
	if !shouldDeliver {
		return nil
	}
	for _, rule := range rules {
		if rule.Status != StatusFailed || (rule.Repository != "*" && rule.Repository != d.Repository) || (rule.Environment != "*" && rule.Environment != d.Environment) {
			continue
		}
		if _, err = s.exec(ctx, `INSERT INTO notification_deliveries(id,deployment_id,rule_id,kind,status,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(deployment_id,rule_id,kind) DO NOTHING`, uuid.NewString(), d.ID, rule.ID, kind, "delivered", now()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureDemoRule(ctx context.Context, workspaceID string) error {
	var count int
	if err := s.queryRow(ctx, `SELECT COUNT(*) FROM notification_rules WHERE workspace_id=?`, workspaceID).Scan(&count); err != nil || count > 0 {
		return err
	}
	_, err := s.CreateRule(ctx, NotificationRule{WorkspaceID: workspaceID, Repository: "*", Environment: "*", Status: StatusFailed, Channel: "slack", Target: "#release-watch"})
	return err
}

func (s *Store) ListConnections(ctx context.Context, workspaceID string) ([]ProviderConnection, error) {
	rows, err := s.query(ctx, `SELECT id,provider,name,created_at FROM provider_connections WHERE workspace_id=? ORDER BY provider`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ProviderConnection, 0)
	for rows.Next() {
		var item ProviderConnection
		var created string
		if err := rows.Scan(&item.ID, &item.Provider, &item.Name, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SaveConnection(ctx context.Context, workspaceID, provider, name, encryptedSecret string) (ProviderConnection, error) {
	item := ProviderConnection{ID: uuid.NewString(), Provider: provider, Name: name, CreatedAt: time.Now().UTC()}
	_, err := s.exec(ctx, `INSERT INTO provider_connections(id,workspace_id,provider,name,secret_encrypted,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(workspace_id,provider) DO UPDATE SET name=excluded.name,secret_encrypted=excluded.secret_encrypted`, item.ID, workspaceID, provider, name, encryptedSecret, item.CreatedAt.Format(time.RFC3339Nano))
	return item, err
}

func encodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}
func decodeCursor(cursor string) (string, string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func NewDeploymentID() string {
	return fmt.Sprintf("dpl_%s", strings.ReplaceAll(uuid.NewString(), "-", "")[:12])
}
