# Event Registration & Ticketing System

A production-ready REST API for event registration and ticketing with advanced concurrency control, built with Go, Gin framework, and PostgreSQL.

## 🎯 Overview

This system demonstrates enterprise-level event management with **race condition prevention** using database-level locking. It handles the critical challenge of concurrent registrations for limited-capacity events, ensuring no overbooking occurs even under high load.

## ✨ Key Features

- **🎫 Event Management**: Create, read, and manage events with capacity controls
- **👤 User Registration**: Secure registration with middleware-based user identification  
- **🔒 Concurrency Control**: Transaction-safe registration with `SELECT FOR UPDATE` row-level locking
- **📊 Real-time Analytics**: Track available slots and registration statistics
- **🧪 Automated Testing**: Built-in concurrent load simulation (20 users)
- **🛡️ Error Handling**: Comprehensive error responses with proper HTTP status codes
- **📝 Structured Logging**: Request tracking with user and event context

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Layer (Gin)                     │
├─────────────────────────────────────────────────────────┤
│                 Business Logic Layer                    │
├─────────────────────────────────────────────────────────┤
│              Data Access Layer (Repository)             │
├─────────────────────────────────────────────────────────┤
│            Database Layer (PostgreSQL)                  │
└─────────────────────────────────────────────────────────┘
```

### Clean Architecture Principles

- **Handlers**: HTTP request/response processing only
- **Services**: Business logic and validation rules
- **Repository**: Database operations and transaction management
- **Models**: Data structures and entities
- **Middleware**: Cross-cutting concerns (authentication, logging)

## 🗄️ Database Schema

### Core Tables

```sql
users (id, name, email, role, created_at)
    ├── 1:N relationship with events (organizer_id)
    └── 1:N relationship with registrations

events (id, name, description, capacity, available_slots, organizer_id, created_at)
    ├── 1:N relationship with registrations
    └── N:1 relationship with users (organizer_id)

registrations (id, user_id, event_id, created_at)
    ├── N:1 relationship with users (user_id)
    ├── N:1 relationship with events (event_id)
    └── UNIQUE constraint (user_id, event_id) - prevents duplicate registrations
```

### Key Constraints

- **Foreign Keys**: Referential integrity between tables
- **Check Constraints**: `capacity > 0`, `available_slots >= 0`
- **Unique Index**: `(user_id, event_id)` - one registration per user per event
- **Indexes**: `event_id`, `organizer_id` for common queries

## 🔒 Concurrency Control Strategy

### The Race Condition Problem

When multiple users attempt to register for the last available slot simultaneously:

```
Time T1: User A reads available_slots = 1
Time T2: User B reads available_slots = 1
Time T3: User C reads available_slots = 1
Time T4: User A registers (decrements to 0)
Time T5: User B registers (decrements to -1) → OVERBOOKING
```

### Database-Level Solution

The system uses **pessimistic locking** with PostgreSQL's `SELECT FOR UPDATE` inside a transaction:

```go
func (r *RegistrationRepository) RegisterForEvent(userID, eventID int) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var availableSlots int
    err = tx.QueryRow(
        "SELECT available_slots FROM events WHERE id = $1 FOR UPDATE",
        eventID,
    ).Scan(&availableSlots)
    if err != nil {
        return err
    }

    if availableSlots <= 0 {
        return errors.New("event is full")
    }

    if _, err := tx.Exec(
        "INSERT INTO registrations (user_id, event_id) VALUES ($1, $2)",
        userID, eventID,
    ); err != nil {
        return err
    }

    if _, err := tx.Exec(
        "UPDATE events SET available_slots = available_slots - 1 WHERE id = $1",
        eventID,
    ); err != nil {
        return err
    }

    return tx.Commit()
}
```

### Why This Works

1. **Row-Level Lock**: `FOR UPDATE` locks the specific event row so other transactions wait.
2. **Atomic Transaction**: Read, insert, and update happen as a single unit.
3. **Validation Under Lock**: `available_slots` is checked while the row is locked.
4. **Rollback Safety**: Any error aborts the transaction and leaves data consistent.

## 🚀 Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL 13+
- Git

### Installation

```bash
git clone https://github.com/bhagya518/Event-registration.git
cd Event-registration

go mod tidy
cp .env.example .env
# Edit .env with your PostgreSQL connection details
```

### Environment Configuration

```bash
# Option 1: Cloud / managed Postgres
DATABASE_URL=postgresql://username:password@host:port/database?sslmode=require

# Option 2: Local Postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=event_ticketing_system
DB_SSLMODE=disable

PORT=8080
```

### Database Setup

```bash
psql "postgresql://username:password@host:port/database" \
  -f migrations/001_create_tables.sql
# or
psql "$DATABASE_URL" -f migrations/001_create_tables.sql
```

### Run the Server

```bash
go run ./cmd/server
# API available at http://localhost:8080
```

## 📚 API Documentation

### Base URL

```text
http://localhost:8080
```

### Health Check

```http
GET /health
```

**Response (200):**

```json
{
  "status": "ok",
  "message": "event-ticketing-system API is running"
}
```

### List Events

```http
GET /events
```

**Response (200):**

```json
[
  {
    "id": 1,
    "name": "Tech Conference 2024",
    "description": "Annual technology conference",
    "capacity": 300,
    "available_slots": 275,
    "organizer_id": 1,
    "created_at": "2024-02-21T10:00:00Z"
  }
]
```

### Create Event

```http
POST /events
```

**Request body:**

```json
{
  "name": "Summer Music Festival",
  "description": "3-day outdoor music festival with top artists",
  "capacity": 5000,
  "organizer_id": 1
}
```

**Response (201):**

```json
{
  "id": 2,
  "name": "Summer Music Festival",
  "description": "3-day outdoor music festival with top artists",
  "capacity": 5000,
  "available_slots": 5000,
  "organizer_id": 1,
  "created_at": "2024-02-21T10:30:00Z"
}
```

### Get Event by ID

```http
GET /events/{id}
```

**Response (200):**

```json
{
  "id": 1,
  "name": "Tech Conference 2024",
  "description": "Annual technology conference",
  "capacity": 300,
  "available_slots": 275,
  "organizer_id": 1,
  "created_at": "2024-02-21T10:00:00Z"
}
```

### Register for Event

```http
POST /events/{id}/register
```

**Headers:**

```text
X-User-ID: 123
```

**Response (201):**

```json
{
  "id": 45,
  "user_id": 123,
  "event_id": 1,
  "created_at": "2024-02-21T10:45:00Z"
}
```

### List Event Registrations

```http
GET /events/{id}/registrations
```

**Response (200):**

```json
[
  {
    "registration_id": 45,
    "user_id": 123,
    "name": "John Doe",
    "email": "john@example.com",
    "registered_at": "2024-02-21T10:45:00Z"
  }
]
```

### Simulate Concurrent Registrations

```http
POST /events/{id}/simulate
```

**Response (200):**

```json
{
  "total_attempts": 20,
  "successful": 1,
  "failed": 19
}
```

### Error Responses

#### 404 Not Found

```json
{
  "error": "not_found",
  "detail": "event not found"
}
```

#### 409 Conflict (Event Full)

```json
{
  "error": "event_full",
  "detail": "event is full"
}
```

#### 401 Unauthorized (Missing User ID)

```json
{
  "error": "missing_user_id",
  "detail": "X-User-ID header is required"
}
```

## 🧪 Testing

### Automated Concurrency Test

```bash
go test ./internal/tests -v

# Expected:
# total=20, successful=1, failed=19
# PASS: TestConcurrentBooking
```

The test creates an event with capacity 1, spawns multiple goroutines attempting concurrent registration, and asserts that exactly one succeeds and `available_slots` becomes 0.

### Manual Testing (Thunder Client / Postman)

1. `GET /health` – verify server and DB connectivity.
2. `POST /events` – create an event.
3. `GET /events` – list events and confirm creation.
4. `POST /events/{id}/register` with `X-User-ID` – register users.
5. `POST /events/{id}/simulate` – observe concurrency behavior.
6. `GET /events/{id}/registrations` – inspect registrations.

## 🔧 Configuration

### Environment Variables

| Variable       | Description                         | Default                   |
|----------------|-------------------------------------|---------------------------|
| `DATABASE_URL` | Full PostgreSQL connection string  | –                         |
| `DB_HOST`      | Database host                      | `localhost`               |
| `DB_PORT`      | Database port                      | `5432`                    |
| `DB_USER`      | Database user                      | `postgres`                |
| `DB_PASSWORD`  | Database password                  | `postgres`                |
| `DB_NAME`      | Database name                      | `event_ticketing_system`  |
| `DB_SSLMODE`   | SSL mode                           | `disable`                 |
| `PORT`         | HTTP server port                   | `8080`                    |

### Logging Format

The application uses structured logging with key-value style messages, for example:

```text
msg=http_register_success user_id=123 event_id=45 registration_id=789
msg=http_register_failed_event_full user_id=124 event_id=45
msg=http_simulate_results event_id=1 total_attempts=20 successful=1 failed=19
```
