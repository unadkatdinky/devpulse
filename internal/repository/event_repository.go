package repository

import (
	"github.com/unadkatdinky/devpulse/internal/models"
	"gorm.io/gorm"
)

// EventRepository handles all database operations for GitHubEvents.
type EventRepository struct {
	db *gorm.DB
}

// NewEventRepository creates a new EventRepository.
func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

// GetStats returns aggregate statistics about events.
// This is a struct just for returning stats — not a database model.
type EventStats struct {
	TotalEvents    int64            `json:"total_events"`
	ProcessedCount int64            `json:"processed_count"`
	ByType         []EventTypeCount `json:"by_type"`
	TopRepos       []RepoCount      `json:"top_repos"`
}

// EventTypeCount holds a count for one event type.
type EventTypeCount struct {
	EventType string `json:"event_type" gorm:"column:event_type"`
	Count     int64  `json:"count" gorm:"column:count"`
}

// RepoCount holds a count for one repository.
type RepoCount struct {
	RepoFullName string `json:"repo_full_name" gorm:"column:repo_full_name"`
	Count        int64  `json:"count" gorm:"column:count"`
}

// GetStats queries the database for dashboard statistics.
func (r *EventRepository) GetStats() (*EventStats, error) {
	stats := &EventStats{}

	// Count total events.
	// db.Model sets which table, db.Count fills the variable.
	if err := r.db.Model(&models.GitHubEvent{}).Count(&stats.TotalEvents).Error; err != nil {
		return nil, err
	}

	// Count processed events.
	if err := r.db.Model(&models.GitHubEvent{}).
		Where("processed = ?", true).
		Count(&stats.ProcessedCount).Error; err != nil {
		return nil, err
	}

	// Count events grouped by type.
	// Raw SQL here because GORM's group-by syntax is verbose.
	// We select event_type and COUNT(*) as count, group by event_type,
	// order by count descending so the most common type is first.
	if err := r.db.Model(&models.GitHubEvent{}).
		Select("event_type, COUNT(*) as count").
		Group("event_type").
		Order("count DESC").
		Scan(&stats.ByType).Error; err != nil {
		return nil, err
	}

	// Count events grouped by repository — top 5 most active repos.
	if err := r.db.Model(&models.GitHubEvent{}).
		Select("repo_full_name, COUNT(*) as count").
		Group("repo_full_name").
		Order("count DESC").
		Limit(5).
		Scan(&stats.TopRepos).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// ListEvents returns a paginated list of events, newest first.
// page starts at 1. pageSize is how many per page.
func (r *EventRepository) ListEvents(page, pageSize int) ([]models.GitHubEvent, int64, error) {
	var events []models.GitHubEvent
	var total int64

	// Count total matching rows (for the frontend to know how many pages exist).
	if err := r.db.Model(&models.GitHubEvent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset.
	// Page 1 = offset 0, Page 2 = offset pageSize, Page 3 = offset 2*pageSize
	offset := (page - 1) * pageSize

	// Fetch the page.
	// We omit the Payload field — it can be huge and the list view doesn't need it.
	// Omit means "select all columns except these"
	if err := r.db.
		Omit("payload").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// GetEventByID returns a single event with its full payload.
func (r *EventRepository) GetEventByID(id string) (*models.GitHubEvent, error) {
	var event models.GitHubEvent

	// First finds the first row matching the condition.
	// If no row is found, GORM returns gorm.ErrRecordNotFound.
	result := r.db.Where("id = ?", id).First(&event)
	if result.Error != nil {
		return nil, result.Error
	}

	return &event, nil
}