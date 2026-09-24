package app

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var supportedProviders = map[string]bool{
	"github": true, "gitlab": true, "circleci": true, "vercel": true, "netlify": true, "aws-codepipeline": true,
}

type Config struct {
	WebhookSecrets map[string]string
	SecretNames    []string
	EncryptionKey  string
	Production     bool
}

type Service struct {
	store  *Store
	config Config
}

func NewService(store *Store, config Config) *Service {
	return &Service{store: store, config: config}
}

func (s *Service) ValidateConfig() error {
	if s.config.Production && s.config.EncryptionKey == "" {
		return errors.New("ENCRYPTION_KEY is required in production")
	}
	return nil
}

func (s *Service) ConfiguredProviders() []string {
	providers := make([]string, 0, len(s.config.WebhookSecrets))
	for _, provider := range SortedProviders() {
		if s.config.WebhookSecrets[provider] != "" {
			providers = append(providers, provider)
		}
	}
	return providers
}

func (s *Service) Verify(provider string, header http.Header, body []byte) error {
	return s.VerifyWithSecret(provider, s.config.WebhookSecrets[provider], header, body)
}

func (s *Service) VerifyWithSecret(provider, secret string, header http.Header, body []byte) error {
	if secret == "" {
		return fmt.Errorf("provider %q is not configured", provider)
	}
	switch provider {
	case "github":
		return verifyHexHMAC(sha256.New, secret, body, header.Get("X-Hub-Signature-256"), "sha256=")
	case "gitlab":
		return verifyGitLab(secret, header, body)
	case "circleci":
		return verifyCircleCI(secret, header.Get("Circleci-Signature"), body)
	case "vercel":
		return verifyHexHMAC(sha1.New, secret, body, header.Get("X-Vercel-Signature"), "sha1=")
	case "netlify":
		return verifyNetlify(secret, header.Get("X-Webhook-Signature"), body)
	case "aws-codepipeline":
		return verifyHexHMAC(sha256.New, secret, body, header.Get("X-DeployPulse-Relay-Signature"), "sha256=")
	default:
		return errors.New("unsupported provider")
	}
}

func (s *Service) EncryptSecret(secret string) (string, error) {
	keyMaterial := s.config.EncryptionKey
	if keyMaterial == "" {
		if s.config.Production {
			return "", errors.New("ENCRYPTION_KEY is required in production")
		}
		keyMaterial = "deploy-pulse-development-key"
	}
	key := sha256.Sum256([]byte(keyMaterial))
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

func (s *Service) DecryptSecret(encrypted string) (string, error) {
	keyMaterial := s.config.EncryptionKey
	if keyMaterial == "" {
		if s.config.Production {
			return "", errors.New("ENCRYPTION_KEY is required in production")
		}
		keyMaterial = "deploy-pulse-development-key"
	}
	key := sha256.Sum256([]byte(keyMaterial))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(encrypted)
	if err != nil || len(data) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted provider secret")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	return string(plain), err
}

// Process only returns after the deployment and webhook state commit. The queue ACKs after this call.
func (s *Service) Process(ctx context.Context, eventID string) error {
	event, err := s.store.Webhook(ctx, eventID)
	if err != nil {
		return err
	}
	deployment, checks, logs, err := s.parse(event)
	if err != nil {
		return fmt.Errorf("parse %s event %s: %w", event.Provider, event.ProviderEventID, err)
	}
	created, err := s.store.SaveProcessedDeployment(ctx, event.ID, deployment, checks, logs)
	if err != nil {
		return err
	}
	if created {
		if err := s.store.DeliverNotifications(ctx, deployment); err != nil {
			log.Printf("correlation_id=%s notification_error=%q", event.CorrelationID, err)
		}
	}
	log.Printf("correlation_id=%s provider=%s event=%s deployment=%s status=%s", event.CorrelationID, event.Provider, event.ProviderEventID, deployment.ID, deployment.Status)
	return nil
}

func verifyHexHMAC(newHash func() hash.Hash, secret string, body []byte, signature, prefix string) error {
	signature = strings.TrimPrefix(signature, prefix)
	if signature == "" {
		return errors.New("missing signature")
	}
	mac := hmac.New(newHash, []byte(secret))
	_, _ = mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(want)) {
		return errors.New("invalid signature")
	}
	return nil
}

func verifyCircleCI(secret, signature string, body []byte) error {
	for _, part := range strings.Split(signature, ",") {
		version, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if version == "v1" && found {
			return verifyHexHMAC(sha256.New, secret, body, value, "")
		}
	}
	return errors.New("missing v1 signature")
}

func verifyGitLab(secret string, header http.Header, body []byte) error {
	if signature := header.Get("Webhook-Signature"); signature != "" {
		messageID, timestamp := header.Get("Webhook-Id"), header.Get("Webhook-Timestamp")
		key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
		if err != nil || messageID == "" || timestamp == "" {
			return errors.New("invalid GitLab signing token")
		}
		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write([]byte(messageID + "." + timestamp + "." + string(body)))
		want := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
		for _, candidate := range strings.Fields(signature) {
			if hmac.Equal([]byte(candidate), []byte(want)) {
				return nil
			}
		}
		return errors.New("invalid signature")
	}
	if hmac.Equal([]byte(header.Get("X-Gitlab-Token")), []byte(secret)) {
		return nil
	}
	return errors.New("invalid signature")
}

// Netlify signs its notification as an HS256 JWS with an issuer and payload hash claim.
func verifyNetlify(secret, token string, body []byte) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("missing JWS signature")
	}
	decode := func(value string, destination any) error {
		bytes, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			bytes, err = base64.URLEncoding.DecodeString(value)
			if err != nil {
				return err
			}
		}
		return json.Unmarshal(bytes, destination)
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	var claims struct {
		Issuer string `json:"iss"`
		Hash   string `json:"sha256"`
	}
	if err := decode(parts[0], &header); err != nil || decode(parts[1], &claims) != nil || header.Algorithm != "HS256" || claims.Issuer != "netlify" {
		return errors.New("invalid JWS payload")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		signature, err = base64.URLEncoding.DecodeString(parts[2])
	}
	if err != nil {
		return errors.New("invalid JWS signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	got := sha256.Sum256(body)
	if !hmac.Equal(signature, mac.Sum(nil)) || !hmac.Equal([]byte(claims.Hash), []byte(hex.EncodeToString(got[:]))) {
		return errors.New("invalid signature")
	}
	return nil
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
