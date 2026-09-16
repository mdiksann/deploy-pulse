package app

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Run with REDIS_TEST_URL=redis://localhost:6379/15 to exercise real Redis Streams semantics.
func TestRedisStreamACKReclaimAndDLQ(t *testing.T) {
	url := os.Getenv("REDIS_TEST_URL")
	if url == "" {
		t.Skip("set REDIS_TEST_URL to run Redis integration test")
	}
	q, err := OpenRedisStream(url)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	id := uuid.NewString()
	q.stream, q.group, q.deadStream, q.consumer, q.claimIdle, q.maxAttempts = "test:events:"+id, "test:workers:"+id, "test:events:"+id+":dlq", "first", 0, 2
	if err := q.ensureGroup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := q.Publish(t.Context(), "event-1"); err != nil {
		t.Fatal(err)
	}
	message := readTestMessage(t, q, ">")
	if err := q.process(t.Context(), message, func(context.Context, string) error { return errors.New("crashed before commit") }, func(context.Context, string, error, int) error { return nil }); err != nil {
		t.Fatal(err)
	}
	pending, err := q.client.XPending(t.Context(), q.stream, q.group).Result()
	if err != nil || pending.Count != 1 {
		t.Fatalf("pending after failed process = %#v, %v", pending, err)
	}
	q.consumer = "reclaimer"
	claimed, _, err := q.client.XAutoClaim(t.Context(), &redis.XAutoClaimArgs{Stream: q.stream, Group: q.group, Consumer: q.consumer, MinIdle: 0, Start: "0-0", Count: 1}).Result()
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = %#v, %v", claimed, err)
	}
	committed := false
	if err := q.process(t.Context(), claimed[0], func(context.Context, string) error { committed = true; return nil }, func(context.Context, string, error, int) error { return nil }); err != nil {
		t.Fatal(err)
	}
	pending, err = q.client.XPending(t.Context(), q.stream, q.group).Result()
	if err != nil || pending.Count != 0 || !committed {
		t.Fatalf("ACK before commit or pending delivery remained: committed=%v pending=%#v err=%v", committed, pending, err)
	}

	if err := q.Publish(t.Context(), "event-2"); err != nil {
		t.Fatal(err)
	}
	message = readTestMessage(t, q, ">")
	dlqCalled := false
	if err := q.process(t.Context(), message, func(context.Context, string) error { return errors.New("permanent failure") }, func(context.Context, string, error, int) error { dlqCalled = true; return nil }); err != nil {
		t.Fatal(err)
	}
	claimed, _, err = q.client.XAutoClaim(t.Context(), &redis.XAutoClaimArgs{Stream: q.stream, Group: q.group, Consumer: q.consumer, MinIdle: 0, Start: "0-0", Count: 1}).Result()
	if err != nil || len(claimed) != 1 {
		t.Fatalf("second claim = %#v, %v", claimed, err)
	}
	if err := q.process(t.Context(), claimed[0], func(context.Context, string) error { return errors.New("permanent failure") }, func(context.Context, string, error, int) error { dlqCalled = true; return nil }); err != nil {
		t.Fatal(err)
	}
	pending, err = q.client.XPending(t.Context(), q.stream, q.group).Result()
	if err != nil || pending.Count != 0 || !dlqCalled {
		t.Fatalf("DLQ result: called=%v pending=%#v err=%v", dlqCalled, pending, err)
	}
	entries, err := q.client.XLen(t.Context(), q.deadStream).Result()
	if err != nil || entries != 1 {
		t.Fatalf("DLQ stream entries=%d err=%v", entries, err)
	}
}

func readTestMessage(t *testing.T, q *RedisStream, position string) redis.XMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: q.group, Consumer: q.consumer, Streams: []string{q.stream, position}, Count: 1, Block: time.Second}).Result()
	if err != nil || len(streams) != 1 || len(streams[0].Messages) != 1 {
		t.Fatalf("read stream = %#v, %v", streams, err)
	}
	return streams[0].Messages[0]
}
