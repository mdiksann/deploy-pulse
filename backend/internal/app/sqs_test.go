package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func TestSQSBatchRetriesAndRecordsDeadLetter(t *testing.T) {
	store := testStore(t)
	defer store.Close()
	service := NewService(store, Config{})
	for _, item := range []struct {
		id      string
		payload []byte
		tries   string
		status  string
		failed  bool
	}{
		{"good", mustReadFixture(t, "github.json"), "1", "processed", false},
		{"retry", []byte(`{broken`), "1", "queued", true},
		{"dead", []byte(`{broken`), "5", "dead_letter", true},
	} {
		event := WebhookEvent{ID: item.id, WorkspaceID: "workspace", Provider: "github", ProviderEventID: item.id, Payload: item.payload, PayloadHash: "hash", CorrelationID: item.id, ReceivedAt: time.Now().UTC()}
		if _, err := store.RecordWebhook(t.Context(), event); err != nil {
			t.Fatal(err)
		}
		batch := events.SQSEvent{Records: []events.SQSMessage{{MessageId: item.id, Body: item.id, Attributes: map[string]string{"ApproximateReceiveCount": item.tries}}}}
		response, err := ProcessSQSBatch(t.Context(), store, service, batch)
		if err != nil || (len(response.BatchItemFailures) == 1) != item.failed {
			t.Fatalf("%s: response=%+v err=%v", item.id, response, err)
		}
		stored, err := store.Webhook(context.Background(), item.id)
		if err != nil || stored.Status != item.status {
			t.Fatalf("%s: status=%q err=%v", item.id, stored.Status, err)
		}
	}
	count, err := store.DeadLetterCount(t.Context(), "workspace")
	if err != nil || count != 1 {
		t.Fatalf("dead letters=%d err=%v", count, err)
	}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
