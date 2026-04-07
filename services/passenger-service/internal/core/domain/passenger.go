package domain

import "time"

// Passenger maps to the passengers table.
type Passenger struct {
	ID        string
	Email     string
	FullName  string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
