package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GitHubEvent represents a single webhook event received from GitHub.
// Each field maps to a column in the github_events table in PostgreSQL.
type GitHubEvent struct {
	// ID is a UUID — a globally unique string like "a1b2c3d4-...".
	// We use UUIDs instead of auto-incrementing numbers because they're
	// safer to expose in APIs (no one can guess the next ID).
	ID string `gorm:"type:uuid;primaryKey" json:"id"`

	// DeliveryID is the unique ID GitHub assigns to each webhook delivery.
	// Useful for debugging — you can look it up in GitHub's webhook logs.
	DeliveryID string `gorm:"uniqueIndex" json:"delivery_id"`

	// EventType is what kind of event this is: "push", "pull_request",
	// "issues", "star", etc. GitHub sends this in the X-GitHub-Event header.
	EventType string `gorm:"index" json:"event_type"`

	// RepoFullName is the repository name like "torvalds/linux".
	RepoFullName string `gorm:"index" json:"repo_full_name"`

	// Sender is the GitHub username of the person who triggered the event.
	Sender string `json:"sender"`

	// Payload is the raw JSON body GitHub sent us — stored as a string.
	// We keep the full payload so we can re-process it later if needed.
	Payload string `gorm:"type:text" json:"payload"`

	// Processed tells us whether a background worker has handled this event.
	Processed bool `gorm:"default:false" json:"processed"`

	// CreatedAt is automatically set by GORM to the current time on insert.
	CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate is a GORM "hook" — a function that runs automatically
// before every INSERT. We use it to generate a UUID for the ID field.
// Without this, ID would be empty and the insert would fail.
func (e *GitHubEvent) BeforeCreate(tx *gorm.DB) error {
	e.ID = uuid.New().String()
	return nil
}