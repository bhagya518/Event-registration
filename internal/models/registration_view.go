package models

import "time"

// EventRegistrationView represents a registration joined with user details for API responses.
type EventRegistrationView struct {
	RegistrationID int       `json:"registration_id"`
	UserID         int       `json:"user_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	RegisteredAt   time.Time `json:"registered_at"`
}
