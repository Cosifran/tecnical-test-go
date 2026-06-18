# Tasks: Go Backend Implementation

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 2800–3400 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: foundation+domain+JWT → PR 2: repos+handlers → PR 3: WebSocket+wiring → PR 4: docs+simulation |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Domain layer, JWT, fuel calc, config | PR 1 (base: main) | No external deps; fully testable standalone |
| 2 | SQLite repos, HTTP handlers, middleware, masking | PR 2 (base: PR 1) | Depends on domain + services |
| 3 | WebSocket hub+client, main.go wiring, seed data | PR 3 (base: PR 2) | Final integration piece |
| 4 | Simulation script, DESIGN.md, SETUP.md | PR 4 (base: PR 3) | Docs + dev tooling |

---

## Phase 1: Bootstrap & Infrastructure

- [x] 1.1 Initialize Go module: `go mod init`, add `gorilla/websocket`, `mattn/go-sqlite3`, `golang.org/x/crypto/bcrypt`. Verify `go mod tidy` succeeds. **AC:** `go build ./...` compiles. **Effort:** S
- [x] 1.2 Create project structure: `cmd/api/`, `internal/{domain,application,infrastructure,config}/`, `migrations/`, `docs/`. **AC:** All directories exist with placeholder `.gitkeep`. **Effort:** S
- [x] 1.3 Create `internal/config/config.go`: load `JWT_SECRET`, `DB_PATH`, `PORT`, `BCRYPT_COST` from `.env` via `os.Getenv`. Validate JWT_SECRET ≥ 32 bytes on load. **AC:** `config.Load()` returns populated struct or error for missing/invalid secret. **Effort:** S
- [x] 1.4 Write `migrations/001_init.sql`: CREATE TABLE users, vehicles, sensor_data, alerts with indexes (as per design §Database Design). **AC:** SQL parses without errors; all constraints match spec data models. **Effort:** S
- [x] 1.5 Create `internal/infrastructure/persistence/db.go`: open SQLite with WAL mode, run migrations on startup, expose `*sql.DB`. **AC:** In-memory `:memory:` DB opens and runs migrations; WAL pragma set. **Effort:** M

---

## Phase 2: Domain Layer (zero external deps)

- [x] 2.1 Write `internal/domain/entity.go`: User, Vehicle, SensorData, Alert structs with JSON tags matching spec data models. **AC:** Structs compile with correct types; `SensorData.Value` is `json.RawMessage`. **Effort:** S
- [x] 2.2 Write `internal/domain/repository.go`: interfaces — `UserRepository` (FindByEmail, Create), `VehicleRepository` (FindByID, FindAll, FindByDeviceID, Create), `SensorDataRepository` (FindByVehicleID, BulkInsert), `AlertRepository` (Create, FindAll). **AC:** All method signatures match what services will call. **Effort:** M
- [x] 2.3 Write `internal/domain/errors.go`: sentinel errors — `ErrNotFound`, `ErrUnauthorized`, `ErrValidation`, `ErrConflict`. **AC:** `errors.Is(err, domain.ErrNotFound)` works. **Effort:** S

---

## Phase 3: Application Layer — Core Business Logic

- [x] 3.1 **🔒 SECURITY-CRITICAL** — `internal/infrastructure/jwt/token.go`: manual JWT implementation using `crypto/hmac` + `crypto/sha256` + `encoding/base64`. Implement `Generate(claims, secret)`, `Validate(token, secret)`, `ParseClaims(token, secret)`. Use `base64.RawURLEncoding`, `hmac.Equal` for constant-time comparison. Access tokens: 15 min; refresh tokens: 7 days with `type: "refresh"` claim. Inject `now func() time.Time` for testability. **AC:** Round-trip: generate → validate returns claims; tampered signature fails with distinct error; expired token fails; malformed (≠3 segments) fails. **Effort:** L
- [x] 3.2 **🔒 SECURITY-CRITICAL** — `internal/application/auth_service.go`: `Login(email, password)` — bcrypt verify, generate access+refresh tokens. `Refresh(refreshToken)` — validate refresh token, ISSUE NEW refresh token AND INVALIDATE the old one (rotation per user clarification). **AC:** Login with valid creds returns both tokens; wrong password → `ErrUnauthorized`; refresh with valid token returns new access+refresh; reusing old refresh → `ErrUnauthorized`. **Effort:** L
- [ ] 3.3 — `internal/infrastructure/jwt/token.go`: manual JWT implementation using `crypto/hmac` + `crypto/sha256` + `encoding/base64`. Implement `Generate(claims, secret)`, `Validate(token, secret)`, `ParseClaims(token, secret)`. Use `base64.RawURLEncoding`, `hmac.Equal` for constant-time comparison. Access tokens: 15 min; refresh tokens: 7 days with `type: "refresh"` claim. Inject `now func() time.Time` for testability. **AC:** Round-trip: generate → validate returns claims; tampered signature fails with distinct error; expired token fails; malformed (≠3 segments) fails. **Effort:** L
- [ ] 3.2 **🔒 SECURITY-CRITICAL** — `internal/application/auth_service.go`: `Login(email, password)` — bcrypt verify, generate access+refresh tokens. `Refresh(refreshToken)` — validate refresh token, ISSUE NEW refresh token AND INVALIDATE the old one (rotation per user clarification). **AC:** Login with valid creds returns both tokens; wrong password → `ErrUnauthorized`; refresh with valid token returns new access+refresh; reusing old refresh → `ErrUnauthorized`. **Effort:** L
- [ ] 3.3 **⚠️ LOGIC-CRITICAL** — `internal/application/fuel_service.go`: `CheckAutonomy(vehicleID)` — fetch last N=10 fuel readings, compute consumption rate = (oldest_level − newest_level) / Δt_hours, project autonomy = newest_level / rate. Alert if autonomy < 1 hour. Guard: < 3 readings → skip; rate ≤ 0 → no alert; NaN/Inf → skip. **AC:** See spec scenarios §Predictive Fuel Autonomy: 3 readings with 5L/h rate + 4L level → alert (0.8h); 2L/h + 10L → no alert (5h); < 3 readings → no calculation. **Effort:** M
- [ ] 3.4 `internal/application/sensor_service.go`: `IngestBatch(points []SensorInput)` — validate all points (device_id exists, timestamp ≤ now+30s, type enum, value schema per type). Atomicity: reject entire batch if any invalid. Persist via repo, broadcast via WebSocket, call fuel check per fuel point. **AC:** Valid batch → 201 with count; invalid type → 400 with details; future timestamp → 400; unknown device_id → 400. **Effort:** M
- [ ] 3.5 `internal/application/vehicle_service.go`: `ListVehicles()`, `GetVehicleHistory(id, from, to, sensorType)`. Inject masking function. **AC:** Returns vehicles list; history filterable by type and time range. **Effort:** M
- [x] 3.6 `internal/application/masking.go`: `MaskDeviceID(raw string) string` → `DEV-****-{last4}`. Edge case: raw < 4 chars → `DEV-****-????`. **AC:** `"DEV-12345678-ABCD"` → `"DEV-****-ABCD"`; `"AB"` → `"DEV-****-????"`; deterministic: same input always produces same output. **Effort:** S

---

## Phase 4: Infrastructure — SQLite Repositories

- [ ] 4.1 `internal/infrastructure/persistence/sqlite/user_repo.go`: implement `UserRepository`. `FindByEmail` returns user with bcrypt hash. `Create` inserts with UUID generation (`crypto/rand` hex). **AC:** Create+FindByEmail round-trip works. **Effort:** M
- [ ] 4.2 `internal/infrastructure/persistence/sqlite/vehicle_repo.go`: implement `VehicleRepository`. `FindAll` returns all; `FindByDeviceID` exact match; `FindByID` UUID lookup. **AC:** All queries return expected vehicles. **Effort:** M
- [ ] 4.3 `internal/infrastructure/persistence/sqlite/sensor_repo.go`: implement `SensorDataRepository`. `BulkInsert` within a single transaction. `FindByVehicleID` with optional `from`, `to`, `type` filters (dynamic WHERE clauses). **AC:** Insert 2 points → count=2; query by type+range returns correct subset. **Effort:** M
- [ ] 4.4 `internal/infrastructure/persistence/sqlite/alert_repo.go`: implement `AlertRepository`. `Create` inserts alert. `FindAll` returns all alerts ordered by created_at DESC. **AC:** Create+FindAll round-trip. **Effort:** S

---

## Phase 5: Infrastructure — HTTP Handlers & Middleware

- [ ] 5.1 `internal/infrastructure/http/response.go`: JSON response helpers (`writeJSON`, `writeError`). Standard error format: `{"error": "code", "message": "..."}`. **AC:** `writeError(w, 401, "invalid_token", "...")` produces correct JSON body and status code. **Effort:** S
- [ ] 5.2 `internal/infrastructure/http/request.go`: request binding helpers (`decodeJSON`, `readParam`). **AC:** Malformed JSON → 400 with `validation_failed`. **Effort:** S
- [ ] 5.3 `internal/infrastructure/http/handler/auth_handler.go`: `POST /auth/login`, `POST /auth/refresh`. Parse body, call AuthService, return tokens. **AC:** Login → 200 with `access_token`+`refresh_token`+`token_type`; wrong password → 401. Refresh → 200 with new access+refresh; reused old refresh → 401. **Effort:** M
- [ ] 5.4 `internal/infrastructure/http/handler/sensor_handler.go`: `POST /sensors/data`. Decode batch, call SensorService, return `{"inserted": N}`. Enforce max 100 items. **AC:** Valid 2-point batch → 201 `{"inserted": 2}`; invalid batch → 400 with `details` array. **Effort:** M
- [ ] 5.5 `internal/infrastructure/http/handler/vehicle_handler.go`: `GET /vehicles`, `GET /vehicles/{id}/history?from=&to=&type=`. Apply masking for non-admin on BOTH endpoints (user clarification: history also masks). **AC:** Admin sees raw `device_id`; regular user sees `DEV-****-{last4}` in vehicle list AND history responses. **Effort:** M
- [ ] 5.6 `internal/infrastructure/http/handler/alert_handler.go`: `GET /alerts`. Admin → full list with details. Regular user → `200 OK` with `{"alerts": []}`. **AC:** Admin sees `low_fuel` alerts; user sees empty array (not 403). **Effort:** S
- [ ] 5.7 `internal/infrastructure/http/middleware.go`: `AuthMiddleware` — extract Bearer token (or `?token=` for WS), validate JWT, inject `sub`/`email`/`role` into context. `RBACMiddleware(allowedRoles...)` — check role from context, 403 if not allowed. `LoggingMiddleware`. **AC:** Valid token → handler executes with context populated; invalid → 401; wrong role → 403. **Effort:** L
- [ ] 5.8 `internal/infrastructure/http/router.go`: register all routes with `http.ServeMux`, apply middleware chains per endpoint per design §RBAC permissions table. WebSocket upgrade endpoint at `/api/v1/ws`. **AC:** `GET /api/v1/vehicles` without token → 401; with user token → 200 (masked). **Effort:** M

---

## Phase 6: Infrastructure — WebSocket

- [ ] 6.1 `internal/infrastructure/websocket/hub.go`: Hub struct with `register`/`unregister`/`broadcast` channels, `clients map`, single `Run()` goroutine. Broadcast is non-blocking (drop if client send buffer full). **AC:** Multiple clients register, broadcast reaches all. **Effort:** M
- [ ] 6.2 `internal/infrastructure/websocket/client.go`: Client struct with `ReadPump`/`WritePump` goroutines. Set read limit: 4096 bytes inbound. Set write buffer: 64KB outbound (user clarification). Auth check via `?token=` query param in WS upgrade path. On read/write error → unregister from Hub, close connection. **AC:** Client connects with valid token → registered in hub; invalid token → 401 during upgrade; disconnect → removed from hub; message size > 4096 → connection closed. **Effort:** M

---

## Phase 7: Integration & Wiring

- [ ] 7.1 `cmd/api/main.go`: load config, open DB + run migrations, instantiate repos → services → handlers → hub → router, start `http.ListenAndServe`. Graceful shutdown via `signal.NotifyContext`. Start Hub.Run() goroutine. **AC:** `go run ./cmd/api` starts without errors; server responds to health check. **Effort:** M
- [ ] 7.2 Seed script: insert admin (`admin@example.com`) and regular (`user@example.com`) users with bcrypt-hashed passwords, plus 3 vehicles with known device_ids. Run via `go run ./cmd/api -seed`. **AC:** Admin and user can login; 3 vehicles exist in DB. **Effort:** S

---

## Phase 8: Sensor Simulation Script

- [ ] 8.1 `scripts/simulate.go`: standalone Go program that authenticates as admin, then POSTs randomized sensor data batches (GPS, fuel, temperature) every 5s for N iterations to a running server. Accept `-server` and `-iterations` flags. Include a mode that sends decreasing fuel levels to trigger alert. **AC:** Running against local server produces successful 201 responses; alert triggered when fuel drops below threshold. **Effort:** M

---

## Phase 9: Testing

- [x] 9.1 **Unit** — `internal/infrastructure/jwt/token_test.go`: table-driven tests for generation, validation, expiration, tampering, malformed tokens, wrong secret. Use fixed clock injection. **AC:** Covers all spec JWT scenarios (§Manual JWT Authentication). **Effort:** M
- [ ] 9.2 **Unit** — `internal/application/fuel_service_test.go`: table-driven tests with mock FuelReading sequences. Covers: < 3 readings, rate 5L/h→alert, rate 2L/h→no alert, rate 0, NaN guard. **AC:** Covers all spec fuel scenarios (§Predictive Fuel Autonomy). **Effort:** M
- [x] 9.3 **Unit** — `internal/application/masking_test.go`: deterministic output verification for normal IDs, short IDs, edge cases. **AC:** Output format `DEV-****-{last4}` always correct. **Effort:** S
- [ ] 9.4 **Unit** — `internal/infrastructure/http/middleware_test.go`: table-driven tests for AuthMiddleware (valid/expired/malformed token) and RBACMiddleware (admin passes, user denied on admin-only). **AC:** Status codes match spec. **Effort:** M
- [ ] 9.5 **Integration** — `internal/infrastructure/http/handler/auth_handler_test.go`: full auth flow — login → get tokens → access protected endpoint → refresh (rotation) → old refresh rejected. Use in-memory SQLite, real test server. **AC:** Covers spec auth integration scenario. **Effort:** M
- [ ] 9.6 **Integration** — `internal/infrastructure/http/handler/sensor_handler_test.go`: seed vehicle → POST fuel batch crossing threshold → GET alerts as admin → verify alert → GET alerts as user → verify empty. **AC:** Covers spec sensor→alert integration scenario. **Effort:** M
- [ ] 9.7 **Integration** — `internal/infrastructure/websocket/`: connect WS client with valid token → POST sensor data → assert client receives broadcast. Connect with invalid token → assert rejected. **AC:** Covers spec WebSocket integration scenario. **Effort:** M

---

## Phase 10: Documentation

- [ ] 10.1 `docs/DESIGN.md`: stack rationale, architecture decisions, trade-off table (from design §Key Decisions), diagram references. **AC:** Matches AGENTS.md deliverable requirements; explains WHY each choice was made. **Effort:** M
- [ ] 10.2 `docs/SETUP.md`: step-by-step: prerequisites (Go 1.21+), clone, `cp .env.example .env`, configure JWT_SECRET, `go mod tidy`, `go run ./cmd/api -seed`, verify endpoints. **AC:** A new developer can start the server in ≤ 5 minutes following instructions. **Effort:** S

---

## Critical Path Summary

```
1.1 → 1.4 → 2.1 → 2.2 → 3.1 → 3.2 → 5.7 → 5.8 → 7.1
                           ↘ 4.1–4.4 ↗        ↗
                           3.3–3.6 ↗
```

| Phase | Tasks | Est. Effort | Critical Tasks |
|-------|-------|-------------|----------------|
| 1. Bootstrap | 5 | S+S+S+S+M = 2.5M | None |
| 2. Domain | 3 | S+M+S = 1.5M | None |
| 3. Application | 6 | L+L+M+M+M+S = 4L+3M | **3.1** (JWT), **3.2** (auth+rotation), **3.3** (fuel) |
| 4. Repos | 4 | M+M+M+S = 3M+S | None |
| 5. HTTP | 8 | S+S+M+M+M+S+L+M = L+5M+2S | **5.7** (middleware) |
| 6. WebSocket | 2 | M+M = 2M | None |
| 7. Wiring | 2 | M+S = M+S | **7.1** (main) |
| 8. Simulation | 1 | M | None |
| 9. Testing | 7 | M+M+S+M+M+M+M = 6M+S | None |
| 10. Docs | 2 | M+S = M+S | None |

**Total estimated effort:** 5L + 22M + 9S ≈ **4–5 days** solo (1.5 days with pair help on critical path)

## Size Anomalies

- **Phase 3 tasks together** exceed the 400-line review budget — JWT alone (`token.go`) is ~200 lines, auth_service ~150, fuel_service ~120 → PR 1 will be ~500–600 lines with tests. This is acceptable as a foundation PR since it contains the three most critical components and their tests.
- **Phase 5 HTTP handlers** total ~700 lines. Splitting handlers into separate commits within PR 2 keeps each reviewable.
- If single PR delivery is required, the full implementation exceeds the budget by 7–8×. **Strongly recommend chained PRs** (4 slices as outlined in Work Units above).
