package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProviderFixturesNormalize(t *testing.T) {
	t.Parallel()
	for provider, expected := range map[string]Status{"github": StatusFailed, "gitlab": StatusSuccess, "circleci": StatusRunning, "vercel": StatusSuccess, "netlify": StatusFailed, "aws-codepipeline": StatusSuccess} {
		provider, expected := provider, expected
		t.Run(provider, func(t *testing.T) {
			payload, err := os.ReadFile(filepath.Join("testdata", provider+".json"))
			if err != nil {
				t.Fatal(err)
			}
			service := NewService(nil, Config{SecretNames: []string{"DEPLOY_TOKEN"}})
			eventID := ProviderEventID(provider, payload)
			if eventID == "" {
				t.Fatal("fixture has no provider event id")
			}
			deployment, checks, logs, err := service.parse(WebhookEvent{ID: "evt", WorkspaceID: "workspace", Provider: provider, ProviderEventID: eventID, Payload: payload, ReceivedAt: time.Now().UTC()})
			if err != nil {
				t.Fatal(err)
			}
			if deployment.Status != expected {
				t.Fatalf("status=%s, want %s", deployment.Status, expected)
			}
			if deployment.Repository == "unknown/repository" || deployment.CommitSHA == "unknown" {
				t.Fatalf("fixture was not normalized: %#v", deployment)
			}
			if len(checks) == 0 {
				t.Fatal("expected at least one check")
			}
			if provider == "github" && logs[0] != "token=[REDACTED]" {
				t.Fatalf("log was not redacted: %q", logs[0])
			}
		})
	}
}

func TestIdempotencyAndRecoveryNotification(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	event := WebhookEvent{ID: "evt-1", WorkspaceID: "workspace", Provider: "github", ProviderEventID: "run-1", Payload: []byte(`{}`), PayloadHash: "hash", CorrelationID: "correlation", ReceivedAt: time.Now().UTC()}
	created, err := store.RecordWebhook(ctx, event)
	if err != nil || !created {
		t.Fatalf("first webhook: created=%v err=%v", created, err)
	}
	event.ID = "evt-2"
	created, err = store.RecordWebhook(ctx, event)
	if err != nil || created {
		t.Fatalf("duplicate webhook: created=%v err=%v", created, err)
	}
	if err := store.EnsureDemoRule(ctx, "workspace"); err != nil {
		t.Fatal(err)
	}
	failed := deploymentForTest("failed", StatusFailed, time.Now().Add(-time.Hour))
	created, err = store.SaveDeployment(ctx, failed, nil, nil)
	if err != nil || !created {
		t.Fatalf("save failed deployment: %v", err)
	}
	if err = store.DeliverNotifications(ctx, failed); err != nil {
		t.Fatal(err)
	}
	succeeded := deploymentForTest("success", StatusSuccess, time.Now())
	created, err = store.SaveDeployment(ctx, succeeded, nil, nil)
	if err != nil || !created {
		t.Fatalf("save successful deployment: %v", err)
	}
	if err = store.DeliverNotifications(ctx, succeeded); err != nil {
		t.Fatal(err)
	}
	if err = store.DeliverNotifications(ctx, succeeded); err != nil {
		t.Fatal(err)
	}
	var deliveries int
	if err = store.db.QueryRow(`SELECT COUNT(*) FROM notification_deliveries`).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if deliveries != 2 {
		t.Fatalf("deliveries=%d, want exactly failure plus recovery", deliveries)
	}
}

func TestDeadLetterDoesNotBlockLaterDeployment(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{})
	bad := WebhookEvent{ID: "bad", WorkspaceID: "workspace", Provider: "github", ProviderEventID: "bad", Payload: []byte(`{`), PayloadHash: "bad", CorrelationID: "bad", ReceivedAt: time.Now().UTC()}
	created, err := store.RecordWebhook(ctx, bad)
	if err != nil || !created {
		t.Fatal(err)
	}
	service.process(ctx, bad.ID)
	payload, err := os.ReadFile(filepath.Join("testdata", "github.json"))
	if err != nil {
		t.Fatal(err)
	}
	good := WebhookEvent{ID: "good", WorkspaceID: "workspace", Provider: "github", ProviderEventID: ProviderEventID("github", payload), Payload: payload, PayloadHash: "good", CorrelationID: "good", ReceivedAt: time.Now().UTC()}
	created, err = store.RecordWebhook(ctx, good)
	if err != nil || !created {
		t.Fatal(err)
	}
	service.process(ctx, good.ID)
	items, _, err := store.ListDeployments(ctx, "workspace", ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("deployments=%d, want 1 after failed parsing", len(items))
	}
	deadLetters, err := store.ListDeadLetters(ctx, "workspace")
	if err != nil {
		t.Fatal(err)
	}
	if len(deadLetters) != 1 {
		t.Fatalf("dead letters=%d, want 1", len(deadLetters))
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(context.Background(), filepath.Join(t.TempDir(), "deploy-pulse.db"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
func deploymentForTest(eventID string, status Status, started time.Time) Deployment {
	finished := started.Add(time.Minute)
	if status == StatusSuccess {
		return Deployment{ID: NewDeploymentID(), WorkspaceID: "workspace", Provider: "github", ProviderEventID: eventID, Repository: "api/service", Branch: "main", CommitSHA: eventID, Environment: "production", Actor: "dev", Status: status, OriginalStatus: string(status), StartedAt: started, FinishedAt: finished, Duration: "1m0s", CreatedAt: started}
	}
	return Deployment{ID: NewDeploymentID(), WorkspaceID: "workspace", Provider: "github", ProviderEventID: eventID, Repository: "api/service", Branch: "main", CommitSHA: eventID, Environment: "production", Actor: "dev", Status: status, OriginalStatus: string(status), FailureCategory: "build_failure", FailureSummary: "build failed", StartedAt: started, FinishedAt: finished, Duration: "1m0s", CreatedAt: started}
}
