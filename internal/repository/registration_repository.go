package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"event-ticketing-system/internal/models"
)

var (
	// ErrEventFull is returned when there are no available slots left for an event.
	ErrEventFull = errors.New("event is full")
)

const (
	querySelectAvailableSlotsForUpdate = `
		SELECT available_slots
		FROM events
		WHERE id = $1
		FOR UPDATE
	`

	queryInsertRegistration = `
		INSERT INTO registrations (user_id, event_id)
		VALUES ($1, $2)
		RETURNING id, user_id, event_id, created_at
	`

	queryDecrementAvailableSlots = `
		UPDATE events
		SET available_slots = available_slots - 1
		WHERE id = $1
	`
)

// RegistrationRepository handles event registration operations.
type RegistrationRepository struct {
	db *sql.DB
}

func NewRegistrationRepository(db *sql.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

// RegisterForEvent performs a transaction-safe registration for a given user and event.
//
// Concurrency control explanation:
// - The SELECT ... FOR UPDATE statement acquires a row-level lock on the matching
//   event row in the events table for the duration of the transaction.
// - While one transaction holds this lock, any other concurrent transaction
//   attempting to SELECT ... FOR UPDATE the same event row will block until the
//   first transaction commits or rolls back.
// - This ensures that the "available_slots" value read and updated in this
//   transaction is isolated from concurrent modifications, preventing race
//   conditions where multiple users could see the same available_slots > 0 and
//   overbook the last remaining spots.
func (r *RegistrationRepository) RegisterForEvent(ctx context.Context, userID, eventID int) (*models.Registration, error) {
	log.Printf("msg=registration_attempt user_id=%d event_id=%d", userID, eventID)

	// Start a database transaction.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("msg=registration_tx_begin_failed user_id=%d event_id=%d err=%v", userID, eventID, err)
		return nil, err
	}

	// Always ensure we roll back if something goes wrong before commit.
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("msg=registration_tx_rollback_failed user_id=%d event_id=%d err=%v", userID, eventID, rbErr)
				return
			}
			log.Printf("msg=registration_tx_rolled_back user_id=%d event_id=%d", userID, eventID)
		}
	}()

	// 1) Lock the event row and read current available_slots.
	var availableSlots int
	// SELECT ... FOR UPDATE acquires a row-level lock on this event row.
	if err = tx.QueryRowContext(ctx, querySelectAvailableSlotsForUpdate, eventID).Scan(&availableSlots); err != nil {
		return nil, err
	}

	// 2) Check capacity.
	if availableSlots <= 0 {
		// No slots left; rollback and return a domain-specific error.
		log.Printf("msg=event_full user_id=%d event_id=%d", userID, eventID)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("msg=registration_tx_rollback_failed user_id=%d event_id=%d err=%v", userID, eventID, rbErr)
		}
		return nil, ErrEventFull
	}

	// 3) Insert registration record.
	var reg models.Registration
	row := tx.QueryRowContext(ctx, queryInsertRegistration, userID, eventID)
	if err = row.Scan(&reg.ID, &reg.UserID, &reg.EventID, &reg.CreatedAt); err != nil {
		log.Printf("msg=registration_insert_failed user_id=%d event_id=%d err=%v", userID, eventID, err)
		return nil, err
	}

	// 4) Decrement available_slots for the event.
	if _, err = tx.ExecContext(ctx, queryDecrementAvailableSlots, eventID); err != nil {
		log.Printf("msg=available_slots_decrement_failed user_id=%d event_id=%d err=%v", userID, eventID, err)
		return nil, err
	}

	// 5) Commit the transaction; this releases the row lock.
	if err = tx.Commit(); err != nil {
		log.Printf("msg=registration_tx_commit_failed user_id=%d event_id=%d err=%v", userID, eventID, err)
		return nil, err
	}

	log.Printf("msg=registration_success user_id=%d event_id=%d registration_id=%d", userID, eventID, reg.ID)

	return &reg, nil
}
