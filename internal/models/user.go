package models

import "time"

// User maps directly to the users table in PostgreSQL
// GORM reads the struct and creates the table automatically
// Field name rules: Go uses PascalCase, GORM converts to snake_case for DB
//   Name         → name
//   PasswordHash → password_hash
//   CreatedAt    → created_at
type User struct {
	// Primary key — auto-incremented by PostgreSQL
	ID        uint      `json:"id"         gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name  string `json:"name"  gorm:"not null"`
	Email string `json:"email" gorm:"uniqueIndex;not null"`
	// uniqueIndex = PostgreSQL enforces no two users share an email
	// not null = PostgreSQL rejects rows where this column is empty

	PasswordHash string `json:"-" gorm:"column:password_hash;not null"`
	// json:"-" means this field is completely invisible in JSON output
	// It will NEVER appear in any API response
	// This is the correct way to handle sensitive fields
}