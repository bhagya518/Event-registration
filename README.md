# Event Registration & Ticketing System (Go + Gin + PostgreSQL)

## 1. Problem statement

This project is a REST API for event registration and ticketing (similar to Eventbrite).

Core requirements:

- Users can browse events.
- Organizers can create events with limited capacity.
- Users can register for events.
- The system must prevent **overbooking** when many users try to register concurrently for the last remaining spots.

The most important engineering challenge is **concurrency safety** at the database level.

---

## 2. Architecture explanation

This codebase follows a clean-architecture inspired layout:

- `cmd/server`
  - Application entrypoint (`main.go`) and HTTP server wiring.
- `internal/handlers`
  - Gin HTTP handlers (request parsing, response formatting, status codes).
  - Handlers do **not** contain business logic.
- `internal/services`
  - Business logic / use-cases.
  - Validations and orchestration live here.
- `internal/repository`
  - Database access using `database/sql`.
  - Contains the transaction-safe registration logic.
- `internal/database`
  - PostgreSQL connection setup and environment configuration.
- `internal/middleware`
  - Request middleware (e.g., extracting `X-User-ID`).

Dependency direction:

`handlers -> services -> repositories -> database`

This keeps business logic testable and isolates persistence concerns.

---

## 3. Database schema

Tables:

- `users`
- `events`
- `registrations`

Key constraints:

- `events.capacity > 0`
- `events.available_slots >= 0`
- `registrations` has a unique constraint: `UNIQUE(user_id, event_id)`

The production-ready PostgreSQL migration is located at:

- `migrations/001_create_tables.sql`

---

## 4. Race condition explanation

### The overbooking problem

If you implement registration as two separate steps without locking:

1. Read `available_slots`
2. If `available_slots > 0`, insert registration and decrement slots

Under concurrency, multiple requests can read the same `available_slots` value before any decrement occurs.

Example:

- `available_slots = 1`
- Two requests read `available_slots` at the same time and both see `1`
- Both insert registrations
- Both decrement

Result: **2 registrations for 1 slot** (overbooking).

---

## 5. Concurrency strategy (transaction + SELECT FOR UPDATE)

This project prevents overbooking using **database transactions** and **row-level locks**.

The critical query:

```sql
SELECT available_slots
FROM events
WHERE id = $1
FOR UPDATE;
```

Why this works:

- `FOR UPDATE` locks the selected `events` row.
- While the transaction is open, any concurrent transaction trying to lock the same event row must wait.
- This ensures that only one transaction at a time can:
  - check `available_slots`
  - insert a registration
  - decrement `available_slots`

Registration flow (simplified):

1. `BEGIN`
2. `SELECT ... FOR UPDATE`
3. If `available_slots > 0`:
   - `INSERT INTO registrations ...`
   - `UPDATE events SET available_slots = available_slots - 1`
   - `COMMIT`
4. Else:
   - `ROLLBACK`
   - return error `"event is full"`

Implementation lives in:

- `internal/repository/registration_repository.go`

---

## 6. Simulation endpoint explanation

Endpoint:

- `POST /events/:id/simulate`

Behavior:

- Spawns **100 goroutines** concurrently.
- Each goroutine attempts to register a *different user* for the same event.
- Uses `sync.WaitGroup` to wait for all attempts.
- Tracks success/failure counts via atomic counters.

Response:

```json
{
  "total_attempts": 100,
  "successful": 10,
  "failed": 90
}
```

This demonstrates that the system does not overbook:

- If event capacity is `C`, the number of successful registrations will not exceed `C`.

---

## 7. How to run locally

### Prerequisites

- Go 1.22+
- PostgreSQL 13+

### Configure environment

1. Create a `.env` file from the example:

```bash
cp .env.example .env
```

2. Update values if needed:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

### Start the server

From the project root:

```bash
go mod tidy
go run ./cmd/server
```

Health check:

```bash
curl http://localhost:8080/health
```

---

## 8. Example curl commands

### Create an event

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Go Conference",
    "description": "Conference about Go",
    "capacity": 10,
    "organizer_id": 1
  }'
```

### List events

```bash
curl http://localhost:8080/events
```

### Get event by ID

```bash
curl http://localhost:8080/events/1
```

### Register for an event

Registration reads user identity from the header:

```bash
curl -X POST http://localhost:8080/events/1/register \
  -H "X-User-ID: 123"
```

### Simulate concurrent registrations

```bash
curl -X POST http://localhost:8080/events/1/simulate
```

---

## 9. Future improvements

- Add proper migration runner and a `migrations/` workflow.
- Implement `GET /events/:id/registrations` fully (currently returns 501).
- Add authentication and authorization:
  - Only organizers can create events.
  - Only organizers can view registrations for their events.
- Add pagination and filtering for event listing.
- Add idempotency keys for registrations.
- Add structured logging with request IDs and latency metrics.
- Add integration tests and a dedicated concurrency stress test suite.

---

## Notes on logging

The API uses Go's standard `log` package with key-value style messages (e.g., `msg=... user_id=... event_id=...`) to make logs easy to filter/search.
