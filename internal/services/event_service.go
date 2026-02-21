package services

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"event-ticketing-system/internal/models"
	"event-ticketing-system/internal/repository"
)

// EventService contains business logic for events and registrations.
type EventService struct {
	eventRepo         *repository.EventRepository
	registrationRepo  *repository.RegistrationRepository
}

func NewEventService(eventRepo *repository.EventRepository, registrationRepo *repository.RegistrationRepository) *EventService {
	return &EventService{
		eventRepo:        eventRepo,
		registrationRepo: registrationRepo,
	}
}

// CreateEventInput represents the payload required to create a new event.
type CreateEventInput struct {
	Name        string
	Description string
	Capacity    int
	OrganizerID int
}

var (
	// ErrInvalidCapacity is returned when event capacity is not positive.
	ErrInvalidCapacity = errors.New("capacity must be greater than zero")
)

// CreateEvent applies business rules and delegates to the repository to persist an event.
func (s *EventService) CreateEvent(ctx context.Context, in CreateEventInput) (*models.Event, error) {
	log.Printf("msg=create_event_attempt organizer_id=%d name=%q capacity=%d", in.OrganizerID, in.Name, in.Capacity)
	if in.Capacity <= 0 {
		log.Printf("msg=create_event_failed_invalid_capacity organizer_id=%d capacity=%d", in.OrganizerID, in.Capacity)
		return nil, ErrInvalidCapacity
	}
	// Initially, available_slots equals capacity.
	event := &models.Event{
		Name:           in.Name,
		Description:    in.Description,
		Capacity:       in.Capacity,
		AvailableSlots: in.Capacity,
		OrganizerID:    in.OrganizerID,
	}

	created, err := s.eventRepo.CreateEvent(ctx, event)
	if err != nil {
		log.Printf("msg=create_event_failed organizer_id=%d err=%v", in.OrganizerID, err)
		return nil, err
	}
	log.Printf("msg=create_event_success event_id=%d organizer_id=%d capacity=%d", created.ID, created.OrganizerID, created.Capacity)
	return created, nil
}

// ListEvents returns all events.
func (s *EventService) ListEvents(ctx context.Context) ([]*models.Event, error) {
	return s.eventRepo.GetAllEvents(ctx)
}

// GetEventByID returns a single event by ID.
// Returns sql.ErrNoRows if the event does not exist.
func (s *EventService) GetEventByID(ctx context.Context, id int) (*models.Event, error) {
	event, err := s.eventRepo.GetEventByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// ListRegistrationsByEvent returns registrations for an event joined with user details.
// Business rule: event must exist; otherwise returns sql.ErrNoRows.
func (s *EventService) ListRegistrationsByEvent(ctx context.Context, eventID int) ([]*models.EventRegistrationView, error) {
	_, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	regs, err := s.eventRepo.GetRegistrationsByEventWithUser(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return regs, nil
}

// RegisterForEvent contains the business logic for registering a user to an event.
// It delegates the critical concurrency-safe registration to the repository method
// that uses a database transaction and SELECT ... FOR UPDATE.
func (s *EventService) RegisterForEvent(ctx context.Context, userID, eventID int) (*models.Registration, error) {
	log.Printf("msg=service_register_attempt user_id=%d event_id=%d", userID, eventID)
	// Business rule: ensure the event exists before attempting registration.
	_, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("msg=service_register_failed_event_not_found user_id=%d event_id=%d", userID, eventID)
			return nil, err
		}
		log.Printf("msg=service_register_failed_event_lookup user_id=%d event_id=%d err=%v", userID, eventID, err)
		return nil, err
	}

	reg, err := s.registrationRepo.RegisterForEvent(ctx, userID, eventID)
	if err != nil {
		if err == repository.ErrEventFull {
			log.Printf("msg=service_register_failed_event_full user_id=%d event_id=%d", userID, eventID)
			return nil, err
		}
		log.Printf("msg=service_register_failed user_id=%d event_id=%d err=%v", userID, eventID, err)
		// Propagate domain-specific errors like repository.ErrEventFull
		return nil, err
	}
	log.Printf("msg=service_register_success user_id=%d event_id=%d registration_id=%d", userID, eventID, reg.ID)

	return reg, nil
}
