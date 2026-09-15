package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

func OpenStore(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) PruneRetained(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM log_excerpts WHERE created_at < ?`, time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhook_events WHERE received_at < ?`, time.Now().UTC().AddDate(0, 0, -90).Format(time.RFC3339Nano))
	return err
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
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
  payload BLOB NOT NULL, payload_hash TEXT NOT NULL, correlation_id TEXT NOT NULL, status TEXT NOT NULL,
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
);`)
	return err
}

func (s *Store) RecordWebhook(ctx context.Context, event WebhookEvent) (bool, error) {
	result, err := s.db.ExecContext(ctx, `INSERT INTO webhook_events
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
	err := s.db.QueryRowContext(ctx, `SELECT id,workspace_id,provider,provider_event_id,payload,payload_hash,correlation_id,status,received_at FROM webhook_events WHERE id=?`, id).
		Scan(&event.ID, &event.WorkspaceID, &event.Provider, &event.ProviderEventID, &event.Payload, &event.PayloadHash, &event.CorrelationID, &event.Status, &received)
	if err != nil {
		return event, err
	}
	event.ReceivedAt, _ = time.Parse(time.RFC3339Nano, received)
	return event, nil
}

func (s *Store) MarkWebhook(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE webhook_events SET status=?, processed_at=? WHERE id=?`, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *Store) SaveDeployment(ctx context.Context, deployment Deployment, checks []DeploymentCheck, logs []string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO deployments
  (id,workspace_id,provider,provider_event_id,provider_url,repository,branch,commit_sha,environment,actor,status,original_status,failure_category,failure_summary,started_at,finished_at,duration,created_at)
  VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id,provider,provider_event_id) DO NOTHING`,
		deployment.ID, deployment.WorkspaceID, deployment.Provider, deployment.ProviderEventID, deployment.ProviderURL,
		deployment.Repository, deployment.Branch, deployment.CommitSHA, deployment.Environment, deployment.Actor,
		deployment.Status, deployment.OriginalStatus, deployment.FailureCategory, deployment.FailureSummary,
		deployment.StartedAt.UTC().Format(time.RFC3339Nano), deployment.FinishedAt.UTC().Format(time.RFC3339Nano), deployment.Duration, deployment.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 0 {
		return false, err
	}
	for _, check := range checks {
		if _, err := tx.ExecContext(ctx, `INSERT INTO deployment_checks(id,deployment_id,name,status,summary,started_at,finished_at) VALUES(?,?,?,?,?,?,?)`,
			check.ID, deployment.ID, check.Name, check.Status, check.Summary, check.StartedAt, check.FinishedAt); err != nil {
			return false, err
		}
	}
	for i, line := range logs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO log_excerpts(id,deployment_id,line_number,content,created_at) VALUES(?,?,?,?,?)`, uuid.NewString(), deployment.ID, i+1, line, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return false, err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO repositories(id,workspace_id,name,created_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,name) DO NOTHING`, uuid.NewString(), deployment.WorkspaceID, deployment.Repository, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) ListDeployments(ctx context.Context, workspaceID string, f ListFilter) ([]Deployment, string, error) {
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	clauses, args := []string{"workspace_id=?"}, []any{workspaceID}
	for column, value := range map[string]string{"repository": f.Repository, "branch": f.Branch, "environment": f.Environment, "provider": f.Provider, "status": f.Status, "actor": f.Actor} {
		if value != "" {
			clauses = append(clauses, column+"=?")
			args = append(args, value)
		}
	}
	if !f.Start.IsZero() {
		clauses = append(clauses, "started_at>=?")
		args = append(args, f.Start.UTC().Format(time.RFC3339Nano))
	}
	if !f.End.IsZero() {
		clauses = append(clauses, "started_at<=?")
		args = append(args, f.End.UTC().Format(time.RFC3339Nano))
	}
	if f.Cursor != "" {
		if at, id, ok := decodeCursor(f.Cursor); ok {
			clauses = append(clauses, "(started_at < ? OR (started_at = ? AND id < ?))")
			args = append(args, at, at, id)
		}
	}
	args = append(args, f.Limit+1)
	rows, err := s.db.QueryContext(ctx, `SELECT id,workspace_id,provider,provider_event_id,provider_url,repository,branch,commit_sha,environment,actor,status,original_status,failure_category,failure_summary,started_at,finished_at,duration,created_at FROM deployments WHERE `+strings.Join(clauses, " AND ")+` ORDER BY started_at DESC,id DESC LIMIT ?`, args...)
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
		next = encodeCursor(last.StartedAt, last.ID)
		items = items[:f.Limit]
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
	row := s.db.QueryRowContext(ctx, `SELECT id,workspace_id,provider,provider_event_id,provider_url,repository,branch,commit_sha,environment,actor,status,original_status,failure_category,failure_summary,started_at,finished_at,duration,created_at FROM deployments WHERE workspace_id=? AND id=?`, workspaceID, id)
	d, err := scanDeployment(row)
	if err != nil {
		return detail, err
	}
	detail.Deployment = d
	checks, err := s.db.QueryContext(ctx, `SELECT id,deployment_id,name,status,summary,started_at,finished_at FROM deployment_checks WHERE deployment_id=? ORDER BY started_at,id`, id)
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
	logs, err := s.db.QueryContext(ctx, `SELECT content FROM log_excerpts WHERE deployment_id=? ORDER BY line_number DESC LIMIT 200`, id)
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
	return detail, checks.Err()
}

func (s *Store) AddDeadLetter(ctx context.Context, event WebhookEvent, parseErr error) (DeadLetterEvent, error) {
	d := DeadLetterEvent{ID: uuid.NewString(), WebhookEventID: event.ID, Provider: event.Provider, Error: parseErr.Error(), RetryCount: 0, RawPayloadRef: event.ID, CreatedAt: time.Now().UTC()}
	_, err := s.db.ExecContext(ctx, `INSERT INTO dead_letter_events(id,webhook_event_id,provider,error,retry_count,raw_payload_ref,created_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(webhook_event_id) DO UPDATE SET error=excluded.error,retry_count=dead_letter_events.retry_count+1`, d.ID, d.WebhookEventID, d.Provider, d.Error, d.RetryCount, d.RawPayloadRef, d.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return d, err
	}
	return d, s.MarkWebhook(ctx, event.ID, "dead_letter")
}

func (s *Store) ListDeadLetters(ctx context.Context, workspaceID string) ([]DeadLetterEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT d.id,d.webhook_event_id,d.provider,d.error,d.retry_count,d.raw_payload_ref,d.created_at FROM dead_letter_events d JOIN webhook_events w ON w.id=d.webhook_event_id WHERE w.workspace_id=? ORDER BY d.created_at DESC`, workspaceID)
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

func (s *Store) ReprocessDeadLetter(ctx context.Context, workspaceID, id string) (string, error) {
	var eventID string
	err := s.db.QueryRowContext(ctx, `SELECT d.webhook_event_id FROM dead_letter_events d JOIN webhook_events w ON w.id=d.webhook_event_id WHERE d.id=? AND w.workspace_id=?`, id, workspaceID).Scan(&eventID)
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM dead_letter_events WHERE id=?`, id)
	if err != nil {
		return "", err
	}
	return eventID, s.MarkWebhook(ctx, eventID, "queued")
}

func (s *Store) ListRules(ctx context.Context, workspaceID string) ([]NotificationRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,workspace_id,repository,environment,status,channel,target FROM notification_rules WHERE workspace_id=? ORDER BY repository,environment`, workspaceID)
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO notification_rules(id,workspace_id,repository,environment,status,channel,target,created_at) VALUES(?,?,?,?,?,?,?,?)`, rule.ID, rule.WorkspaceID, rule.Repository, rule.Environment, rule.Status, rule.Channel, rule.Target, time.Now().UTC().Format(time.RFC3339Nano))
	return rule, err
}

func (s *Store) DeliverNotifications(ctx context.Context, d Deployment) error {
	rules, err := s.ListRules(ctx, d.WorkspaceID)
	if err != nil {
		return err
	}
	kind := "failure"
	shouldDeliver := d.Status == StatusFailed
	if d.Status == StatusSuccess {
		var count int
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deployments WHERE workspace_id=? AND repository=? AND environment=? AND status='failed' AND started_at<?`, d.WorkspaceID, d.Repository, d.Environment, d.StartedAt.UTC().Format(time.RFC3339Nano)).Scan(&count)
		if err != nil {
			return err
		}
		shouldDeliver = count > 0
		kind = "recovery"
	}
	if !shouldDeliver {
		return nil
	}
	for _, rule := range rules {
		if rule.Status != StatusFailed || (rule.Repository != "*" && rule.Repository != d.Repository) || (rule.Environment != "*" && rule.Environment != d.Environment) {
			continue
		}
		_, err = s.db.ExecContext(ctx, `INSERT INTO notification_deliveries(id,deployment_id,rule_id,kind,status,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(deployment_id,rule_id,kind) DO NOTHING`, uuid.NewString(), d.ID, rule.ID, kind, "delivered", time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureDemoRule(ctx context.Context, workspaceID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_rules WHERE workspace_id=?`, workspaceID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := s.CreateRule(ctx, NotificationRule{WorkspaceID: workspaceID, Repository: "*", Environment: "*", Status: StatusFailed, Channel: "slack", Target: "#release-watch"})
	return err
}

func (s *Store) ListConnections(ctx context.Context, workspaceID string) ([]ProviderConnection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,provider,name,created_at FROM provider_connections WHERE workspace_id=? ORDER BY provider`, workspaceID)
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO provider_connections(id,workspace_id,provider,name,secret_encrypted,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(workspace_id,provider) DO UPDATE SET name=excluded.name,secret_encrypted=excluded.secret_encrypted`, item.ID, workspaceID, provider, name, encryptedSecret, item.CreatedAt.Format(time.RFC3339Nano))
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
