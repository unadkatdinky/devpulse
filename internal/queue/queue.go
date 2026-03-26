package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// EventQueueKey is the name of the Redis list used as our job queue.
	// Think of this as the name of the conveyor belt.
	EventQueueKey = "devpulse:events:queue"
)

// EventJob is the unit of work we put into Redis.
// It must be JSON-serializable because Redis stores strings, not Go structs.
type EventJob struct {
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	RepoFullName string `json:"repo_full_name"`
	DeliveryID   string `json:"delivery_id"`
}

// Queue wraps the Redis client and provides Push/Pop methods.
type Queue struct {
	client *redis.Client
}

// New creates a new Queue.
func New(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// Push adds a job to the queue.
// json.Marshal converts the Go struct to a JSON string.
// LPUSH pushes it onto the left end of the Redis list.
func (q *Queue) Push(ctx context.Context, job EventJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := q.client.LPush(ctx, EventQueueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to push job to Redis: %w", err)
	}

	return nil
}

// Pop removes and returns the next job from the queue.
// BRPOP pops from the right end (opposite of LPUSH = FIFO order).
// The timeout means "wait up to X seconds for a job to appear".
// If nothing appears in that time, return nil — the worker will loop and try again.
func (q *Queue) Pop(ctx context.Context, timeout time.Duration) (*EventJob, error) {
	// BRPop returns a slice: [key, value]
	// key is the queue name, value is the job JSON
	result, err := q.client.BRPop(ctx, timeout, EventQueueKey).Result()
	if err != nil {
		// redis.Nil means the timeout expired with no jobs — not a real error
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop job from Redis: %w", err)
	}

	// result[1] is the job JSON (result[0] is the queue key name)
	var job EventJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// Len returns how many jobs are currently waiting in the queue.
// Useful for monitoring and debugging.
func (q *Queue) Len(ctx context.Context) (int64, error) {
	length, err := q.client.LLen(ctx, EventQueueKey).Result()
	if err != nil {
		return 0, err
	}
	return length, nil
}

// StartWorkers launches N worker goroutines that continuously pop and process jobs.
// processFunc is a function you pass in that does the actual work for each job.
// This pattern is called "dependency injection" — the queue doesn't know what
// to do with jobs, it just manages the queue. You tell it what to do.
func (q *Queue) StartWorkers(ctx context.Context, count int, processFunc func(EventJob)) {
	log.Printf("Starting %d Redis queue workers", count)
	for i := 0; i < count; i++ {
		go func(workerID int) {
			log.Printf("Redis worker %d started", workerID)
			for {
				// Check if context is cancelled (server shutting down)
				select {
				case <-ctx.Done():
					log.Printf("Redis worker %d shutting down", workerID)
					return
				default:
					// Try to get a job — wait up to 2 seconds
					job, err := q.Pop(ctx, 2*time.Second)
					if err != nil {
						log.Printf("Redis worker %d error: %v", workerID, err)
						continue
					}
					// nil job means timeout expired with no jobs — loop again
					if job == nil {
						continue
					}
					// Process the job
					processFunc(*job)
				}
			}
		}(i + 1)
	}
}