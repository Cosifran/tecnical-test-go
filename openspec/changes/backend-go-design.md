# Design: Go Backend for IoT Fleet Monitoring

## Technical Approach

Build a Clean Architecture Go backend with four layers (domain → application → infrastructure → cmd). The domain layer defines entities and interfaces with zero external dependencies. The application layer implements business logic (JWT, fuel calculation) against domain interfaces. The infrastructure layer provides concrete implementations (SQLite repos, HTTP handlers, WebSocket hub). Manual JWT uses only `crypto/hmac` + `crypto/sha256` + `encoding/base64`. SQLite stores all data in a single file. WebSockets broadcast via a goroutine-based Hub pattern.

---

## Architecture Overview

### High-Level Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                        cmd/api/main.go                       │
│                  (wires everything, starts server)            │
└──────────────┬───────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────────────────┐
│                    INFRASTRUCTURE LAYER                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────┐  │
│  │ HTTP     │  │ SQLite   │  │ WebSocket│  │ JWT Impl    │  │
│  │ Handlers │  │ Repo     │  │ Hub      │  │ (HMAC-S256) │  │
│  └──────────┘  └──────────┘  └──────────┘  └─────────────┘  │
└──────────────┬───────────────────────────────────────────────┘
               │ depends on
               ▼
┌──────────────────────────────────────────────────────────────┐
│                    APPLICATION LAYER                          │
│  ┌──────────┐  ┌──────────────┐  ┌──────────┐               │
│  │ AuthService│ │ FuelService  │  │ SensorService│            │
│  └──────────┘  └──────────────┘  └──────────┘               │
└──────────────┬───────────────────────────────────────────────┘
               │ depends on
               ▼
┌──────────────────────────────────────────────────────────────┐
│                    DOMAIN LAYER (no external deps)            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────┐  │
│  │ User     │  │ Vehicle  │  │SensorData│  │ Alert       │  │
│  │ Entity   │  │ Entity   │  │ Entity   │  │ Entity      │  │
│  └──────────┘  └──────────┘  └──────────┘  └─────────────┘  │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ Interfaces: UserRepository, VehicleRepository, etc.    │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Owns | Depends On | External Deps? |
|-------|------|-----------|----------------|
| **Domain** | Entities, interfaces | Nothing | NO |
| **Application** | Business logic, use cases | Domain interfaces | NO |
| **Infrastructure** | HTTP handlers, SQLite repos, WebSocket hub, JWT impl | Domain + Application | YES (sqlite3 driver, gorilla/websocket) |
| **Cmd** | Wiring, startup | All layers | YES |

### REST Data Flow

```
HTTP Request
  │
  ▼
Router (net/http ServeMux)
  │
  ▼
Middleware Chain (auth → RBAC → logging)
  │
  ▼
Handler (extracts params, calls service)
  │
  ▼
Service / Application (business logic)
  │
  ▼
Repository Interface (domain)
  │
  ▼
SQLite Repository (infrastructure impl)
  │
  ▼
database/sql → SQLite file
```

### WebSocket Data Flow

```
Client connects: ws://host/api/v1/ws?token=<jwt>
  │
  ▼
Auth Middleware (validates JWT from query param)
  │
  ▼
Hub.Register (new Client goroutine)
  │
  ├── Client.ReadPump (reads from WS, ignores — we only push)
  └── Client.WritePump (listens on Client.send channel)
         │
         ▼
Hub.Broadcast (when sensor data ingested)
  │
  ├── Client A.send ← JSON message
  └── Client B.send ← JSON message
```

---

## Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point: wire deps, start server
├── internal/
│   ├── domain/
│   │   ├── entity.go               # User, Vehicle, SensorData, Alert structs
│   │   ├── repository.go           # Interfaces: UserRepository, VehicleRepository, etc.
│   │   └── errors.go               # Domain-level error types (ErrNotFound, ErrUnauthorized)
│   ├── application/
│   │   ├── auth_service.go         # Login, token generation, validation logic
│   │   ├── fuel_service.go         # Fuel autonomy calculation, alert generation
│   │   ├── sensor_service.go       # Sensor data ingestion, validation, delegation
│   │   └── vehicle_service.go      # Vehicle CRUD
│   ├── infrastructure/
│   │   ├── http/
│   │   │   ├── router.go           # ServeMux route registration
│   │   │   ├── middleware.go       # AuthMiddleware, RBACMiddleware, LoggingMiddleware
│   │   │   ├── handler/
│   │   │   │   ├── auth_handler.go       # POST /auth/login, POST /auth/refresh
│   │   │   │   ├── sensor_handler.go     # POST /sensors/data
│   │   │   │   ├── vehicle_handler.go    # GET /vehicles, GET /vehicles/{id}/history
│   │   │   │   └── alert_handler.go      # GET /alerts
│   │   │   ├── request.go          # Request binding helpers
│   │   │   └── response.go         # JSON response helpers, error format
│   │   ├── persistence/
│   │   │   ├── sqlite/
│   │   │   │   ├── user_repo.go          # UserRepository implementation
│   │   │   │   ├── vehicle_repo.go       # VehicleRepository implementation
│   │   │   │   ├── sensor_repo.go        # SensorDataRepository implementation
│   │   │   │   └── alert_repo.go         # AlertRepository implementation
│   │   │   └── db.go               # Connection setup, migration runner
│   │   ├── websocket/
│   │   │   ├── hub.go              # Hub struct, Register/Unregister/Broadcast
│   │   │   └── client.go           # Client struct, ReadPump/WritePump goroutines
│   │   └── jwt/
│   │       └── token.go            # Manual JWT: Generate, Validate, ParseClaims
│   └── config/
│       └── config.go               # Env loading: JWT_SECRET, DB_PATH, PORT, etc.
├── migrations/
│   └── 001_init.sql                # CREATE TABLE statements
├── docs/
│   ├── DESIGN.md                   # Stack rationale (per AGENTS.md deliverable)
│   └── SETUP.md                    # Local deployment guide
├── go.mod
├── go.sum
└── .env.example                    # Template for environment variables
```

**Why this structure?**

- `internal/` is a Go convention: the compiler prevents external packages from importing it. This is our architectural boundary.
- `domain/` has ZERO import from `infrastructure/` — dependency inversion via interfaces ensures testability.
- Each handler file maps to one API resource group. Small files = easy to review.
- `migrations/` as raw SQL keeps things simple — no migration framework for a 3-day test.

---

## Component Design

### Auth / JWT: Manual Implementation

**How manual JWT works:**

A JWT token is three base64url-encoded segments separated by dots:

```
header.payload.signature
```

**Token structure:**

```
┌─────────────────────────────────────────────────┐
│ Header (JSON → base64url)                        │
│ { "alg": "HS256", "typ": "JWT" }                │
├─────────────────────────────────────────────────┤
│ Payload (JSON → base64url)                       │
│ { "sub": "user-uuid",                            │
│   "email": "user@example.com",                   │
│   "role": "admin",                               │
│   "iat": 1700000000,                             │
│   "exp": 1700000900 }  ← 15 min after iat       │
├─────────────────────────────────────────────────┤
│ Signature                                        │
│ HMAC-SHA256(                                     │
│   base64url(header) + "." + base64url(payload),  │
│   secret_key                                     │
│ ) → base64url(result)                            │
└─────────────────────────────────────────────────┘
```

**Validation flow:**

```
Token arrives in Authorization: Bearer <token>
  │
  ▼
Split by "." → must have 3 parts
  │
  ▼
Recompute: HMAC-SHA256(header + "." + payload, secret)
  │
  ▼
Compare signature (constant-time) with token's 3rd segment
  │
  ├─ Mismatch → 401 "invalid_token"
  │
  ▼ (match)
Decode payload JSON
  │
  ▼
Check exp > time.Now()
  │
  ├─ Expired → 401 "token_expired"
  │
  ▼ (valid)
Extract sub, email, role → inject into request context
```

**Critical implementation details:**

- Use `crypto/hmac` with `crypto/sha256` for signing — NO external JWT library.
- Use `encoding/base64` with `base64.URLEncoding.WithPadding(base64.NoPadding)` for base64url (JWT spec requires no padding).
- Use `hmac.Equal()` for constant-time signature comparison — prevents timing attacks.
- Secret MUST be ≥32 bytes (256 bits), loaded from env var `JWT_SECRET`.
- Access token: 15 min `exp`. Refresh token: 7 day `exp`, claim `type: "refresh"`.

### Sensor Ingestion

**Request format:**

```json
POST /api/v1/sensors/data
Authorization: Bearer <token>
Content-Type: application/json

[
  {
    "device_id": "DEV-12345678-ABCD",
    "timestamp": "2026-06-16T10:30:00Z",
    "type": "fuel",
    "value": { "level": 45.2, "unit": "liters" }
  },
  {
    "device_id": "DEV-12345678-ABCD",
    "timestamp": "2026-06-16T10:30:00Z",
    "type": "gps",
    "value": { "lat": -34.6037, "lng": -58.3816 }
  }
]
```

**Processing flow:**

```
Handler receives batch
  │
  ▼
Validate all items (schema + timestamp ≤ now)
  │
  ├─ Any invalid → 400, reject ENTIRE batch (atomicity)
  │
  ▼ (all valid)
Resolve device_id → vehicle_id (lookup in DB)
  │
  ▼
Service.SensorData.Ingest(data [])
  │
  ├── Persist all points (single transaction)
  └── For each fuel point → FuelService.CheckAutonomy(vehicle_id)
  │
  ▼
Hub.Broadcast(sensor_data_message)
  │
  ▼
201 Created { "inserted": 2 }
```

**Validation rules:**

- `device_id`: non-empty string, must exist in `vehicles` table
- `timestamp`: valid RFC 3339 / ISO 8601, must not be in the future (tolerance: ±30s clock skew)
- `type`: one of `gps`, `fuel`, `temperature`
- `value`: validated against type-specific schema (GPS needs lat+lng, fuel needs level+unit, temp needs celsius)
- Batch size: max 100 items

### Fuel Calculation

This is the **most critical business logic** — it directly maps to the spec requirement of alerting when fuel autonomy < 1 hour.

**Algorithm:**

```
Input: vehicle_id (after a new fuel reading is ingested)

1. Fetch last N fuel readings for this vehicle (N=10, ordered by timestamp DESC)
   └── If count < 3 → SKIP (insufficient data, no projection possible)

2. Calculate consumption rate:
   ┌────────────────────────────────────────────────────────┐
   │ rate = (oldest_level - newest_level) /                 │
   │        (newest_timestamp - oldest_timestamp) in hours  │
   │                                                        │
   │ If rate ≤ 0 → fuel is not decreasing, no alert needed  │
   └────────────────────────────────────────────────────────┘

3. Project time-to-empty:
   ┌────────────────────────────────────────────────────────┐
   │ autonomy_hours = newest_level / rate                   │
   │                                                        │
   │ If autonomy_hours < 1.0 → GENERATE low_fuel alert      │
   └────────────────────────────────────────────────────────┘
```

**Why this approach?**

- Using the oldest and newest of the last N points gives a **linear regression approximation** without the complexity of actual least-squares. For a 3-day test, this is the right balance of correctness vs complexity.
- N=10 gives enough data to smooth sensor noise without making the window too wide.
- Minimum of 3 points prevents wild projections from too-little data.

**Edge cases:**

| Edge Case | Handling |
|-----------|----------|
| < 3 readings | Skip calculation, no alert |
| Rate ≤ 0 (refueled or stationary) | No alert — fuel is increasing or stable |
| Rate is NaN/Inf | Guard with `math.IsNaN` / `math.IsInf`, skip |
| Sensor spikes (bad data) | Linear regression over 10 points smooths outliers |
| Fuel level jumps up (refuel) | Oldest → newest may show negative rate; skip |
| Concurrent ingestions for same vehicle | SQLite WAL mode serializes writes; no race condition |

**Alert creation:**

```
If autonomy < 1 hour:
  → Create Alert record in DB (type: "low_fuel", severity: "critical", vehicle_id, details)
  → Broadcast alert via WebSocket hub
```

### WebSocket Hub

**Hub pattern (Go idiomatic):**

```
┌─────────────────────────────────────────┐
│                  Hub                      │
│                                           │
│  register   chan *Client  ───┐            │
│  unregister chan *Client  ───┤            │
│  broadcast  chan []byte   ───┤            │
│                                           │
│  clients map[*Client]bool                 │
│                                           │
│  ┌──── Run() goroutine ─────────────┐    │
│  │ for {                             │    │
│  │   select {                        │    │
│  │   case c := <-register:           │    │
│  │     clients[c] = true             │    │
│  │   case c := <-unregister:         │    │
│  │     delete(clients, c)            │    │
│  │     close(c.send)                 │    │
│  │   case msg := <-broadcast:        │    │
│  │     for c := range clients {      │    │
│  │       c.send <- msg               │    │
│  │     }                             │    │
│  │   }                               │    │
│  │ }                                  │    │
│  └───────────────────────────────────┘    │
└─────────────────────────────────────────┘
         │
         │ registers
         ▼
┌─────────────────────────────────────────┐
│               Client                      │
│                                           │
│  hub   *Hub                               │
│  conn  *websocket.Conn                    │
│  send  chan []byte    ← Hub pushes here   │
│                                           │
│  ┌── WritePump goroutine ────────────   │
│  │ for msg := range c.send {         │   │
│  │   c.conn.WriteMessage(Text, msg)  │   │
│  │ }                                  │   │
│  │ (on channel close → close conn)   │   │
│  └───────────────────────────────────┘   │
│                                           │
│  ┌── ReadPump goroutine ─────────────┐   │
│  │ Reads from conn (keepsalive)      │   │
│  │ On error → hub.unregister         │   │
│  └───────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

**Key design decisions:**

- Hub runs a **single goroutine** (`Run()`) that owns the `clients` map — no mutex needed for the map itself.
- Each Client has **two goroutines**: ReadPump (keeps connection alive, detects disconnect) and WritePump (sends messages from `send` channel).
- If a WritePump fails, the client is unregistered and its connection closed — other clients are unaffected (per spec).
- Broadcast is fire-and-forget: if a client's `send` channel is full (buffered with size 256), the message is dropped for that client rather than blocking the hub.

### ID Masking

**Where it happens:** HTTP handler / response serialization layer. The domain and persistence layers always work with raw IDs.

**How it works:**

```go
// Pseudocode — in response.go or a helper
func MaskDeviceID(raw string) string {
    if len(raw) < 4 {
        return "DEV-****-????" // safety for malformed IDs
    }
    last4 := raw[len(raw)-4:]
    return "DEV-****-" + last4
}
```

**Flow:**

```
Handler prepares response with raw device_id
  │
  ▼
Check user role from context
  │
  ├── role == "admin" → serialize as-is
  └── role != "admin" → apply MaskDeviceID() before JSON encoding
```

**Why handler layer, not DB layer?**

- DB always stores the real ID — queries, joins, and lookups need it.
- Masking is a **presentation concern**: the same entity is shown differently depending on who asks.
- A handler-level response DTO pattern keeps the domain clean and makes masking testable in isolation.

### RBAC (Role-Based Access Control)

**Middleware chain:**

```
Request → AuthMiddleware → RBACMiddleware(requiredRole) → Handler
```

**AuthMiddleware:**

1. Extract `Authorization: Bearer <token>` header (or `?token=` for WebSocket)
2. Validate token signature and expiration
3. Parse claims → inject `{sub, email, role}` into `context.Context`
4. If invalid → 401, stop chain

**RBACMiddleware:**

```go
// Pseudocode
func RBACMiddleware(allowedRoles ...string) Middleware {
    return func(next Handler) Handler {
        return func(w, r) {
            role := r.Context().Value(ctxKeyRole).(string)
            if !contains(allowedRoles, role) {
                writeError(w, 403, "forbidden", "Insufficient permissions")
                return
            }
            next(w, r)
        }
    }
}
```

**Endpoint permissions:**

| Endpoint | Allowed Roles |
|----------|--------------|
| `POST /auth/login` | Public |
| `POST /auth/refresh` | Public (valid refresh token) |
| `POST /sensors/data` | admin, user |
| `GET /vehicles` | admin, user (masking applied for user) |
| `GET /vehicles/{id}/history` | admin, user (masking applied for user) |
| `GET /alerts` | admin only (returns empty for user at handler level) |
| `WS /ws` | admin, user |

**Alert visibility note:** Rather than returning 403 for regular users on `GET /alerts`, the spec says to return `200 OK` with empty array `[]`. This is handled in the alert handler: if role != admin, return empty list immediately.

---

## Database Design

### Schema (SQLite)

```sql
-- migrations/001_init.sql

CREATE TABLE IF NOT EXISTS users (
    id         TEXT PRIMARY KEY,          -- UUID as text
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,             -- bcrypt hash
    role       TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))  -- ISO 8601 string
);

CREATE TABLE IF NOT EXISTS vehicles (
    id         TEXT PRIMARY KEY,
    device_id  TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sensor_data (
    id         TEXT PRIMARY KEY,
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
    type       TEXT NOT NULL CHECK (type IN ('gps', 'fuel', 'temperature')),
    value      TEXT NOT NULL,             -- JSON stored as text
    timestamp  TEXT NOT NULL,             -- ISO 8601 from device
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS alerts (
    id         TEXT PRIMARY KEY,
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
    type       TEXT NOT NULL,             -- e.g., 'low_fuel'
    severity   TEXT NOT NULL DEFAULT 'critical',
    details    TEXT,                      -- JSON with calculation details
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_sensor_data_vehicle_timestamp
    ON sensor_data(vehicle_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_sensor_data_vehicle_type_timestamp
    ON sensor_data(vehicle_id, type, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_vehicle
    ON alerts(vehicle_id);

CREATE INDEX IF NOT EXISTS idx_alerts_type
    ON alerts(type);
```

**Why TEXT for UUIDs and timestamps?**

- SQLite has no native UUID or timestamp type. Storing as TEXT keeps values human-readable in the SQLite CLI (helpful for debugging during a test).
- ISO 8601 strings sort lexicographically = chronologically, so range queries on `timestamp` work correctly with the index.

**Why `value` as TEXT (not a separate table per sensor type)?**

- Sensor types have different schemas (GPS vs fuel vs temperature). Normalizing into separate tables adds complexity with no benefit for a 3-day test.
- SQLite supports JSON functions (`json_extract`) for future querying, but we don't need them now.
- Application layer validates and deserializes the JSON — the DB just stores it.

---

## API Design

### REST Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/auth/login` | No | Login, get tokens |
| `POST` | `/api/v1/auth/refresh` | No | Refresh access token |
| `POST` | `/api/v1/sensors/data` | Yes | Ingest sensor batch |
| `GET` | `/api/v1/vehicles` | Yes | List vehicles |
| `GET` | `/api/v1/vehicles/{id}/history` | Yes | Historical sensor data |
| `GET` | `/api/v1/alerts` | Yes (admin) | List alerts |
| `WS` | `/api/v1/ws` | Yes | Real-time updates |

### Endpoint Details

#### POST /api/v1/auth/login

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "securepass123"
}
```

**Success (200):**
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Invalid credentials (401):**
```json
{
  "error": "invalid_credentials",
  "message": "Email or password is incorrect"
}
```

#### POST /api/v1/auth/refresh

**Request:**
```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

**Success (200):**
```json
{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

#### POST /api/v1/sensors/data

**Request:** (see Sensor Ingestion section above)

**Success (201):**
```json
{
  "inserted": 2
}
```

**Validation error (400):**
```json
{
  "error": "validation_failed",
  "message": "One or more data points failed validation",
  "details": [
    { "index": 1, "field": "type", "message": "must be one of: gps, fuel, temperature" }
  ]
}
```

#### GET /api/v1/vehicles

**Success (200) — Admin:**
```json
{
  "vehicles": [
    {
      "id": "a1b2c3d4-...",
      "device_id": "DEV-12345678-ABCD",
      "name": "Truck 01",
      "created_at": "2026-06-16T10:00:00Z"
    }
  ]
}
```

**Success (200) — Regular user:**
```json
{
  "vehicles": [
    {
      "id": "a1b2c3d4-...",
      "device_id": "DEV-****-ABCD",
      "name": "Truck 01",
      "created_at": "2026-06-16T10:00:00Z"
    }
  ]
}
```

#### GET /api/v1/vehicles/{id}/history?type=fuel&from=2026-06-15T00:00:00Z&to=2026-06-16T23:59:59Z

**Success (200):**
```json
{
  "vehicle_id": "a1b2c3d4-...",
  "data": [
    {
      "id": "s1-uuid",
      "type": "fuel",
      "value": { "level": 45.2, "unit": "liters" },
      "timestamp": "2026-06-16T10:30:00Z"
    }
  ]
}
```

**Query parameters:**

| Param | Required | Description |
|-------|----------|-------------|
| `from` | No | Start of time range (ISO 8601) |
| `to` | No | End of time range (ISO 8601) |
| `type` | No | Filter by sensor type |

#### GET /api/v1/alerts

**Admin (200):**
```json
{
  "alerts": [
    {
      "id": "alert-uuid",
      "vehicle_id": "a1b2c3d4-...",
      "type": "low_fuel",
      "severity": "critical",
      "details": { "autonomy_hours": 0.8, "fuel_level": 4.0, "consumption_rate": 5.0 },
      "created_at": "2026-06-16T10:35:00Z"
    }
  ]
}
```

**Regular user (200):**
```json
{
  "alerts": []
}
```

### WebSocket Messages

**Connection:** `ws://host/api/v1/ws?token=<valid_jwt>`

**Server → Client message (sensor data):**
```json
{
  "event": "sensor_data",
  "data": {
    "device_id": "DEV-12345678-ABCD",
    "type": "fuel",
    "value": { "level": 45.2, "unit": "liters" },
    "timestamp": "2026-06-16T10:30:00Z"
  }
}
```

**Server → Client message (alert):**
```json
{
  "event": "alert",
  "data": {
    "vehicle_id": "a1b2c3d4-...",
    "type": "low_fuel",
    "severity": "critical",
    "details": { "autonomy_hours": 0.8 }
  }
}
```

### Standard Error Response Format

All errors use this structure:

```json
{
  "error": "<machine_readable_code>",
  "message": "<human readable description>"
}
```

| Error Code | HTTP Status | When |
|-----------|-------------|------|
| `invalid_credentials` | 401 | Wrong email/password |
| `invalid_token` | 401 | Token signature invalid or malformed |
| `token_expired` | 401 | Token past exp claim |
| `forbidden` | 403 | Role lacks permission |
| `not_found` | 404 | Resource doesn't exist |
| `validation_failed` | 400 | Request body fails schema check |
| `invalid_timestamp` | 400 | Future timestamp in sensor data |
| `method_not_allowed` | 405 | Wrong HTTP method |
| `internal_error` | 500 | Unexpected server error |

---

## Testing Strategy

### Unit Tests

| Layer | What | How |
|-------|------|-----|
| **Domain** | Entity field validation | Direct struct tests (minimal — entities are mostly data) |
| **Application — JWT** | Token generation, signature validation, expiration, tampered/malformed rejection | Table-driven tests with fixed clock. Generate tokens, validate them, assert outcomes. |
| **Application — Fuel** | Consumption rate calc, autonomy projection, threshold detection, insufficient data, negative rate | Table-driven tests with deterministic fuel reading sequences. Mock clock for time.Now(). |
| **Application — Masking** | `MaskDeviceID()` output format, last-4 preservation, edge cases (short IDs) | Table-driven tests with input → expected output |
| **Application — RBAC** | Middleware allows/denies based on role in context | Create mock request with role in context, call middleware, assert status code |

### JWT Test Patterns

```go
// Pseudocode — table-driven test structure
tests := []struct {
    name        string
    token       string
    secret      string
    now         time.Time
    wantErr     string  // expected error code, empty if valid
    wantSubject string
}{
    {"valid token", validToken, secret, now, "", "user-1"},
    {"expired token", expiredToken, secret, future, "token_expired", ""},
    {"tampered payload", tamperedToken, secret, now, "invalid_token", ""},
    {"wrong secret", validToken, "wrong", now, "invalid_token", ""},
    {"malformed (2 parts)", "a.b", secret, now, "invalid_token", ""},
}
```

Key: use an injected `clockFunc func() time.Time` instead of `time.Now()` so tests control time.

### Fuel Calculation Test Patterns

```go
tests := []struct {
    name           string
    readings       []FuelReading  // pre-seeded, deterministic
    wantAlert      bool
    wantAutonomy   float64
}{
    {
        "2 readings only → no alert (insufficient)",
        []FuelReading{{50, T-2h}, {48, T}},
        false, 0,
    },
    {
        "3 readings, rate 5L/h, level 4L → alert (0.8h < 1h)",
        []FuelReading{{14, T-2h}, {9, T-1h}, {4, T}},
        true, 0.8,
    },
    {
        "3 readings, rate 2L/h, level 10L → no alert (5h > 1h)",
        []FuelReading{{14, T-2h}, {12, T-1h}, {10, T}},
        false, 5.0,
    },
    {
        "rate is 0 (refueled) → no alert",
        []FuelReading{{5, T-2h}, {8, T-1h}, {10, T}},
        false, 0,
    },
}
```

### Integration Tests

| Flow | Approach |
|------|----------|
| **Full auth flow** | Start test server with in-memory SQLite. POST login → get token → GET protected endpoint → POST refresh → GET with new token. |
| **Sensor ingest → Alert** | Seed vehicle. POST sensor batch with fuel data crossing threshold. GET alerts as admin → verify alert present. GET alerts as user → verify empty. |
| **WebSocket connection** | Connect WS client with valid token. POST sensor data. Verify WS client receives broadcast message. |

Integration tests use `:memory:` SQLite (no file on disk) and a real `net/http.Server` on a random port.

---

## Key Decisions & Trade-offs

| Decision | Choice | Rejected | Rationale |
|----------|--------|----------|-----------|
| **Database** | SQLite | PostgreSQL | Zero-config, single file, ACID. For 3-day test, no ops overhead. Swap to Postgres later by changing repo impl only. |
| **HTTP framework** | `net/http` stdlib + `ServeMux` | Gin, Echo, Chi | No framework magic. Go's stdlib is production-grade. Less deps = less surprises. Shows language mastery. |
| **Architecture** | Clean / Hexagonal | Flat MVC | Interfaces at domain boundary = trivial mocking. Proves architectural maturity (evaluation criteria: 30% code quality). |
| **JWT** | Manual (HMAC-SHA256) | golang-jwt, jose | Explicit project requirement. Demonstrates crypto understanding. Higher risk of edge-case bugs → mitigated by thorough tests. |
| **WebSocket** | gorilla/websocket | nhooyr.io/websocket, stdlib | De-facto Go standard. Well-documented. nhooyr is good but gorilla has more examples and community support. |
| **UUID** | `crypto/rand` + hex encoding | google/uuid, github.com/satori/uuid | Minimize dependencies. 16 random bytes → hex string is trivial and sufficient. |
| **Config** | `.env` file + `os.Getenv` | Viper, envconfig | One struct, one loader. No framework for 4 env vars. |
| **Password hashing** | `golang.org/x/crypto/bcrypt` | argon2id | bcrypt is explicitly required (cost ≥ 10). `x/crypto` is semi-stdlib, maintained by Go team. |

**Manual JWT trade-offs:**

- **Pro**: Satisfies requirement. Demonstrates understanding. No dependency.
- **Con**: Must handle base64url padding, constant-time comparison, edge cases ourselves.
- **Mitigation**: Comprehensive table-driven tests covering all failure modes.

---

## Go-Specific Notes for Beginners

### What is a `struct`?

A struct is Go's way of grouping related data — think of it like a class that holds fields but no methods (methods are added separately):

```go
type Vehicle struct {
    ID       string
    DeviceID string
    Name     string
}
```

You create one: `v := Vehicle{ID: "abc", DeviceID: "DEV-123", Name: "Truck"}`

### What is an `interface`?

An interface defines a **contract** — a set of methods something must implement. Any type that has those methods automatically satisfies the interface (no `implements` keyword needed):

```go
type VehicleRepository interface {
    FindByID(id string) (*Vehicle, error)
    FindAll() ([]Vehicle, error)
}
```

Any struct with a `FindByID` and `FindAll` method matching these signatures IS a `VehicleRepository`. This is how we swap SQLite for Postgres later — just write a new struct that satisfies the interface.

### What is a `goroutine`?

A goroutine is a lightweight thread managed by the Go runtime (not the OS). Start one with `go`:

```go
go hub.Run()  // runs concurrently, doesn't block the caller
```

Goroutines are cheap — you can have thousands. The WebSocket hub uses goroutines for each client's read/write pumps.

### What is a `channel`?

Channels are how goroutines communicate safely. Think of them as a pipe: one goroutine sends, another receives:

```go
ch := make(chan []byte)   // create a channel that carries byte slices
ch <- someData            // send data (blocks if channel is full)
msg := <-ch               // receive data (blocks if channel is empty)
```

The WebSocket hub uses channels for `register`, `unregister`, and `broadcast`.

### What is `go mod`?

`go mod` is Go's dependency manager. `go.mod` declares your module and its dependencies (like `package.json` for Node):

```
module github.com/yourname/fleet-monitor

go 1.21

require (
    github.com/gorilla/websocket v1.5.1
    github.com/mattn/go-sqlite3 v1.14.22
    golang.org/x/crypto v0.23.0
)
```

Run `go mod tidy` to clean up. Run `go mod download` to fetch dependencies.

### Go Project Layout Conventions

- `cmd/` — entry points (main packages)
- `internal/` — private application code (Go compiler enforces this)
- `pkg/` — public libraries (if any) — we don't use this
- `migrations/` — SQL files
- One package per directory, package name = directory name
- Tests live alongside code: `user_repo.go` → `user_repo_test.go`

---

## Open Questions

- [ ] Should refresh token rotation be implemented (invalidate old refresh on use)? Spec says old remains valid until expiration — confirm this is acceptable.
- [ ] Should the history endpoint return masked device_id for regular users? The spec says masking applies to "vehicle list" — clarify if history also masks.
- [ ] Max WebSocket message size — what limit? Suggest 4096 bytes for inbound (we only push), 64KB for outbound.
