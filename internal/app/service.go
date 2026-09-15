package app

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var supportedProviders = map[string]bool{
	"github": true, "gitlab": true, "circleci": true, "vercel": true, "netlify": true, "aws-codepipeline": true,
}

type Config struct {
	WebhookSecrets map[string]string
	SecretNames    []string
}

type Service struct {
	store   *Store
	config  Config
	jobs    chan string
	stop    chan struct{}
	stopped sync.Once
}

func NewService(store *Store, config Config) *Service {
	return &Service{store: store, config: config, jobs: make(chan string, 512), stop: make(chan struct{})}
}

func (s *Service) Start(ctx context.Context) { go s.worker(ctx); go s.retention(ctx) }
func (s *Service) Close()                    { s.stopped.Do(func() { close(s.stop) }) }

func (s *Service) Verify(provider string, header http.Header, body []byte) error {
	secret := s.config.WebhookSecrets[provider]
	if secret == "" {
		return fmt.Errorf("provider %q is not configured", provider)
	}
	if provider == "gitlab" {
		if !hmac.Equal([]byte(header.Get("X-Gitlab-Token")), []byte(secret)) {
			return errors.New("invalid signature")
		}
		return nil
	}
	signature := header.Get("X-Hub-Signature-256")
	if signature == "" {
		signature = header.Get("X-DeployPulse-Signature")
	}
	if signature == "" {
		signature = header.Get("X-Vercel-Signature")
	}
	if signature == "" {
		signature = header.Get("X-Webhook-Signature")
	}
	if signature == "" {
		signature = header.Get("Circleci-Signature")
	}
	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return errors.New("missing signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(want)) {
		return errors.New("invalid signature")
	}
	return nil
}

func (s *Service) Enqueue(eventID string) bool {
	select {
	case s.jobs <- eventID:
		return true
	default:
		return false
	}
}

func (s *Service) EncryptSecret(secret string) (string, error) {
	key := sha256.Sum256([]byte(first(s.config.WebhookSecrets["encryption"], "deploy-pulse-development-key")))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return hex.EncodeToString(append(nonce, gcm.Seal(nil, nonce, []byte(secret), nil)...)), nil
}

func (s *Service) worker(ctx context.Context) {
	for {
		select {
		case <-s.stop:
			return
		case eventID := <-s.jobs:
			s.process(ctx, eventID)
		}
	}
}

func (s *Service) retention(ctx context.Context) {
	_ = s.store.PruneRetained(ctx)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			if err := s.store.PruneRetained(ctx); err != nil {
				log.Printf("retention prune failed: %v", err)
			}
		}
	}
}

func (s *Service) process(ctx context.Context, eventID string) {
	event, err := s.store.Webhook(ctx, eventID)
	if err != nil {
		return
	}
	deployment, checks, logs, err := s.parse(event)
	if err != nil {
		log.Printf("correlation_id=%s provider=%s event=%s parse_error=%q", event.CorrelationID, event.Provider, event.ProviderEventID, err)
		_, _ = s.store.AddDeadLetter(ctx, event, err)
		return
	}
	created, err := s.store.SaveDeployment(ctx, deployment, checks, logs)
	if err != nil {
		log.Printf("correlation_id=%s provider=%s event=%s storage_error=%q", event.CorrelationID, event.Provider, event.ProviderEventID, err)
		_, _ = s.store.AddDeadLetter(ctx, event, err)
		return
	}
	if created {
		_ = s.store.DeliverNotifications(ctx, deployment)
	}
	log.Printf("correlation_id=%s provider=%s event=%s deployment=%s status=%s", event.CorrelationID, event.Provider, event.ProviderEventID, deployment.ID, deployment.Status)
	_ = s.store.MarkWebhook(ctx, event.ID, "processed")
}

func ProviderEventID(provider string, payload []byte) string {
	var root map[string]any
	if json.Unmarshal(payload, &root) != nil {
		return ""
	}
	paths := [][]string{{"provider_event_id"}, {"id"}}
	switch provider {
	case "github":
		paths = append([][]string{{"workflow_run", "id"}}, paths...)
	case "gitlab":
		paths = append([][]string{{"object_attributes", "id"}}, paths...)
	case "circleci":
		paths = append([][]string{{"pipeline", "id"}, {"id"}}, paths...)
	case "vercel", "netlify":
		paths = append([][]string{{"deployment", "id"}, {"id"}}, paths...)
	case "aws-codepipeline":
		paths = append([][]string{{"detail", "execution-id"}}, paths...)
	}
	for _, path := range paths {
		if value := stringAt(root, path...); value != "" {
			return value
		}
	}
	return ""
}

func (s *Service) parse(event WebhookEvent) (Deployment, []DeploymentCheck, []string, error) {
	var root map[string]any
	if err := json.Unmarshal(event.Payload, &root); err != nil {
		return Deployment{}, nil, nil, fmt.Errorf("decode %s payload: %w", event.Provider, err)
	}
	if !supportedProviders[event.Provider] {
		return Deployment{}, nil, nil, fmt.Errorf("unsupported provider %q", event.Provider)
	}
	base := objectAt(root, "deployment")
	if len(base) == 0 {
		base = root
	}
	providerData := providerFields(event.Provider, root, base)
	original := first(providerData.status, stringAt(base, "status"), stringAt(base, "state"), stringAt(root, "status"), stringAt(root, "state"))
	if original == "" {
		return Deployment{}, nil, nil, errors.New("event has no provider status")
	}
	started := parseTime(first(providerData.startedAt, stringAt(base, "started_at"), stringAt(base, "created_at"), stringAt(root, "created_at")))
	if started.IsZero() {
		started = event.ReceivedAt
	}
	finished := parseTime(first(providerData.finishedAt, stringAt(base, "finished_at"), stringAt(base, "updated_at"), stringAt(root, "updated_at")))
	status := NormalizeState(original)
	if status != StatusSuccess && status != StatusFailed && status != StatusCancelled {
		finished = time.Time{}
	}
	logs := stringSlice(firstAny(base["log_lines"], root["log_lines"]))
	summary := first(providerData.summary, stringAt(base, "failure_summary"), stringAt(root, "failure_summary"))
	if summary == "" && status == StatusFailed {
		summary = "The provider reported a failed deployment. Inspect the sanitized log excerpt."
	}
	category := FailureCategory(status, summary, logs)
	d := Deployment{ID: NewDeploymentID(), WorkspaceID: event.WorkspaceID, Provider: event.Provider, ProviderEventID: event.ProviderEventID, ProviderURL: first(providerData.url, stringAt(base, "provider_url"), stringAt(base, "url")), Repository: first(providerData.repository, stringAt(base, "repository"), stringAt(root, "repository"), "unknown/repository"), Branch: first(providerData.branch, stringAt(base, "branch"), stringAt(root, "branch"), "unknown"), CommitSHA: first(providerData.sha, stringAt(base, "commit_sha"), stringAt(base, "sha"), stringAt(root, "sha"), "unknown"), Environment: first(providerData.environment, stringAt(base, "environment"), stringAt(root, "environment"), "production"), Actor: first(providerData.actor, stringAt(base, "actor"), stringAt(root, "actor"), "unknown"), Status: status, OriginalStatus: original, FailureCategory: category, FailureSummary: s.redact(summary), StartedAt: started, FinishedAt: finished, Duration: duration(started, finished), CreatedAt: time.Now().UTC()}
	checks := parseChecks(firstAny(base["checks"], root["checks"]), d.ID, status, started, finished, s)
	if len(checks) == 0 {
		checks = []DeploymentCheck{{ID: uuid.NewString(), DeploymentID: d.ID, Name: "Provider deployment", Status: status, Summary: s.redact(summary), StartedAt: started.Format(time.RFC3339), FinishedAt: formatTime(finished)}}
	}
	return d, checks, s.redactLines(logs), nil
}

type providerPayload struct{ repository, branch, sha, environment, actor, status, startedAt, finishedAt, url, summary string }

func providerFields(provider string, root, base map[string]any) providerPayload {
	result := providerPayload{}
	switch provider {
	case "github":
		workflow := objectAt(root, "workflow_run")
		repo := objectAt(root, "repository")
		actor := objectAt(workflow, "actor")
		result = providerPayload{repository: stringAt(repo, "full_name"), branch: stringAt(workflow, "head_branch"), sha: stringAt(workflow, "head_sha"), environment: stringAt(workflow, "environment"), actor: first(stringAt(actor, "login"), stringAt(actor, "name")), status: first(stringAt(workflow, "conclusion"), stringAt(workflow, "status")), startedAt: stringAt(workflow, "created_at"), finishedAt: stringAt(workflow, "updated_at"), url: stringAt(workflow, "html_url")}
	case "gitlab":
		attr := objectAt(root, "object_attributes")
		project := objectAt(root, "project")
		user := objectAt(root, "user")
		result = providerPayload{repository: stringAt(project, "path_with_namespace"), branch: stringAt(attr, "ref"), sha: stringAt(attr, "sha"), environment: stringAt(attr, "environment"), actor: first(stringAt(user, "username"), stringAt(user, "name")), status: stringAt(attr, "status"), startedAt: stringAt(attr, "created_at"), finishedAt: stringAt(attr, "updated_at"), url: stringAt(attr, "web_url")}
	case "circleci":
		pipeline := objectAt(root, "pipeline")
		vcs := objectAt(pipeline, "vcs")
		trigger := objectAt(pipeline, "trigger")
		triggerActor := objectAt(trigger, "actor")
		result = providerPayload{repository: first(stringAt(vcs, "repository"), stringAt(pipeline, "repository")), branch: stringAt(vcs, "branch"), sha: stringAt(vcs, "revision"), environment: stringAt(pipeline, "environment"), actor: first(stringAt(triggerActor, "login"), stringAt(trigger, "actor")), status: first(stringAt(pipeline, "status"), stringAt(root, "status")), startedAt: first(stringAt(pipeline, "created_at"), stringAt(root, "created_at")), finishedAt: first(stringAt(pipeline, "updated_at"), stringAt(root, "updated_at")), url: first(stringAt(pipeline, "web_url"), stringAt(root, "web_url"))}
	case "vercel":
		meta := objectAt(base, "meta")
		result = providerPayload{repository: stringAt(meta, "githubCommitRepo"), branch: stringAt(meta, "githubCommitRef"), sha: stringAt(meta, "githubCommitSha"), environment: first(stringAt(base, "target"), stringAt(base, "environment")), actor: stringAt(meta, "githubCommitAuthorName"), status: first(stringAt(base, "state"), stringAt(base, "readyState")), startedAt: first(stringAt(base, "createdAt"), stringAt(base, "created_at")), finishedAt: first(stringAt(base, "readyAt"), stringAt(base, "ready_at")), url: first(stringAt(base, "url"), stringAt(base, "inspectorUrl"))}
	case "netlify":
		commit := objectAt(base, "commit")
		user := objectAt(base, "user")
		result = providerPayload{repository: first(stringAt(base, "repository"), stringAt(root, "repository")), branch: stringAt(base, "branch"), sha: first(stringAt(base, "commit_ref"), stringAt(commit, "sha")), environment: first(stringAt(base, "context"), stringAt(base, "environment")), actor: first(stringAt(user, "full_name"), stringAt(base, "committer")), status: first(stringAt(base, "state"), stringAt(base, "status")), startedAt: first(stringAt(base, "created_at"), stringAt(base, "createdAt")), finishedAt: first(stringAt(base, "updated_at"), stringAt(base, "published_at")), url: first(stringAt(base, "ssl_url"), stringAt(base, "url"))}
	case "aws-codepipeline":
		detail := objectAt(root, "detail")
		result = providerPayload{repository: first(stringAt(detail, "repository"), stringAt(detail, "pipeline")), branch: stringAt(detail, "branch"), sha: first(stringAt(detail, "commit"), stringAt(detail, "version")), environment: first(stringAt(detail, "environment"), "production"), actor: first(stringAt(detail, "actor"), "aws"), status: stringAt(detail, "state"), startedAt: first(stringAt(detail, "start-time"), stringAt(detail, "startTime")), finishedAt: first(stringAt(detail, "stop-time"), stringAt(detail, "stopTime")), url: stringAt(detail, "url")}
	}
	return result
}

func NormalizeState(value string) Status {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "queued", "pending", "waiting", "created", "not_started", "requested":
		return StatusQueued
	case "running", "in_progress", "in progress", "building", "active":
		return StatusRunning
	case "success", "succeeded", "completed", "ready", "passed", "pass":
		return StatusSuccess
	case "failed", "failure", "error", "errored", "timed_out", "timed out":
		return StatusFailed
	case "cancelled", "canceled", "skipped", "aborted":
		return StatusCancelled
	default:
		return StatusUnknown
	}
}

func FailureCategory(status Status, summary string, lines []string) string {
	if status == StatusCancelled {
		return "cancelled"
	}
	if status != StatusFailed {
		return ""
	}
	text := strings.ToLower(summary + " " + strings.Join(lines, " "))
	switch {
	case strings.Contains(text, "test"):
		return "test_failure"
	case strings.Contains(text, "build") || strings.Contains(text, "compile"):
		return "build_failure"
	case strings.Contains(text, "deploy") || strings.Contains(text, "release"):
		return "deploy_failure"
	default:
		return "unknown"
	}
}

var secretPattern = regexp.MustCompile(`(?i)((?:token|password|secret|api[_-]?key)[\w.-]*\s*[=:]\s*)([^\s,;]+)`)
var privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)

func (s *Service) redact(value string) string {
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED PRIVATE KEY]")
	value = secretPattern.ReplaceAllString(value, "${1}[REDACTED]")
	for _, name := range s.config.SecretNames {
		if name == "" {
			continue
		}
		named := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(name) + `\s*[=:]\s*)([^\s,;]+)`)
		value = named.ReplaceAllString(value, "${1}[REDACTED]")
	}
	return value
}
func (s *Service) redactLines(lines []string) []string {
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, s.redact(line))
	}
	return result
}

func parseChecks(value any, deploymentID string, fallback Status, started, finished time.Time, service *Service) []DeploymentCheck {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	checks := make([]DeploymentCheck, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := first(stringAt(m, "name"), "Provider check")
		status := NormalizeState(first(stringAt(m, "status"), string(fallback)))
		checks = append(checks, DeploymentCheck{ID: uuid.NewString(), DeploymentID: deploymentID, Name: name, Status: status, Summary: service.redact(stringAt(m, "summary")), StartedAt: first(stringAt(m, "started_at"), started.Format(time.RFC3339)), FinishedAt: first(stringAt(m, "finished_at"), formatTime(finished))})
	}
	return checks
}
func objectAt(root map[string]any, path ...string) map[string]any {
	current := root
	for _, key := range path {
		value, ok := current[key]
		if !ok {
			return nil
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}
func stringAt(root map[string]any, path ...string) string {
	if root == nil {
		return ""
	}
	current := any(root)
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	switch value := current.(type) {
	case string:
		return value
	case float64:
		return fmt.Sprintf("%.0f", value)
	case json.Number:
		return value.String()
	default:
		return ""
	}
}
func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
func firstAny(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
func stringSlice(value any) []string {
	switch v := value.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	case []string:
		return v
	case string:
		return strings.Split(v, "\n")
	default:
		return nil
	}
}
func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z07:00"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
func duration(start, finish time.Time) string {
	if start.IsZero() || finish.IsZero() {
		return ""
	}
	return finish.Sub(start).Round(time.Second).String()
}

func SortedProviders() []string {
	providers := make([]string, 0, len(supportedProviders))
	for provider := range supportedProviders {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func (s *Service) SeedDemo(ctx context.Context, workspaceID string) error {
	if err := s.store.EnsureDemoRule(ctx, workspaceID); err != nil {
		return err
	}
	now := time.Now().UTC()
	items := []struct {
		provider, event, repo, branch, sha, env, actor string
		status                                         Status
		summary, category                              string
		ago                                            time.Duration
		logs                                           []string
	}{
		{"vercel", "demo-vercel-failed", "northstar/web", "release/3.18", "d9f314b", "staging", "Mika Santos", StatusFailed, "Payment suite failed in 3 tests.", "test_failure", 18 * time.Minute, []string{"running pnpm test:payments", "FAIL  payment/checkout.spec.ts", "token=[REDACTED]", "3 tests failed"}},
		{"github", "demo-github-failed", "checkout/api", "main", "a821c9e", "production", "Ari N.", StatusFailed, "Container image build exited with code 1.", "build_failure", 42 * time.Minute, []string{"building checkout/api", "Error: image build exited with code 1", "password=[REDACTED]"}},
		{"github", "demo-github-success", "checkout/api", "main", "a821c71", "production", "Ari N.", StatusSuccess, "", "", 2 * time.Hour, []string{"build complete", "deployment healthy"}},
		{"gitlab", "demo-gitlab-running", "ledger/worker", "feature/settlement", "5b89dd4", "staging", "Naya F.", StatusRunning, "", "", 5 * time.Minute, []string{"running migration check"}},
		{"netlify", "demo-netlify-success", "marketing/site", "main", "8ca12e0", "production", "Jules P.", StatusSuccess, "", "", 6 * time.Hour, []string{"published successfully"}},
	}
	for _, item := range items {
		started := now.Add(-item.ago)
		finished := started.Add(2*time.Minute + 18*time.Second)
		if item.status == StatusRunning {
			finished = time.Time{}
		}
		d := Deployment{ID: NewDeploymentID(), WorkspaceID: workspaceID, Provider: item.provider, ProviderEventID: item.event, ProviderURL: "https://example.com/deployments/" + item.event, Repository: item.repo, Branch: item.branch, CommitSHA: item.sha, Environment: item.env, Actor: item.actor, Status: item.status, OriginalStatus: string(item.status), FailureCategory: item.category, FailureSummary: item.summary, StartedAt: started, FinishedAt: finished, Duration: duration(started, finished), CreatedAt: now}
		checks := []DeploymentCheck{{ID: uuid.NewString(), DeploymentID: d.ID, Name: "Build and deploy", Status: item.status, Summary: item.summary, StartedAt: started.Format(time.RFC3339), FinishedAt: formatTime(finished)}}
		created, err := s.store.SaveDeployment(ctx, d, checks, s.redactLines(item.logs))
		if err != nil {
			return err
		}
		if created {
			_ = s.store.DeliverNotifications(ctx, d)
		}
	}
	return nil
}
