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

### What I intentionally left out

- No request rate limiting (would add for production)
- No request ID / correlation middleware (would add for observability)
- No pagination on list endpoints (documented as TODO)
- No CORS configuration (not needed without a frontend)

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

## Test Credentials

Seed data is loaded automatically. Use these credentials to log in:

```
Email:    test@example.com
Password: password123
```

## API Reference

A Bruno API collection is included in `api-collection/` with requests for every endpoint. Open it with [Bruno](https://www.usebruno.com/).

### Authentication

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

### Error Responses

| Status | Body |
|--------|------|
| 400 | `{"error": "validation failed", "fields": {"email": "is required"}}` |
| 401 | `{"error": "unauthorized"}` |
| 403 | `{"error": "forbidden"}` |
| 404 | `{"error": "not found"}` |

## What I'd Do With More Time

- **Pagination** on list endpoints (`?page=&limit=`) with total count in response headers
- **GET /projects/:id/stats** endpoint returning task counts grouped by status and assignee
- **Integration tests** — at least covering auth flow, project CRUD, and task permission checks
- **Rate limiting** on auth endpoints to prevent brute-force attacks
- **Request ID middleware** for distributed tracing and log correlation
- **Input sanitization** and stricter validation (e.g., max length on strings)
- **Database connection pooling tuning** based on load testing
- **CI/CD pipeline** with linting, tests, and Docker image publishing
