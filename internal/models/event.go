package models

import "time"

// Event represents an event that users can register for.
type Event struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Capacity       int       `json:"capacity"`
	AvailableSlots int       `json:"available_slots"`
	OrganizerID    int       `json:"organizer_id"`
	CreatedAt      time.Time `json:"created_at"`
}
