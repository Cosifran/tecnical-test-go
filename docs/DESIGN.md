# DESIGN.md — IoT Fleet Monitoring System

## Stack Rationale

| Component | Choice | Why |
|-----------|--------|-----|
| **Backend Language** | Go 1.21+ | Strong concurrency model (goroutines), fast compilation, excellent standard library for HTTP/WebSocket, and zero-surprise deployment (single binary). |
| **Database** | SQLite | Zero configuration, no separate server process, perfect for a 3-day proof of concept. WAL mode provides read concurrency. Easy swap to PostgreSQL later. |
| **WebSocket** | Gorilla WebSocket | Industry-standard Go WebSocket library. Stable, well-tested, handles ping/pong and close frames correctly. |
| **JWT** | Manual (HMAC-SHA256) | Explicit requirement of the technical test. Uses `crypto/hmac` and `encoding/base64` from Go stdlib — no third-party JWT library. |
| **Password Hashing** | bcrypt (`golang.org/x/crypto`) | Industry standard for password hashing. Configurable cost factor. The only external crypto dependency. |
| **HTTP Framework** | `net/http` + `ServeMux` (Go 1.22+) | No external framework. Go 1.22+ adds method-based routing (`POST /path`) and path parameters (`{id}`) to the standard library. Keeps the dependency count minimal. |
| **Frontend** | React 19 + Vite 7 SPA | Client-rendered dashboard. Vite chosen over Next.js because the backend is a separate origin — SSR adds no value for an auth-gated dashboard. |

## Architecture: Clean Architecture

```
┌──────────────────────────────────────────────────┐
│                   cmd/api/main.go                 │
│               (Composition Root / Wiring)         │
└────────────────────┬─────────────────────────────┘
                     │
         ┌───────────▼───────────┐
         │    Infrastructure     │
         │  ┌─────────────────┐  │
         │  │ HTTP Handlers   │  │
         │  │ Middleware       │  │
         │  │ WebSocket Hub    │  │
         │  │ JWT TokenService │  │
         │  │ SQLite Repos     │  │
         │  └────────┬────────┘  │
         └───────────┼───────────┘
                     │
         ┌───────────▼───────────┐
         │    Application Layer   │
         │  ┌─────────────────┐   │
         │  │ AuthService      │   │
         │  │ SensorService    │   │
         │  │ VehicleService   │   │
         │  │ FuelService      │   │
         │  │ Masking          │   │
         │  └────────┬────────┘   │
         └───────────┼───────────┘
                     │
         ┌───────────▼───────────┐
         │      Domain Layer     │
         │  ┌─────────────────┐  │
         │  │ Entities         │  │
         │  │ Repository IFs   │  │
         │  │ Errors            │  │
         │  └─────────────────┘  │
         └───────────────────────┘
```

**Dependency Rule**: Dependencies point inward only. Domain has NO imports from application or infrastructure. Application imports domain only. Infrastructure imports both domain and application.

## Key Architecture Decisions

### 1. Manual JWT vs Library

**Decision**: Implement JWT encoding/decoding using `crypto/hmac` and `encoding/base64` from Go stdlib only.

| Aspect | Manual JWT | JWT Library (e.g., `golang-jwt`) |
|--------|-----------|----------------------------------|
| **Pros** | No external dependency. Full control over token format. Demonstrates understanding of JWT spec. | Battle-tested. Handles edge cases (algorithm none, timing attacks). Less code. |
| **Cons** | More code to maintain. Must handle base64url padding, constant-time comparison, and expiration manually. | Black-box dependency. Harder to debug when something goes wrong. |
| **Verdict** | Required by spec. The implementation is well-tested with 10+ test cases covering tampering, expiration, and malformed tokens. |

### 2. SQLite vs PostgreSQL

**Decision**: Use SQLite with WAL mode for persistence.

| Aspect | SQLite | PostgreSQL |
|--------|--------|-----------|
| **Pros** | Zero config. Single file. No Docker dependency. WAL mode gives read concurrency. Easy to inspect with `sqlite3` CLI. | Production-grade. Better concurrent write throughput. Richer query planner. |
| **Cons** | Single-writer limit (WAL helps). Not suitable for high-write production workloads. | Requires separate server process. More complex deployment. Overkill for a demo. |
| **Verdict** | Correct for a 3-day test. Easy swap path: repository interfaces abstract the database, so only the persistence layer changes. |

### 3. WebSocket for Real-Time Updates

**Decision**: Use WebSocket (Gorilla) with a Hub/Client fan-out pattern.

| Aspect | WebSocket | Server-Sent Events (SSE) |
|--------|-----------|--------------------------|
| **Pros** | Full-duplex. Industry standard for IoT dashboards. Gorilla library is mature. | Simpler. HTTP-only. Auto-reconnect. No library needed. |
| **Cons** | Requires upgrader. Ping/pong management. Token auth needs `?token=` workaround. | Unidirectional (server→client only). No binary support. |
| **Verdict** | Spec requires WebSocket. The Hub/Client pattern with channel-based registration is idiomatic Go and avoids mutexes. |

### 4. Role-Based Access Control with Device ID Masking

**Decision**: Non-admin users see masked device IDs (`DEV-****-XC54`). Admin users see raw IDs.

| Aspect | Service-Layer Masking | Handler-Layer Masking |
|--------|----------------------|----------------------|
| **Pros** | Business rule lives where business logic lives. Testable with service-level tests. | Simpler — just transform the JSON response. |
| **Cons** | Requires passing user role into service methods. | Mixing concerns — HTTP handler knows about business rules. |
| **Verdict** | Service layer. Masking is a business rule ("non-admins must not see raw IDs"). The handler's job is to parse requests and serialize responses, not decide what data looks like. |

## Data Flow: Sensor Ingestion

```
Device → POST /api/v1/sensors/data (JSON batch)
  → AuthMiddleware (validate JWT, inject claims)
  → RBACMiddleware (admin only)
  → SensorHandler.Ingest
    → SensorService.IngestBatch
      1. Validate each input (type, timestamp, value schema)
      2. Resolve device_id → vehicle_id (with caching)
      3. Generate UUIDs, parse timestamps
      4. BulkInsert (transactional: all succeed or all fail)
      5. For fuel readings → FuelService.CheckAutonomy
         → If autonomy < 1h → AlertRepo.Create ("low_fuel")
  → 201 Created {inserted: N}
```

## Data Flow: WebSocket Connection

```
Client → GET /api/v1/ws?token=<jwt>
  → ServeWS
    1. Extract ?token= from query string
    2. Validate token via TokenService.Validate
    3. If invalid → 401, close
    4. If valid → upgrade HTTP to WebSocket
    5. Create Client, register with Hub
    6. Start ReadPump goroutine (ping/pong, disconnection detection)
    7. Start WritePump goroutine (broadcast relay to client)

Hub.Broadcast(msg) → fans out to all registered Clients' send channels
```

## Database Schema

See `migrations/001_init.sql` for the full schema. Key design choices:

- **UUIDs as TEXT**: SQLite has no UUID type. UUIDs are stored as text strings.
- **Timestamps as TEXT**: ISO 8601 format. Sorts lexicographically. Round-trips cleanly through Go's `time.RFC3339`.
- **JSON values as TEXT**: `sensor_data.value` stores JSON payloads validated in Go code.
- **WAL mode**: Enabled at connection time for concurrent read/write.

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| **Domain** | Entity structure, error constants | Direct struct checks |
| **Application** | Business logic (auth, fuel calculation, masking) | Service-level unit tests with mock interfaces |
| **JWT** | Token generation, validation, edge cases | Table-driven tests (tampered, expired, malformed) |
| **Persistence** | Repository CRUD operations | In-memory SQLite (`":memory:"`) |
| **HTTP** | Handlers, middleware | `httptest.NewRecorder()` |
| **WebSocket** | Hub register/unregister/broadcast | Unit tests with channel-based assertions |

## Configuration

All configuration via environment variables (see `.env.example`):

| Variable | Default | Purpose |
|----------|---------|---------|
| `JWT_SECRET` | (required) | HMAC-SHA256 signing key, min 32 chars |
| `DB_PATH` | `fleet.db` | SQLite database file path |
| `PORT` | `8080` | HTTP server port |
| `BCRYPT_COST` | `12` | Password hashing cost factor (min 10) |
| `ACCESS_TOKEN_TTL_MINUTES` | `15` | Access token lifetime |
| `REFRESH_TOKEN_TTL_DAYS` | `7` | Refresh token lifetime |

---

## Frontend Web Dashboard

### Tech Stack

| Component | Choice | Why |
|-----------|--------|-----|
| **Framework** | React 19 | Latest React with Compiler support — no `useMemo`/`useCallback` needed |
| **Build Tool** | Vite 7 | Fast HMR, native ESM, simple proxy config. No SSR needed — backend is a separate origin |
| **Routing** | react-router-dom v7 | Industry standard data router with layout routes and lazy loading |
| **State** | Zustand 5 | Lightweight (~3KB), built-in `persist` middleware for localStorage caching, `useShallow` selectors |
| **Maps** | MapLibre GL + react-map-gl | Free, open-source map rendering. react-map-gl v8.1+ supports React 19 |
| **Charts** | Recharts 2 | Simple, declarative chart API for fuel/temperature history |
| **Styling** | Tailwind CSS 4 | Utility-first, CSS-first configuration, no `var()` in className |
| **Testing** | Vitest + Testing Library | Fast, jsdom environment, component + unit test coverage |
| **WebSocket** | Native WebSocket via singleton manager | No library — full control over reconnect backoff and message dispatch |
| **HTTP Client** | Native fetch wrapper | Zero deps. Bearer injection, 401 refresh queue, typed responses |

### Architecture Overview

```
frontend-web/src/
├── api/              # apiClient (fetch wrapper) + endpoint functions
├── components/       # UI components (Layout, MapView, ChartView, Toast, etc.)
├── guards/           # AuthGuard + AdminGuard route protection
├── hooks/            # useAuth, useOnline
├── pages/            # Route-level components (LoginPage, VehicleListPage, etc.)
├── stores/           # Zustand slices (authStore, fleetStore, wsStore, toastStore)
├── styles/           # globals.css (Tailwind 4 directives)
├── types/            # TypeScript interfaces (domain, api, ws)
├── utils/            # cn, jwt decode, device ID masking
└── websocket/        # WebSocketManager singleton + useWebSocket hook
```

**Data flow**: React components read from Zustand stores → stores call `apiClient` for REST → `WebSocketManager` push updates into stores → components re-render via `useShallow` selectors.

**Offline**: `fleetStore` persists to localStorage via Zustand `persist` (capped at 50 history points per vehicle). When `navigator.onLine` is false, an `OfflineBanner` appears and the UI serves cached data. On reconnect, the app refetches and reconnects WebSocket.

**RBAC**: JWT payload is decoded client-side to extract the user role. `AuthGuard` redirects unauthenticated users to `/login`. `AdminGuard` redirects non-admins to `/vehicles`. Device IDs are masked in render via `maskDeviceId()` for non-admin users.

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Vite SPA over Next.js | Vite | Backend is separate origin at `:8080`. SSR provides no value for an auth-gated dashboard. SPA is simpler to reason about offline behavior. |
| Zustand over Redux/Context | Zustand 5 | 3KB, built-in localStorage persist, `useShallow` selectors. No provider nesting. |
| Native fetch over Axios | Custom `apiClient` | Zero deps, full control over 401 retry queue. Only one login form — no need for 14KB of Axios. |
| localStorage over IndexedDB | Zustand persist | ~5MB cap is sufficient for vehicle list + 50 history points per vehicle. Sync API. |
| Singleton WebSocket | `WebSocketManager` class + hook | Singleton ensures one connection. Reconnect state survives unmount. Testable with `MockWebSocket`. |

### Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `apiClient` 401 retry + refresh queue | Mock `globalThis.fetch`; assert retry-once, queue concurrency |
| Unit | `maskDeviceId` function | Mirror backend tests: short strings, normal IDs, edge cases |
| Unit | JWT payload decode | Test base64 decode, type extraction, malformed input |
| Unit | Store logic (auth, fleet, ws) | Mock apiClient; assert state transitions |
| Component | Pages (Login, VehicleList, Alerts) | Mock stores, verify rendering and RBAC |
| Component | Interactive elements (Map, Chart, Toast) | Mock MapLibre, verify props and state updates |
| Integration | WebSocket reconnect | Mock WebSocket; verify backoff timing and reconnect |

**Total tests**: 183 across 24 test files. Mock strategy: `vi.fn()` for `fetch`, custom `MockWebSocket` class for WS tests.