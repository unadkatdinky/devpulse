// package worker

// import (
// 	"encoding/json"
// 	"log"

// 	"github.com/unadkatdinky/devpulse/internal/models"
// 	"gorm.io/gorm"
// )

// // EventJob is the unit of work we pass through the channel.
// // It contains everything a worker needs to process one webhook event.
// type EventJob struct {
// 	Event *models.GitHubEvent
// }

// // Pool manages a group of background workers and the channel between them.
// type Pool struct {
// 	// jobs is the channel. Think of it as a conveyor belt.
// 	// The HTTP handler puts jobs on one end; workers pick them off the other.
// 	// The number 100 is the buffer — up to 100 jobs can wait in line
// 	// before the channel starts blocking.
// 	jobs chan EventJob

// 	// db is the database connection shared by all workers.
// 	db *gorm.DB

// 	// size is how many worker goroutines to run simultaneously.
// 	size int
// }

// // New creates a new worker pool but doesn't start it yet.
// func New(db *gorm.DB, size int) *Pool {
// 	return &Pool{
// 		jobs: make(chan EventJob, 100),
// 		db:   db,
// 		size: size,
// 	}
// }

// // Start launches the worker goroutines. Call this once when your app starts.
// // Each worker runs in its own goroutine — they all watch the same channel
// // and race to pick up the next job.
// func (p *Pool) Start() {
// 	log.Printf("Starting worker pool with %d workers", p.size)
// 	for i := 0; i < p.size; i++ {
// 		// 'go' means "run this function in the background"
// 		// i+1 is just a label so logs say "Worker 1", "Worker 2", etc.
// 		go p.runWorker(i + 1)
// 	}
// }

// // Submit puts a new job on the channel (the conveyor belt).
// // The HTTP handler calls this after receiving a webhook.
// // This returns immediately — it doesn't wait for the job to finish.
// func (p *Pool) Submit(job EventJob) {
// 	p.jobs <- job
// }

// // runWorker is the function each background goroutine runs.
// // It loops forever, waiting for jobs to appear on the channel.
// // The 'for job := range p.jobs' syntax means:
// // "keep looping, and each time a job appears, put it in 'job' and run the body"
// func (p *Pool) runWorker(id int) {
// 	log.Printf("Worker %d started", id)
// 	for job := range p.jobs {
// 		p.processEvent(id, job)
// 	}
// }

// // processEvent is the actual work a worker does for each job.
// // Right now it: logs the event, parses the payload, marks it as processed.
// // In Day 4+ you'll add real analytics logic here.
// func (p *Pool) processEvent(workerID int, job EventJob) {
// 	event := job.Event
// 	log.Printf("Worker %d processing event: type=%s repo=%s delivery=%s",
// 		workerID, event.EventType, event.RepoFullName, event.DeliveryID)

// 	// Parse the raw JSON payload into a generic map so we can inspect it.
// 	// map[string]interface{} means "a map where keys are strings and
// 	// values can be anything" — it's Go's way of handling unknown JSON shapes.
// 	var payload map[string]interface{}
// 	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
// 		log.Printf("Worker %d: failed to parse payload for event %s: %v",
// 			workerID, event.ID, err)
// 		return
// 	}

// 	// Log some interesting fields from the payload depending on event type.
// 	switch event.EventType {
// 	case "push":
// 		if commits, ok := payload["commits"].([]interface{}); ok {
// 			log.Printf("Worker %d: push event has %d commits", workerID, len(commits))
// 		}
// 	case "pull_request":
// 		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
// 			log.Printf("Worker %d: PR title: %v", workerID, pr["title"])
// 		}
// 	case "issues":
// 		if issue, ok := payload["issue"].(map[string]interface{}); ok {
// 			log.Printf("Worker %d: Issue title: %v", workerID, issue["title"])
// 		}
// 	}

// 	// Mark the event as processed in the database.
// 	// db.Model picks the row by ID, then Update changes just the 'processed' column.
// 	result := p.db.Model(&models.GitHubEvent{}).
// 		Where("id = ?", event.ID).
// 		Update("processed", true)

// 	if result.Error != nil {
// 		log.Printf("Worker %d: failed to mark event %s as processed: %v",
// 			workerID, event.ID, result.Error)
// 		return
// 	}

// 	log.Printf("Worker %d: successfully processed event %s", workerID, event.ID)
// }

package worker

import (
	"encoding/json"
	"log"

	"github.com/unadkatdinky/devpulse/internal/models"
	"github.com/unadkatdinky/devpulse/internal/queue"
	"gorm.io/gorm"
)

// Processor handles the actual work for each event job.
// It's separate from the queue so the queue package stays generic.
type Processor struct {
	db *gorm.DB
}

// New creates a new Processor.
func New(db *gorm.DB) *Processor {
	return &Processor{db: db}
}

// ProcessJob is the function passed to queue.StartWorkers.
// It receives a job, fetches the full event from PostgreSQL,
// processes it, and marks it as done.
func (p *Processor) ProcessJob(job queue.EventJob) {
	log.Printf("Processing job: type=%s repo=%s delivery=%s",
		job.EventType, job.RepoFullName, job.DeliveryID)

	// Fetch the full event from PostgreSQL using the ID from the job.
	var event models.GitHubEvent
	if err := p.db.Where("id = ?", job.EventID).First(&event).Error; err != nil {
		log.Printf("Worker: failed to fetch event %s: %v", job.EventID, err)
		return
	}

	// Parse the raw payload for event-specific logic.
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		log.Printf("Worker: failed to parse payload for event %s: %v", job.EventID, err)
		return
	}

	// Event-specific processing.
	switch event.EventType {
	case "push":
		if commits, ok := payload["commits"].([]interface{}); ok {
			log.Printf("Worker: push event with %d commits in %s",
				len(commits), event.RepoFullName)
		}
	case "pull_request":
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			log.Printf("Worker: PR '%v' in %s", pr["title"], event.RepoFullName)
		}
	case "issues":
		if issue, ok := payload["issue"].(map[string]interface{}); ok {
			log.Printf("Worker: Issue '%v' in %s", issue["title"], event.RepoFullName)
		}
	default:
		log.Printf("Worker: received %s event in %s", event.EventType, event.RepoFullName)
	}

	// Mark as processed in PostgreSQL.
	if err := p.db.Model(&models.GitHubEvent{}).
		Where("id = ?", event.ID).
		Update("processed", true).Error; err != nil {
		log.Printf("Worker: failed to mark event %s as processed: %v", event.ID, err)
		return
	}

	log.Printf("Worker: successfully processed event %s", event.ID)
}