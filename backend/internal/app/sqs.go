package app

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSQueue struct {
	client *sqs.Client
	url    string
}

func OpenSQSQueue(ctx context.Context, url string) (*SQSQueue, error) {
	if url == "" {
		return nil, errors.New("SQS_QUEUE_URL is required")
	}
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &SQSQueue{client: sqs.NewFromConfig(awsConfig), url: url}, nil
}

func (q *SQSQueue) Publish(ctx context.Context, eventID string) error {
	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: &q.url, MessageBody: &eventID})
	return err
}

func (q *SQSQueue) Healthy(ctx context.Context) error {
	_, err := q.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &q.url, AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
	})
	return err
}

// ProcessSQSBatch reports only failed records, so successful records are deleted by Lambda.
func ProcessSQSBatch(ctx context.Context, store *Store, service *Service, batch events.SQSEvent) (events.SQSEventResponse, error) {
	if len(batch.Records) == 0 {
		return events.SQSEventResponse{}, store.PruneRetained(ctx)
	}
	response := events.SQSEventResponse{BatchItemFailures: make([]events.SQSBatchItemFailure, 0)}
	for _, record := range batch.Records {
		if record.Body == "" {
			log.Printf("discarding empty SQS message %s", record.MessageId)
			continue
		}
		err := service.Process(ctx, record.Body)
		if err == nil {
			continue
		}
		attempts, _ := strconv.Atoi(record.Attributes["ApproximateReceiveCount"])
		log.Printf("event_id=%s attempts=%d process_error=%q", record.Body, attempts, err)
		if attempts >= 5 {
			event, lookupErr := store.Webhook(ctx, record.Body)
			if lookupErr == nil && event.Status != "processed" {
				_, lookupErr = store.AddDeadLetter(ctx, event, err, attempts)
			}
			if lookupErr != nil {
				log.Printf("event_id=%s dead_letter_error=%q", record.Body, lookupErr)
			}
		}
		response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
	}
	return response, nil
}
