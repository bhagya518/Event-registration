package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"event-ticketing-system/internal/database"
	"event-ticketing-system/internal/handlers"
)

func main() {
	// Initialize environment and database
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	router := gin.Default()

	// Health handler depends only on db ping to keep layers clean
	healthHandler := handlers.NewHealthHandler(db)

	router.GET("/health", healthHandler.Health)

	// Event handler exposes event and registration endpoints
	eventHandler := handlers.NewEventHandler(db)
	eventHandler.RegisterRoutes(router)

	if err := router.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to run server: %v", err)
	}
}
