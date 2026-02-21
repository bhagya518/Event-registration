package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"event-ticketing-system/internal/middleware"
	"event-ticketing-system/internal/repository"
	"event-ticketing-system/internal/services"
)

// EventHandler exposes HTTP endpoints for events and registrations.
type EventHandler struct {
	service *services.EventService
}

// NewEventHandler constructs an EventHandler with its dependencies wired.
func NewEventHandler(db *sql.DB) *EventHandler {
	// Repositories
	eventRepo := repository.NewEventRepository(db)
	registrationRepo := repository.NewRegistrationRepository(db)

	// Service layer
	service := services.NewEventService(eventRepo, registrationRepo)

	return &EventHandler{service: service}
}

// errorResponse is a structured error payload for consistent JSON responses.
type errorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

// CreateEventRequest is the JSON payload for creating a new event.
type CreateEventRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity" binding:"required"`
	OrganizerID int    `json:"organizer_id" binding:"required"`
}

// RegisterRoutes registers event-related routes on the provided Gin engine.
func (h *EventHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/events", h.CreateEvent)
	r.GET("/events", h.ListEvents)
	r.GET("/events/:id", h.GetEventByID)
	r.POST("/events/:id/register", middleware.UserIDMiddleware(), h.RegisterForEvent)
	r.GET("/events/:id/registrations", h.GetRegistrationsByEvent)
	r.POST("/events/:id/simulate", h.SimulateConcurrentRegistrations)
}

// CreateEvent handles POST /events.
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("msg=http_create_event_invalid_request err=%v", err)
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_request", Detail: err.Error()})
		return
	}

	input := services.CreateEventInput{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
		OrganizerID: req.OrganizerID,
	}

	event, err := h.service.CreateEvent(c.Request.Context(), input)
	if err != nil {
		if err == services.ErrInvalidCapacity {
			log.Printf("msg=http_create_event_invalid_capacity organizer_id=%d capacity=%d", req.OrganizerID, req.Capacity)
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_capacity", Detail: err.Error()})
			return
		}
		log.Printf("msg=http_create_event_internal_error organizer_id=%d err=%v", req.OrganizerID, err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Detail: err.Error()})
		return
	}
	log.Printf("msg=http_create_event_success event_id=%d organizer_id=%d", event.ID, event.OrganizerID)

	c.JSON(http.StatusCreated, event)
}

// ListEvents handles GET /events.
func (h *EventHandler) ListEvents(c *gin.Context) {
	events, err := h.service.ListEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Detail: err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetEventByID handles GET /events/:id.
func (h *EventHandler) GetEventByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_event_id", Detail: "id must be a positive integer"})
		return
	}

	event, err := h.service.GetEventByID(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, errorResponse{Error: "not_found", Detail: "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Detail: err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

// RegisterForEvent handles POST /events/:id/register.
func (h *EventHandler) RegisterForEvent(c *gin.Context) {
	idStr := c.Param("id")
	eventID, err := strconv.Atoi(idStr)
	if err != nil || eventID <= 0 {
		log.Printf("msg=http_register_invalid_event_id id=%q", idStr)
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_event_id", Detail: "id must be a positive integer"})
		return
	}

	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		log.Printf("msg=http_register_missing_user_id_in_context event_id=%d", eventID)
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "missing_user_id", Detail: "X-User-ID header is required"})
		return
	}

	log.Printf("msg=http_register_attempt user_id=%d event_id=%d", userID, eventID)

	reg, err := h.service.RegisterForEvent(c.Request.Context(), userID, eventID)
	if err != nil {
		if err == repository.ErrEventFull {
			log.Printf("msg=http_register_event_full user_id=%d event_id=%d", userID, eventID)
			c.JSON(http.StatusConflict, errorResponse{Error: "event_full", Detail: err.Error()})
			return
		}
		if err == sql.ErrNoRows {
			log.Printf("msg=http_register_event_not_found user_id=%d event_id=%d", userID, eventID)
			c.JSON(http.StatusNotFound, errorResponse{Error: "not_found", Detail: "event not found"})
			return
		}
		log.Printf("msg=http_register_internal_error user_id=%d event_id=%d err=%v", userID, eventID, err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Detail: err.Error()})
		return
	}
	log.Printf("msg=http_register_success user_id=%d event_id=%d registration_id=%d", reg.UserID, eventID, reg.ID)

	c.JSON(http.StatusCreated, reg)
}

// GetRegistrationsByEvent handles GET /events/:id/registrations.
func (h *EventHandler) GetRegistrationsByEvent(c *gin.Context) {
	idStr := c.Param("id")
	eventID, err := strconv.Atoi(idStr)
	if err != nil || eventID <= 0 {
		log.Printf("msg=http_list_registrations_invalid_event_id id=%q", idStr)
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_event_id", Detail: "id must be a positive integer"})
		return
	}

	regs, err := h.service.ListRegistrationsByEvent(c.Request.Context(), eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("msg=http_list_registrations_event_not_found event_id=%d", eventID)
			c.JSON(http.StatusNotFound, errorResponse{Error: "not_found", Detail: "event not found"})
			return
		}
		log.Printf("msg=http_list_registrations_internal_error event_id=%d err=%v", eventID, err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Detail: err.Error()})
		return
	}

	c.JSON(http.StatusOK, regs)
}

// SimulateConcurrentRegistrations handles POST /events/:id/simulate.
//
// It spawns 100 goroutines, each attempting to register a different user for
// the same event concurrently. The underlying registration logic uses a
// transaction with SELECT ... FOR UPDATE to ensure that, even under heavy
// concurrency, the number of successful registrations never exceeds the
// available capacity of the event.
func (h *EventHandler) SimulateConcurrentRegistrations(c *gin.Context) {
	idStr := c.Param("id")
	eventID, err := strconv.Atoi(idStr)
	if err != nil || eventID <= 0 {
		log.Printf("msg=http_simulate_invalid_event_id id=%q", idStr)
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_event_id", Detail: "id must be a positive integer"})
		return
	}

	const totalAttempts = 20

	var wg sync.WaitGroup
	var successCount int32
	var failCount int32

	// Launch 20 concurrent registration attempts with distinct user IDs.
	for i := 0; i < totalAttempts; i++ {
		wg.Add(1)
		userID := i + 1 // simple unique user id for simulation

		go func(uid int) {
			defer wg.Done()

			_, err := h.service.RegisterForEvent(c.Request.Context(), uid, eventID)
			if err != nil {
				// Count all failures (including event full and other errors) as failed attempts.
				atomic.AddInt32(&failCount, 1)
				return
			}

			atomic.AddInt32(&successCount, 1)
		}(userID)
	}

	wg.Wait()
	log.Printf("msg=http_simulate_results event_id=%d total_attempts=%d successful=%d failed=%d", eventID, totalAttempts, successCount, failCount)

	c.JSON(http.StatusOK, gin.H{
		"total_attempts": totalAttempts,
		"successful":     successCount,
		"failed":         failCount,
	})
}
