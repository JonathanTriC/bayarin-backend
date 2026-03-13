# Bayar.in Backend

A production-ready multi-tenant POS (Point of Sale) SaaS backend built with Go.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-4169E1?style=flat&logo=postgresql)
![License](https://img.shields.io/badge/license-MIT-green?style=flat)

---

## Overview

Bayar.in is a multi-tenant POS backend that supports multiple businesses, branches, and staff roles out of the box. It is designed to be fast, secure, and scalable from day one.

**Key features:**
- Multi-tenant architecture — every query is scoped to `business_id`
- Role-based access control — `owner` and `cashier` roles with strict guards
- Atomic payment flow — totals recalculated and committed in a single DB transaction
- JWT authentication with session revocation
- Rate limiting on auth endpoints

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.21+ |
| Framework | Fiber v2 |
| Database | PostgreSQL 15 |
| Auth | JWT (HS256) + session table |
| Query layer | sqlc |
| Migrations | golang-migrate |
| Password hashing | bcrypt |
| UUID | google/uuid |
| Hot reload (dev) | Air |

---

## Project Structure

```text
bayarin-backend/
├── cmd/server/main.go          # Entry point, DI wiring, route groups
├── config/config.go            # .env loader
├── migrations/                 # SQL migration files
├── sqlc/                       # Raw SQL queries for sqlc
├── internal/
│   ├── db/                     # DB connection pool + sqlcgen
│   ├── middleware/             # AuthMiddleware, RoleGuard
│   ├── auth/                   # Register, login, logout, me
│   ├── business/               # Business profile
│   ├── branch/                 # Branch management
│   ├── staff/                  # Staff management
│   ├── menu/                   # Menu items
│   ├── modifier/               # Modifier groups & options
│   ├── table/                  # Table management
│   ├── order/                  # Order + order items
│   ├── payment/                # Payment flow
│   ├── dashboard/              # Owner & cashier dashboards
│   └── httputil/               # Shared HTTP utilities for responses
└── .air.toml                   # Air hot reload config
```

---

## API Documentation

Interactive Swagger documentation is available locally at:

```
http://localhost:8080/swagger/index.html
```

---

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+ (or a [Supabase](https://supabase.com) project)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI
- [sqlc](https://sqlc.dev) CLI
- [Air](https://github.com/air-verse/air) (optional, for hot reload)

Install CLI tools:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/air-verse/air@latest

# Add Go binaries to PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

### 1. Clone the repository

```bash
git clone https://github.com/JonathanTriC/bayarin-backend.git
cd bayarin-backend
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` with your values (see [Environment Variables](#environment-variables) below).

### 3. Run database migrations

```bash
export $(cat .env | xargs)
migrate -path ./migrations -database "$DATABASE_URL" up
```

### 4. Generate sqlc types

```bash
sqlc generate
```

### 5. Install dependencies

```bash
go mod tidy
```

### 6. Run the server

**With hot reload (recommended for development):**
```bash
air
```

**Without hot reload:**
```bash
go run ./cmd/server/main.go
```

The API will be available at `http://localhost:8080`.

### 7. Verify it's running

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Environment Variables

Create a `.env` file in the project root based on `.env.example`:

```dotenv
# PostgreSQL connection string
# For Supabase: use the Session Pooler URL with sslmode=require
DATABASE_URL=postgres://postgres:password@localhost:5432/bayarin?sslmode=disable

# JWT signing secret — use a long random string in production
JWT_SECRET=change-me-to-a-long-random-secret

# Port the server listens on
PORT=8080
```

| Variable | Description | Required |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | ✅ |
| `JWT_SECRET` | Secret key for signing JWT tokens | ✅ |
| `PORT` | Port the HTTP server listens on | ✅ |

> ⚠️ Never commit your `.env` file. It is already listed in `.gitignore`.

---

## Quick Smoke Test

```bash
# Register owner
curl -X POST http://localhost:8080/api/v1/auth/register-owner \
  -H "Content-Type: application/json" \
  -d '{"owner_name":"Jonathan","email":"owner@test.com","password":"secret123","business_name":"Warung Maju"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@test.com","password":"secret123"}'

# Set token
TOKEN="<token from login response>"

# Check dashboard
curl http://localhost:8080/api/v1/dashboard/owner \
  -H "Authorization: Bearer $TOKEN"
```

---

## License

MIT © [Jonathan Tri](https://jonathantri.com)
