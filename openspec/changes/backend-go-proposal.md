# Proposal: Go Backend for IoT Fleet Monitoring

## Intent
Build a production-quality Go backend for an IoT fleet monitoring system that handles sensor data ingestion, real-time updates via WebSockets, manual JWT authentication, and predictive fuel alerts.

## Scope

### In Scope
- REST API with manual JWT authentication (HMAC-SHA256 + base64url, NO external JWT libraries)
- Sensor data ingestion endpoints (GPS, fuel, temperature)
- Predictive fuel autonomy calculation with <1 hour alert threshold
- WebSocket server for real-time sensor/location updates
- SQLite database (simpler setup for a 3-day test, easily swappable to PostgreSQL)
- Device ID masking for non-admin users (DEV-****-XC54)
- Role-based access control (admin vs regular user)
- Unit tests for critical logic (JWT validation, fuel calculation)
- Comprehensive documentation (DESIGN.md, SETUP.md)

### Out of Scope
- Frontend (separate proposal)
- Mobile app (separate proposal)
- Actual hardware/IoT device communication (we simulate with HTTP POSTs)
- Production deployment configuration (Docker, k8s, etc.)
- Advanced observability (metrics, tracing)

## Approach

### Stack Selection
- **Language**: Go 1.21+ (standard library rich, excellent concurrency)
- **Database**: SQLite via `database/sql` + `github.com/mattn/go-sqlite3` (zero-config, single file, perfect for a test)
- **WebSockets**: `github.com/gorilla/websocket` (battle-tested, de-facto standard)
- **Router**: `net/http` standard library with `http.ServeMux` (no need for frameworks, keeps it simple)
- **JSON**: `encoding/json` (standard library)
- **Crypto**: `crypto/hmac`, `crypto/sha256`, `encoding/base64` (for manual JWT)
- **Testing**: `testing` package + table-driven tests (Go idiomatic)
- **Migrations**: Simple SQL files executed on startup (no heavy migration tool)

### Architecture
Clean Architecture / Hexagonal approach with clear separation:
```
backend/
├── cmd/api/           # Application entry point
├── internal/
│   ├── domain/        # Business entities and interfaces (no external deps)
│   ├── application/   # Use cases / services (fuel calculation, auth)
│   ├── infrastructure/# DB, HTTP handlers, WebSocket, JWT implementation
│   └── config/        # Configuration loading
├── migrations/        # SQL schema files
└── docs/
    ├── DESIGN.md      # Stack choice rationale
    └── SETUP.md       # Local deployment guide
```

### Why This Stack?
1. **SQLite**: Zero setup, single file, ACID compliant. For a 3-day test, PostgreSQL adds ops overhead without benefit. Can be swapped later.
2. **Standard net/http**: No framework magic to learn. Go's stdlib is powerful enough.
3. **Clean Architecture**: Makes testing trivial (mock interfaces), shows architectural maturity.
4. **Manual JWT**: Demonstrates deep understanding of crypto primitives. Uses only stdlib.

### Key Decisions
- **Fuel Calculation**: Based on last N data points to estimate consumption rate, then project time-to-empty.
- **WebSocket**: One hub per server, broadcasts to all connected clients. Clients filter by device.
- **ID Masking**: Applied at the HTTP handler layer, not DB layer. Raw IDs only in DB.

## Risks
- **Go Learning Curve**: User is new to Go. Need to explain Go idioms as we go.
- **Manual JWT**: Easy to get wrong. Must test edge cases (expired, tampered, malformed).
- **Time Constraint**: 3 days total for backend + frontend + mobile + docs + video. Backend should be ~1 day.

## First Slice
1. Project bootstrap: go.mod, basic structure, SQLite connection
2. Domain models: Vehicle, SensorData, User
3. Manual JWT implementation with tests
4. Auth middleware
5. Sensor data ingestion endpoint
6. Fuel calculation logic with tests
7. WebSocket hub
8. Device ID masking middleware

## Success Criteria
- `go test ./...` passes
- `go run ./cmd/api` starts the server
- Can authenticate via JWT
- Can ingest sensor data
- WebSocket receives real-time updates
- Fuel alerts calculated correctly
- Non-admin sees masked device IDs
