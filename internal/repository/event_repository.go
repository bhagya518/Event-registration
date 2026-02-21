package repository

import (
	"context"
	"database/sql"

	"event-ticketing-system/internal/models"
)

// DBTX abstracts *sql.DB and *sql.Tx so repositories can work with or without transactions.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// EventRepository provides access to events and their registrations.
type EventRepository struct {

db DBTX
}

func NewEventRepository(db DBTX) *EventRepository {
	return &EventRepository{db: db}
}

const (
	queryCreateEvent = `
		INSERT INTO events (name, description, capacity, available_slots, organizer_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, capacity, available_slots, organizer_id, created_at
	`

	queryGetAllEvents = `
		SELECT id, name, description, capacity, available_slots, organizer_id, created_at
		FROM events
		ORDER BY created_at DESC
	`

	queryGetEventByID = `
		SELECT id, name, description, capacity, available_slots, organizer_id, created_at
		FROM events
		WHERE id = $1
	`

	queryGetRegistrationsByEvent = `
		SELECT id, user_id, event_id, created_at
		FROM registrations
		WHERE event_id = $1
		ORDER BY created_at ASC
	`

	queryGetRegistrationsByEventWithUser = `
		SELECT
			r.id AS registration_id,
			r.user_id,
			u.name,
			u.email,
			r.created_at AS registered_at
		FROM registrations r
		JOIN users u ON u.id = r.user_id
		WHERE r.event_id = $1
		ORDER BY r.created_at ASC
	`
)

// CreateEvent inserts a new event into the database.
func (r *EventRepository) CreateEvent(ctx context.Context, e *models.Event) (*models.Event, error) {
	row := r.db.QueryRowContext(ctx, queryCreateEvent,
		e.Name,
		e.Description,
		e.Capacity,
		e.AvailableSlots,
		e.OrganizerID,
	)

	var created models.Event
	if err := row.Scan(
		&created.ID,
		&created.Name,
		&created.Description,
		&created.Capacity,
		&created.AvailableSlots,
		&created.OrganizerID,
		&created.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &created, nil
}

// GetAllEvents retrieves all events.
func (r *EventRepository) GetAllEvents(ctx context.Context) ([]*models.Event, error) {
	rows, err := r.db.QueryContext(ctx, queryGetAllEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Description,
			&e.Capacity,
			&e.AvailableSlots,
			&e.OrganizerID,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// GetEventByID retrieves a single event by its ID.
func (r *EventRepository) GetEventByID(ctx context.Context, id int) (*models.Event, error) {
	row := r.db.QueryRowContext(ctx, queryGetEventByID, id)

	var e models.Event
	if err := row.Scan(
		&e.ID,
		&e.Name,
		&e.Description,
		&e.Capacity,
		&e.AvailableSlots,
		&e.OrganizerID,
		&e.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &e, nil
}

// GetRegistrationsByEvent retrieves all registrations for a given event.
func (r *EventRepository) GetRegistrationsByEvent(ctx context.Context, eventID int) ([]*models.Registration, error) {
	rows, err := r.db.QueryContext(ctx, queryGetRegistrationsByEvent, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regs []*models.Registration
	for rows.Next() {
		var rgt models.Registration
		if err := rows.Scan(
			&rgt.ID,
			&rgt.UserID,
			&rgt.EventID,
			&rgt.CreatedAt,
		); err != nil {
			return nil, err
		}
		regs = append(regs, &rgt)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return regs, nil
}

// GetRegistrationsByEventWithUser retrieves registrations for an event joined with user details.
// Returns an empty slice if there are no registrations.
func (r *EventRepository) GetRegistrationsByEventWithUser(ctx context.Context, eventID int) ([]*models.EventRegistrationView, error) {
	rows, err := r.db.QueryContext(ctx, queryGetRegistrationsByEventWithUser, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regs []*models.EventRegistrationView
	for rows.Next() {
		var v models.EventRegistrationView
		if err := rows.Scan(
			&v.RegistrationID,
			&v.UserID,
			&v.Name,
			&v.Email,
			&v.RegisteredAt,
		); err != nil {
			return nil, err
		}
		regs = append(regs, &v)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if regs == nil {
		return []*models.EventRegistrationView{}, nil
	}

	return regs, nil
}
