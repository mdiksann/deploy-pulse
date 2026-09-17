package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultStream = "deploypulse:events"
	defaultGroup  = "deploypulse-workers"
)

type Publisher interface {
	Publish(context.Context, string) error
	Healthy(context.Context) error
}

type RedisStream struct {
	client      *redis.Client
	stream      string
	group       string
	consumer    string
	deadStream  string
	maxAttempts int
	claimIdle   time.Duration
}

func OpenRedisStream(redisURL string) (*RedisStream, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	consumer, err := os.Hostname()
	if err != nil {
		consumer = "worker"
	}
	return &RedisStream{client: redis.NewClient(options), stream: defaultStream, group: defaultGroup, consumer: fmt.Sprintf("%s-%d", consumer, os.Getpid()), deadStream: defaultStream + ":dlq", maxAttempts: 5, claimIdle: 30 * time.Second}, nil
}

func (q *RedisStream) Close() error                      { return q.client.Close() }
func (q *RedisStream) Healthy(ctx context.Context) error { return q.client.Ping(ctx).Err() }
func (q *RedisStream) Publish(ctx context.Context, eventID string) error {
	_, err := q.client.XAdd(ctx, &redis.XAddArgs{Stream: q.stream, Values: map[string]any{"event_id": eventID}}).Result()
	return err
}

func (q *RedisStream) ensureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0").Err()
	if err != nil && !isBusyGroup(err) {
		return err
	}
	return nil
}

func isBusyGroup(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "BUSYGROUP") || strings.Contains(err.Error(), "already exists"))
}

// Consume acknowledges only successful callback completion. Failed messages stay pending for XAUTOCLAIM.
func (q *RedisStream) Consume(ctx context.Context, handle func(context.Context, string) error, onDeadLetter func(context.Context, string, error, int) error) error {
	if err := q.ensureGroup(ctx); err != nil {
		return err
	}
	claimAt, claimStart := time.Now(), "0-0"
	for ctx.Err() == nil {
		if time.Since(claimAt) >= 5*time.Second {
			messages, next, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: q.stream, Group: q.group, Consumer: q.consumer, MinIdle: q.claimIdle, Start: claimStart, Count: 100}).Result()
			if err != nil && !errors.Is(err, redis.Nil) {
				return err
			}
			claimStart, claimAt = next, time.Now()
			for _, message := range messages {
				if err := q.process(ctx, message, handle, onDeadLetter); err != nil {
					return err
				}
			}
		}
		streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: q.group, Consumer: q.consumer, Streams: []string{q.stream, ">"}, Count: 20, Block: 5 * time.Second}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			return err
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := q.process(ctx, message, handle, onDeadLetter); err != nil {
					return err
				}
			}
		}
	}
	return ctx.Err()
}

func (q *RedisStream) process(ctx context.Context, message redis.XMessage, handle func(context.Context, string) error, onDeadLetter func(context.Context, string, error, int) error) error {
	eventID, _ := message.Values["event_id"].(string)
	if eventID == "" {
		return q.ack(ctx, message.ID)
	}
	if err := handle(ctx, eventID); err == nil {
		return q.ack(ctx, message.ID)
	} else {
		attempts, pendingErr := q.attempts(ctx, message.ID)
		if pendingErr != nil {
			return pendingErr
		}
		log.Printf("event_id=%s stream_id=%s attempts=%d process_error=%q", eventID, message.ID, attempts, err)
		if attempts < q.maxAttempts {
			return nil
		}
		if deadErr := onDeadLetter(ctx, eventID, err, attempts); deadErr != nil {
			return deadErr
		}
		if _, deadErr := q.client.XAdd(ctx, &redis.XAddArgs{Stream: q.deadStream, Values: map[string]any{"event_id": eventID, "error": err.Error(), "attempts": attempts}}).Result(); deadErr != nil {
			return deadErr
		}
		return q.ack(ctx, message.ID)
	}
}

func (q *RedisStream) attempts(ctx context.Context, messageID string) (int, error) {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{Stream: q.stream, Group: q.group, Start: messageID, End: messageID, Count: 1}).Result()
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 1, nil
	}
	return int(pending[0].RetryCount), nil
}

func (q *RedisStream) ack(ctx context.Context, messageID string) error {
	return q.client.XAck(ctx, q.stream, q.group, messageID).Err()
}

type Worker struct {
	service *Service
	store   *Store
	queue   *RedisStream
}

func NewWorker(service *Service, store *Store, queue *RedisStream) *Worker {
	return &Worker{service: service, store: store, queue: queue}
}

func (w *Worker) Run(ctx context.Context) error {
	go w.retain(ctx)
	return w.queue.Consume(ctx, w.service.Process, w.deadLetter)
}

func (w *Worker) deadLetter(ctx context.Context, eventID string, processErr error, attempts int) error {
	event, err := w.store.Webhook(ctx, eventID)
	if err != nil {
		return err
	}
	_, err = w.store.AddDeadLetter(ctx, event, processErr, attempts)
	return err
}

func (w *Worker) retain(ctx context.Context) {
	_ = w.store.PruneRetained(ctx)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.store.PruneRetained(ctx); err != nil {
				log.Printf("retention prune failed: %v", err)
			}
		}
	}
}
