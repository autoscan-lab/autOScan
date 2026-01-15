package domain

import "time"

// Corpus represents an indexed folder or project
type Corpus struct {
	ID        string
	Name      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
