package tests

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"sync/atomic"
	"testing"

	"event-ticketing-system/internal/database"
	"event-ticketing-system/internal/models"
	"event-ticketing-system/internal/repository"
	"event-ticketing-system/internal/services"

	_ "github.com/lib/pq"
)

// TestConcurrentBooking simulates multiple users trying to register for
// an event with only 1 available slot. The test verifies that:
// 1. Exactly 1 registration succeeds
// 2. available_slots becomes 0
// 3. No overbooking occurs
func TestConcurrentBooking(t *testing.T) {
	// Initialize database connection
	db, err := database.NewPostgresDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create test organizer user
	organizerID, err := createTestUser(db, "Test Organizer", "organizer@test.com", "ORGANIZER")
	if err != nil {
		t.Fatalf("Failed to create test organizer: %v", err)
	}

	// Create event with capacity = 1
	eventRepo := repository.NewEventRepository(db)
	event := &models.Event{
		Name:           "Limited Capacity Event",
		Description:    "Test event with only 1 slot",
		Capacity:       1,
		AvailableSlots: 1,
		OrganizerID:    organizerID,
	}

	createdEvent, err := eventRepo.CreateEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	eventID := createdEvent.ID
	log.Printf("Created test event with ID=%d, capacity=%d", eventID, createdEvent.Capacity)

	// Create 50 test users upfront (to satisfy FK constraints)
	userIDs := make([]int, 50)
	for i := 0; i < 50; i++ {
		email := "user" + string(rune('0'+i)) + "@test.com"
		if i >= 10 {
			email = "user" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + "@test.com"
		}
		uid, err := createTestUser(db, "Test User", email, "USER")
		if err != nil {
			t.Fatalf("Failed to create test user %d: %v", i, err)
		}
		userIDs[i] = uid
	}

	// Setup registration repository and service
	registrationRepo := repository.NewRegistrationRepository(db)
	registrationService := services.NewEventService(eventRepo, registrationRepo)

	// Spawn 50 goroutines, each trying to register a different user
	const numAttempts = 50
	var wg sync.WaitGroup
	var successCount int32
	var failCount int32

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		userID := userIDs[i]

		go func(uid int) {
			defer wg.Done()

			_, err := registrationService.RegisterForEvent(ctx, uid, eventID)
			if err != nil {
				// Registration failed (expected for 49 out of 50)
				atomic.AddInt32(&failCount, 1)
				return
			}

			// Registration succeeded
			atomic.AddInt32(&successCount, 1)
		}(userID)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	log.Printf("Test results: total=%d, successful=%d, failed=%d", numAttempts, successCount, failCount)

	// Query final event state
	finalEvent, err := eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		t.Fatalf("Failed to get final event state: %v", err)
	}

	// Assertions
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful registration, got %d", successCount)
	}

	if failCount != numAttempts-1 {
		t.Errorf("Expected %d failed registrations, got %d", numAttempts-1, failCount)
	}

	if finalEvent.AvailableSlots != 0 {
		t.Errorf("Expected available_slots to be 0, got %d", finalEvent.AvailableSlots)
	}

	if successCount+failCount != numAttempts {
		t.Errorf("Success + Fail (%d) should equal total attempts (%d)", successCount+failCount, numAttempts)
	}

	// Cleanup: remove test data
	cleanupTestData(db, eventID, userIDs, organizerID)
}

// createTestUser creates a test user and returns the user ID
func createTestUser(db *sql.DB, name, email, role string) (int, error) {
	var id int
	query := `
		INSERT INTO users (name, email, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, role = EXCLUDED.role
		RETURNING id
	`
	err := db.QueryRow(query, name, email, role).Scan(&id)
	return id, err
}

// cleanupTestData removes test data created during the test
func cleanupTestData(db *sql.DB, eventID int, userIDs []int, organizerID int) {
	ctx := context.Background()

	// Delete registrations for the event
	_, err := db.ExecContext(ctx, "DELETE FROM registrations WHERE event_id = $1", eventID)
	if err != nil {
		log.Printf("Failed to cleanup registrations: %v", err)
	}

	// Delete the event
	_, err = db.ExecContext(ctx, "DELETE FROM events WHERE id = $1", eventID)
	if err != nil {
		log.Printf("Failed to cleanup event: %v", err)
	}

	// Delete test users
	for _, uid := range userIDs {
		_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", uid)
		if err != nil {
			log.Printf("Failed to cleanup user %d: %v", uid, err)
		}
	}

	// Delete organizer
	_, err = db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", organizerID)
	if err != nil {
		log.Printf("Failed to cleanup organizer: %v", err)
	}

	log.Println("Test data cleanup completed")
}
