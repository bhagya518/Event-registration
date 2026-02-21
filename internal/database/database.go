package database

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// NewPostgresDB initializes and returns a new PostgreSQL *sql.DB connection pool.
// It loads environment variables from a .env file if present.
// Supports both individual env vars and full connection string via DATABASE_URL.
func NewPostgresDB() (*sql.DB, error) {
	// Try to load .env from project root
	loadEnvFile(".env")
	loadEnvFile("../.env") // fallback for subdirectories

	// Check for full connection string first (e.g., Neon, AWS RDS, etc.)
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		return openAndVerify(connStr)
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	name := getEnv("DB_NAME", "event_ticketing_system")
	sslmode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, name, sslmode,
	)

	return openAndVerify(dsn)
}

// openAndVerify creates a sql.DB connection and verifies it with Ping.
func openAndVerify(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	log.Println("connected to PostgreSQL database")
	return db, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return // ignore if file doesn't exist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}
