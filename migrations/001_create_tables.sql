-- 001_create_tables.sql
-- Initial schema for Event Registration & Ticketing System

BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('USER', 'ORGANIZER')),
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    capacity INT NOT NULL CHECK (capacity > 0),
    available_slots INT NOT NULL CHECK (available_slots >= 0),
    organizer_id INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS registrations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    event_id INT REFERENCES events(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT registrations_user_event_unique UNIQUE (user_id, event_id)
);

-- Indexes for common access patterns
CREATE INDEX IF NOT EXISTS idx_registrations_event_id
    ON registrations (event_id);

CREATE INDEX IF NOT EXISTS idx_events_organizer_id
    ON events (organizer_id);

COMMIT;
