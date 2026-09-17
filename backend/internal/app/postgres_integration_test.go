package app

import (
	"context"
	"os"
	"testing"
	"time"
)

// Run with TEST_POSTGRES_URL=postgres://... to verify fresh and repeated production migrations.
func TestPostgresMigrationsAndWebhookConstraint(t *testing.T) {
	url := os.Getenv("TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("set TEST_POSTGRES_URL to run PostgreSQL integration test")
	}
	store, err := OpenStore(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	event := WebhookEvent{ID: "pg-" + NewDeploymentID(), WorkspaceID: "postgres-test", Provider: "github", ProviderEventID: NewDeploymentID(), Payload: []byte(`{}`), PayloadHash: "hash", CorrelationID: "test", ReceivedAt: time.Now().UTC()}
	created, err := store.RecordWebhook(context.Background(), event)
	if err != nil || !created {
		t.Fatalf("first event: created=%v err=%v", created, err)
	}
	event.ID = "pg-duplicate-" + NewDeploymentID()
	created, err = store.RecordWebhook(context.Background(), event)
	if err != nil || created {
		t.Fatalf("duplicate event: created=%v err=%v", created, err)
	}
}
