package models

import "time"

// Registration represents a user's registration to an event.
type Registration struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	EventID   int       `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`
}
