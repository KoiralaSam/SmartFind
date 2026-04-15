package domain

import "time"

// Staff is the persistence shape for the staff table.
// PasswordHash is set when loaded from the database for authentication only;
// it must not be exposed outside the service layer.
type Staff struct {
	ID           string
	FullName     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
