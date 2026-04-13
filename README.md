# TaskFlow

A minimal but complete task management REST API built with Go, PostgreSQL, and Docker.

## Overview

TaskFlow allows users to register, log in, create projects, and manage tasks within those projects. Tasks can be assigned to users, filtered by status or assignee, and managed with full CRUD operations.

### Tech Stack

- **Language:** Go 1.23
- **Router:** Chi v5
- **Database:** PostgreSQL 16
- **Migrations:** golang-migrate
- **Auth:** JWT (golang-jwt/v5) + bcrypt
- **Logging:** slog (stdlib, structured JSON)
- **Containers:** Docker with multi-stage builds

## Architecture Decisions

**Domain-based package structure** — Code is organized by domain (`auth/`, `project/`, `task/`) with each package following a handler → service → repository pattern. This keeps related code together and makes the codebase easy to navigate.

**Raw SQL over ORM** — Using `database/sql` with the pgx driver for full control over queries. No magic, no N+1 surprises, and the schema is small enough that the boilerplate is minimal.

**CHECK constraints over PostgreSQL ENUMs** — Status and priority use CHECK constraints rather than ENUM types. ENUMs require `ALTER TYPE` for changes and complicate migrations; CHECK constraints are just strings with validation.

**Partial update via map[string]json.RawMessage** — PATCH endpoints decode into a raw map to distinguish between "field not sent" (don't update), "field sent as null" (set to NULL), and "field sent with value" (update). This is correct REST PATCH semantics.

**DB trigger for updated_at** — A BEFORE UPDATE trigger on tasks auto-updates `updated_at`, keeping the application code clean and ensuring consistency.

**Separate migrate and seed containers** — Migrations run in a dedicated container that exits after completion. The API server only starts after migrations and seeding are done. This ensures a clean startup sequence.

**In-memory rate limiting** — Auth endpoints use a token bucket rate limiter (5 req/s, burst of 10) per IP. In-memory is appropriate for a single-instance deployment; a distributed setup would use Redis.

**Request ID middleware** — Every request gets a `X-Request-ID` header (generated or echoed from the client). This ID is included in all structured log entries for end-to-end tracing.

### What I intentionally left out

- No CORS configuration (not needed without a frontend)
- No input sanitization beyond validation (e.g., max length on strings)
- No database connection pooling tuning based on load testing
- No CI/CD pipeline with linting, tests, and Docker image publishing

## Running Locally

Prerequisites: Docker and Docker Compose.

```bash
git clone <repo-url>
cd taskflow
cp .env.example .env
docker compose up --build
# API available at http://localhost:8080
```

## Running Migrations

Migrations run automatically via the `migrate` container on `docker compose up`. No manual steps required.

To run migrations manually:

```bash
docker compose run --rm migrate -path /migrations -database "$DATABASE_URL" up
```

## Running Tests

Integration tests run against a real database inside Docker. No local Go installation required.

```bash
./scripts/test.sh
```

This spins up a separate test stack (`docker-compose.test.yml`), runs the full suite, and tears everything down. Tests cover:

- Auth flow (register, login, validation, duplicate detection)
- Auth middleware (missing/invalid tokens)
- Request ID propagation
- Project CRUD with pagination
- Project stats endpoint
- Task permission checks (owner vs non-owner)

## Test Credentials

Seed data is loaded automatically. Use these credentials to log in:

```
Email:    test@example.com
Password: password123
```

## API Reference

A Postman collection is included in `postman/` with requests for every endpoint.

### Authentication

Auth endpoints are rate-limited to 5 requests/second per IP (burst of 10).

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | Register with name, email, password |
| POST | `/auth/login` | Returns JWT token (24h expiry) |

### Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects` | List projects (owned or assigned) |
| POST | `/projects` | Create project |
| GET | `/projects/:id` | Get project with tasks |
| GET | `/projects/:id/stats` | Task counts by status and assignee |
| PATCH | `/projects/:id` | Update project (owner only) |
| DELETE | `/projects/:id` | Delete project (owner only) |

### Tasks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects/:id/tasks` | List tasks (filter: `?status=`, `?assignee=`) |
| POST | `/projects/:id/tasks` | Create task |
| PATCH | `/tasks/:id` | Update task |
| DELETE | `/tasks/:id` | Delete task (owner or creator only) |

All endpoints return `Content-Type: application/json`. Auth endpoints are public; all others require `Authorization: Bearer <token>`.

### Pagination

List endpoints (`GET /projects`, `GET /projects/:id/tasks`) support pagination:

```
?page=1&limit=20
```

- `page` defaults to 1, `limit` defaults to 20 (max 100)
- Response headers: `X-Total-Count`, `X-Page`, `X-Per-Page`, `X-Total-Pages`

### Project Stats

`GET /projects/:id/stats` returns:

```json
{
  "total_tasks": 3,
  "by_status": { "todo": 1, "in_progress": 1, "done": 1 },
  "by_assignee": [
    { "assignee_id": "uuid", "name": "Test User", "count": 2 },
    { "assignee_id": null, "name": null, "count": 1 }
  ]
}
```

### Request Tracing

Every response includes an `X-Request-ID` header. Send your own `X-Request-ID` to correlate requests across systems, or let the server generate one. The ID appears in all structured log entries.

### Error Responses

| Status | Body |
|--------|------|
| 400 | `{"error": "validation failed", "fields": {"email": "is required"}}` |
| 401 | `{"error": "unauthorized"}` |
| 403 | `{"error": "forbidden"}` |
| 404 | `{"error": "not found"}` |
| 429 | `{"error": "too many requests"}` |
