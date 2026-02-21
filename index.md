# Event Registration & Ticketing System

A production-ready REST API for event registration and ticketing with advanced concurrency control, built with Go, Gin framework, and PostgreSQL.

## 🎯 Project Highlights

- **🔒 Concurrency Control**: Database-level locking prevents overbooking
- **🏗️ Clean Architecture**: Separation of concerns with Go best practices
- **📊 Real-time Management**: Track available slots and registrations
- **🧪 Automated Testing**: Built-in concurrency simulation
- **🛡️ Production Ready**: Comprehensive error handling and logging

## 📚 Documentation

- **[API Documentation](https://github.com/bhagya518/Event-registration/blob/main/README.md)** - Complete API reference
- **[Concurrency Strategy](https://github.com/bhagya518/Event-registration/blob/main/docs/concurrency_strategy.md)** - Technical deep dive
- **[Source Code](https://github.com/bhagya518/Event-registration)** - Full repository

## 🚀 Quick Start

```bash
git clone https://github.com/bhagya518/Event-registration.git
cd Event-registration
go mod tidy
cp .env.example .env
# Edit .env with your database credentials
go run ./cmd/server
```

## 📖 API Endpoints

| Method | Endpoint | Description |
|---------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/events` | List all events |
| POST | `/events` | Create new event |
| GET | `/events/{id}` | Get event details |
| POST | `/events/{id}/register` | Register for event |
| GET | `/events/{id}/registrations` | List event registrations |
| POST | `/events/{id}/simulate` | Test concurrency |

## 🧪 Testing

```bash
# Run automated concurrency test
go test ./internal/tests -v

# Expected: 1 success, 19 failures (20 concurrent users, 1 slot available)
```

## 🏆 Key Features Demonstrated

- **Database Transactions**: Atomic operations with rollback
- **Row-Level Locking**: `SELECT FOR UPDATE` prevents race conditions
- **Middleware**: User identification via `X-User-ID` header
- **Error Handling**: Structured JSON responses with proper HTTP codes
- **Logging**: Key-value structured logging for observability

---

**Backend API built with ❤️ using Go, Gin, and PostgreSQL**
