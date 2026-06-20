# Exploration: Frontend Web Dashboard

> Change name: `empecemos a crear el front, pero la propuesta para que quedo todo el proceso usando sdd`
> Slug: `frontend-web`
> Phase: explore
> Date: 2026-06-19
> Project: tecnical-test-go

---

## Current State

The backend (`backend/`, Go 1.25 + SQLite + manual JWT + WebSockets) is implemented, fully tested, and runs at `:8080` with seed data. The repo has:

- `docs/DESIGN.md` and `docs/SETUP.md` covering backend architecture, API endpoints, env vars, and demo flows.
- `docs/estudio/` — study sheets and a 5-minute video script.
- `openspec/changes/backend-go-*` — the completed backend change (proposal, spec, design, tasks).
- `openspec/changes/archive/` — empty (backend was not formally archived yet).
- `frontend/` — **empty directory** (only `.` and `..`).
- No `package.json`, no Next.js/Vite config, no source code, no tests, no `node_modules`.

The frontend-web directory referenced in `AGENTS.md` (`frontend-web/`) does not exist either. Anything under `frontend-web/` must be created from scratch in the apply phase.

### Backend API contract (verified from `router.go`, handlers, domain entities, `cmd/api/main.go`)

| Method | Path | Auth | RBAC | Request body | Response body |
|--------|------|------|------|--------------|---------------|
| POST | `/api/v1/auth/login` | none | — | `{email, password}` | `{access_token, refresh_token, token_type:"Bearer", expires_in}` |
| POST | `/api/v1/auth/refresh` | none | — | `{refresh_token}` | Same shape as login. Errors: `token_expired`, `invalid_token` |
| GET | `/api/v1/vehicles` | Bearer | admin, user | — | `{vehicles: Vehicle[]}` (DeviceID masked for non-admin) |
| GET | `/api/v1/vehicles/{id}/history` | Bearer | admin, user | Query: `from` (RFC3339), `to` (RFC3339), `type` (gps/fuel/temperature) | `{vehicle: Vehicle, history: SensorData[]}` |
| POST | `/api/v1/sensors/data` | Bearer | **admin only** | `SensorInput[]` (max 100, min 1) | `{inserted: N}` |
| GET | `/api/v1/alerts` | Bearer | **admin only** | — | `{alerts: Alert[]}` (newest first) |
| GET | `/api/v1/ws` | `?token=` | admin, user | Upgrade → WebSocket | See WebSocket messages below |

### Domain shapes the frontend will consume

- `Vehicle`: `{id, device_id, name, created_at}`. `device_id` is `DEV-****-last4` for non-admin, raw for admin.
- `SensorData`: `{id, vehicle_id, type, value (json.RawMessage), timestamp, created_at}`. `value` is a polymorphic JSON blob whose shape depends on `type`:
  - `gps`: `{lat: number, lng: number}`
  - `fuel`: `{level: number (liters), unit: "liters"}`
  - `temperature`: `{celsius: number}`
- `Alert`: `{id, vehicle_id, type, severity, details (json.RawMessage), created_at}`. Only one type today: `low_fuel`.
- `Claims` (decoded from JWT payload): `{sub (user id), email, role: "admin"|"user", iat, exp, type: "access"|"refresh"}`. TTLs: access 15 min, refresh 7 days.

### Auth mechanics

- All REST endpoints except login/refresh require `Authorization: Bearer <access_token>`. 401 with `invalid_token` on missing/malformed/tampered; 401 with `token_expired` on past `exp`.
- RBAC returns 403 with `forbidden` when role is wrong.
- WebSocket sends the token as a query param: `ws://host/api/v1/ws?token=<access_token>`. Any token error → 401 before upgrade.
- The token's role is in the JWT payload — no need to call any `/me` endpoint. The frontend should decode the access token (or keep a `/me` field) and use role to gate UI (alerts page, raw device IDs).

### WebSocket protocol (verified from `hub.go`, `client.go`, `websocket_test.go`)

- Outbound-only. Server fans out to all connected clients. No client→server message handling on the wire.
- Each broadcast is a JSON object. From `websocket_test.go` lines 70 and 100 the canonical shapes are:
  - `{"type": "sensor_update", "vehicle_id": "..."}` — broadcast on every successful sensor batch.
  - `{"type": "low_fuel", "vehicle_id": "..."}` — broadcast when a `low_fuel` alert is created.
- The simulator (`backend/scripts/simulate.go`) sends a batch every 5 seconds (default 20 iterations). In `-alert-mode` fuel drops fast to trigger `low_fuel`.
- Ping/pong: server pings every 54s (`pingPeriod`), expects pong within 60s (`pongWait`). Client should respond to ping frames automatically (the browser does, but a custom reconnect strategy may want to keep-alive).
- Buffer: server's per-client send channel is 256 messages. Slow clients are dropped automatically (no back-pressure to other clients).

### Critical backend gap — RESOLVED ✅

**~~The WebSocket hub is wired but never broadcast to in production.~~** This was fixed in a surgical backend patch:
- `SensorService` now accepts a `Broadcaster` interface via `WithBroadcaster()`
- After successful `BulkInsert`, the service broadcasts `{"type": "sensor_update", "vehicle_id": "..."}` to all connected WebSocket clients
- `cmd/api/main.go` wires the hub before creating the sensor service
- 2 new tests verify broadcast behavior and nil-safety

The real-time requirement is now unblocked.

### Docs/estudio signals

The estudio folder is purely a backend study aid (clean architecture, JWT, fuel, websockets, masking, testing, demo). It does not prescribe a frontend stack — that decision is still open. `DESIGN.md` line 13 says "React/NextJS (separate project)" and leaves the call to the frontend phase.

### Existing OpenSpec layout

- `openspec/config.yaml` — `schema: spec-driven`, mode is `hybrid`, frontend is marked `not detected`.
- `openspec/specs/` — empty. No `fleet-monitor/system` or similar canonical spec.
- `openspec/changes/archive/` — empty.
- `openspec/changes/backend-go-{proposal,spec,design,tasks}.md` — the closed backend change set. Worth reading for pattern consistency when writing the frontend proposal/spec/design/tasks.

---

## Affected Areas (frontend, all new files)

- `frontend-web/` — entire directory. The repo expects this exact name per `AGENTS.md` (not `frontend/`).
- `package.json` — define scripts: `dev`, `build`, `start` (if Next.js), `lint`, `typecheck`, `test`.
- `tsconfig.json` — strict mode, path alias (`@/*` → `./src/*`).
- `next.config.ts` or `vite.config.ts` — chosen in design phase.
- `tailwind.config.ts` (or PostCSS config for Tailwind 4) + global stylesheet.
- `src/app/` or `src/` structure depending on framework choice.
- `docs/SETUP.md` and `docs/DESIGN.md` — must be updated to mention the frontend dev workflow.
- `openspec/specs/` — may need a `fleet-monitor/system` canonical spec if the user wants archived deltas to merge into something; the existing backend change set did not, so this is optional.

---

## Approaches

### 1. Framework: React SPA (Vite) vs Next.js 15 App Router

| Aspect | Vite + React 19 | Next.js 15 App Router |
|--------|-----------------|------------------------|
| Server Components | No (SPA, all client) | Yes (default) — can stream map data, mask server-side |
| Backend proxy / CORS | Need Vite dev proxy or backend CORS | API routes or rewrites can hide origin |
| Offline-first (offline is 20% of the rubric) | Natural fit — client owns everything | Slightly more complex (Service Worker + client islands) |
| Time-to-scaffold for a 3-day test | Faster (less ceremony) | Slower (need to choose client/server boundaries) |
| WebSocket in dev | Direct — `ws://localhost:8080/api/v1/ws` | Same — WebSocket doesn't get RSC routing. Works fine in `useEffect`. |
| RBAC / role gating | Client-side only | Server Actions could gate, but overkill here |
| Bundle size control | Vite is excellent | Next bundles aggressively; can disable RSC per route |
| Fit with the rubric (code quality 30%) | Simpler mental model; less Next-specific magic | More powerful but more surface area to explain |
| Verdict for this test | **Recommended.** A SPA with a clear store, clear service layer, and explicit offline cache is easier to defend in a 5-min video. |

**Recommendation: Vite + React 19.** The backend is already a separate origin (`localhost:8080`); we don't need SSR to talk to it. RBAC is enforced server-side; client role is only for UI gating. Offline is straightforward: a `useOnline` hook + Zustand-persisted store.

### 2. State management: Zustand vs React Context

| Aspect | Zustand 5 | React Context |
|--------|-----------|---------------|
| Persist middleware | Built-in (localStorage) | DIY + a wrapper lib |
| Selectors / re-render cost | Trivial with `useShallow` | All consumers re-render on any change |
| Slices pattern | First-class | Awkward (nested providers) |
| Auth state with role | One slice | Auth provider + selectors |
| Bundle cost | ~3 KB | 0 |
| Verdict | **Recommended.** | Not enough to justify for this scope. |

**Recommendation: Zustand 5** with three slices: `authStore` (tokens, user, role), `fleetStore` (vehicles, sensor history cache, alerts), `wsStore` (connection status, live message buffer). `authStore` uses `persist` middleware to survive reloads (offline requirement).

### 3. Offline strategy: localStorage vs IndexedDB

| Aspect | localStorage | IndexedDB |
|--------|--------------|-----------|
| Capacity | ~5 MB, sync API | Hundreds of MB, async API |
| Vehicle list + last 24h of history | Fits easily | Overkill |
| Binary or large blobs | Bad | Good |
| Persistence middleware (Zustand) | Built-in | `idb-keyval` wrapper needed |
| Replay strategy on reconnect | Just read the cache | Same, slightly faster on huge data |
| Verdict | **Recommended for v1.** | IndexedDB becomes relevant only if we cache full history of every vehicle. |

**Recommendation: localStorage via Zustand `persist`.** Cache the vehicle list and the most recent history per vehicle. On WS reconnect, drop the live buffer; the cache survives reload. Add an explicit "stale" timestamp so the UI can show "last updated X min ago" when offline.

### 4. Map library: MapLibre GL (user choice)

| Aspect | MapLibre GL | Leaflet (react-leaflet) |
|--------|-------------|------------------------|
| Tiles | Free OSM tiles via demotiles; vector tiles need a provider | Free OSM tiles, no API key |
| Bundle | ~800 KB (WebGL) | ~150 KB |
| Performance for 3–100 vehicles | Excellent (GPU-accelerated) | Fine |
| Tile offline caching | Service Worker tiles, more setup | Doable with `leaflet.offline` plugin |
| React integration | `react-map-gl` is well-maintained | `react-leaflet` v4+ works with React 18/19 |
| Code quality signal | Modern, open-source, no Mapbox token | Simpler; battle-tested |
| Verdict | **User selected.** | |

**Recommendation: MapLibre GL JS + `react-map-gl`.** The user explicitly chose MapLibre over Leaflet. We will use OpenStreetMap raster tiles (no API key required) via `react-map-gl`. For offline, we cache last-known positions in Zustand `fleetStore` rather than tile caching.

### 5. Charting: Recharts vs Chart.js

| Aspect | Recharts | Chart.js (react-chartjs-2) |
|--------|----------|---------------------------|
| React integration | Native components, declarative | Imperative wrapper |
| Bundle | ~95 KB | ~120 KB |
| TypeScript | Excellent | Good |
| Theming with Tailwind 4 | Color constants work (per tailwind-4 skill: `var(--color-primary)` in chart props) | Same |
| Verdict | **Recommended.** | |

**Recommendation: Recharts.** Two charts: (1) fuel level over time per vehicle, (2) temperature over time per vehicle. Both consume the cached history and the live buffer.

### 6. Forms: react-hook-form + zod (light, optional)

Login form only. Can use `useActionState` from React 19 instead, which avoids a new dep. Verdict: **use built-in `useActionState`** to keep the dep list tight.

### 7. Test runner: Vitest + Testing Library vs Playwright only

- `openspec/config.yaml` already says frontend will use Vitest + Testing Library. The backend is strict TDD (`strict_tdd: true`).
- Vitest is faster, has HMR for tests, and works with the same Vite pipeline.
- Playwright for one E2E flow: login → see vehicle list → see live update on the map. Optional but high signal for the rubric.

**Recommendation: Vitest + Testing Library** for unit/component. Defer Playwright to the proposal; mention as "nice-to-have if time allows."

### 8. Recommended stack (concrete)

- **Build**: Vite 5 + React 19 + TypeScript strict.
- **Routing**: react-router-dom 6 (or TanStack Router). Pick react-router-dom for familiarity.
- **State**: Zustand 5 with `persist` (localStorage) for auth + fleet cache.
- **Styling**: Tailwind CSS 4.
- **HTTP**: native `fetch` wrapped in a `apiClient` module that handles the Bearer header and 401-refresh-then-retry once.
- **WebSocket**: a `useWebSocket` hook in a singleton class, with auto-reconnect (exponential backoff up to 30s) and ping/pong from the browser.
- **Map**: MapLibre GL JS + `react-map-gl`, OSM raster tiles.
- **Charts**: Recharts.
- **Tests**: Vitest + Testing Library.

---

## Recommendation

**Vite + React 19 + TypeScript strict + Tailwind 4 + Zustand 5 (persist) + MapLibre GL + Recharts + Vitest.**

This stack:
- Maximizes code quality signal (every layer is testable, no framework magic).
- Hits offline cleanly: Zustand `persist` covers cache; a `useOnline` hook shows stale data with a banner.
- Fits the rubric weights (30% code quality, 25% functionality, 20% offline, 15% docs, 10% tests).
- Avoids new external JWT libraries (none needed on the frontend — we just store the access token and use it as `Authorization: Bearer ...`).
- Lets us reuse the existing backend simulation flow: `go run ./cmd/api` + `go run scripts/simulate.go -alert-mode` to drive the dashboard end-to-end.

**Sequencing the work** (for the proposal phase):
1. Backend broadcast hook (the gap above) — small, surgical, in a separate change or pre-task.
2. Scaffold frontend with Vite, strict TS, Tailwind 4, ESLint.
3. API client + auth store + login page.
4. Protected route wrapper using role from JWT.
5. Vehicle list page with masking verified client-side.
6. Vehicle detail page with map + charts + history.
7. WebSocket hook with reconnect + status indicator.
8. Offline cache + `useOnline` banner.
9. Alert page (admin-only) + visibility guard.
10. Vitest coverage for apiClient masking, WebSocket reconnect, and the masking utility.

---

## Risks

1. **WebSocket never broadcasts in production** (CRITICAL). `hub.Broadcast` is wired nowhere in `cmd/api/main.go`. The frontend can connect, but no messages will arrive unless we add a hook in the sensor ingestion path. **The proposal must call this out as a hard dependency** for the apply phase, or scope the frontend to a degraded mode (no live updates) and add a TODO. My recommendation: a tiny backend delta (a `Notifier` interface) is included as a prerequisite task in this same change, so the demo is honest end-to-end.
2. **WebSocket auth via query param leaks the token in browser history and server logs.** The backend already requires it, so we cannot avoid it. Mitigation: do not log the URL on the client, and pass the access token (15 min TTL) not the refresh token. The frontend should never persist the WS URL with the token.
3. **React 19 + react-leaflet compatibility.** react-leaflet 4 is built for React 18; v5 (when stable) targets React 19. Today (Jun 2026) react-leaflet should be on a version that supports React 19, but the apply phase should verify the exact version and pin it.
4. **Role in JWT vs role in store.** The role in the access token is the source of truth. If the backend changes a user's role mid-session, our UI will not reflect it until the next token refresh. Document this. The frontend should always re-read role from the persisted store, not from a separate `/me` call.
5. **Tailwind 4 conventions.** The `tailwind-4` skill is explicit: never use `var()` in className, never use hex colors. Recharts props need color constants. The design phase must spell this out so the apply phase doesn't violate it.
6. **TypeScript strict + react-leaflet types.** react-leaflet's types can be loose. Apply phase may need a small typed wrapper.
7. **Vite dev proxy for the API and WS.** Need to proxy `/api` and the WS upgrade to `http://localhost:8080` in dev so cookies and the WS origin match. CORS on the backend is currently not configured (the backend serves with no `Access-Control-Allow-Origin`), so without the proxy, the browser will block the request. **The backend must enable CORS for the dev origin, or we ship the Vite proxy.** The proxy is the path of least resistance.
8. **time budget**. 3-day test, day 1 is scaffolding + login, day 2 is the dashboard, day 3 is polish + tests + video. The backend broadcast hook is the only piece that could blow the budget.

---

## Open Questions for the Orchestrator / User

These should be confirmed before the proposal phase writes anything:

1. **~~Backend broadcast hook~~** — ✅ RESOLVED. Fixed in backend.
2. **Backend CORS** — ✅ RESOLVED. CORS middleware added to backend (`Access-Control-Allow-Origin: *`, preflight 204). Postman and frontend dev both work without Vite proxy, though proxy can still be used.
3. **Mobile (React Native)**: `AGENTS.md` lists it as optional. The orchestrator should ask the user whether to scope mobile into this change or defer. My vote: defer. The frontend-web change is large enough.
4. **Test coverage target**: the backend has unit + integration. For the frontend, Vitest is enough. E2E with Playwright is a stretch goal. Confirm before designing tasks.
5. **i18n / a11y**: out of scope for a 3-day test. Confirm.

---

## Ready for Proposal

**Yes**, prerequisites resolved:
1. ✅ Backend WebSocket broadcast gap fixed.
2. ✅ Backend CORS enabled for Postman and frontend dev.

After those two are decided, the orchestrator should dispatch `sdd-propose` to write `openspec/changes/frontend-web/proposal.md` following the same shape as `backend-go-proposal.md`.

---

## Appendix: File-by-file read in this exploration

- `backend/internal/infrastructure/http/router.go` — route table, RBAC order, WS auth via query param
- `backend/internal/infrastructure/http/middleware.go` — `AuthMiddleware` accepts Bearer or `?token=`
- `backend/internal/infrastructure/http/handler/{auth,vehicle,sensor,alert}_handler.go` — request/response shapes
- `backend/internal/infrastructure/websocket/{hub,client}.go` — broadcast model, ping/pong timing, buffer sizes
- `backend/internal/infrastructure/websocket/websocket_test.go` — confirms the WS message format `{"type":"sensor_update","vehicle_id":"..."}` and `{"type":"low_fuel",...}`
- `backend/internal/domain/entity.go` — domain shapes, JWT claims
- `backend/internal/application/{auth_service,vehicle_service,sensor_service,masking}.go` — auth flow, masking rule, sensor validation
- `backend/internal/infrastructure/jwt/token.go` — TTLs (15m access / 7d refresh), HS256, base64url no padding
- `backend/cmd/api/main.go` — wiring; **confirms `hub` is never broadcasted to**
- `backend/scripts/simulate.go` — drives the dashboard via repeated POSTs every 5s
- `docs/DESIGN.md`, `docs/SETUP.md` — stack rationale, endpoint table, demo flow
- `docs/estudio/componentes/04-websockets.md` — confirms intended broadcast flow (gap in production)
- `openspec/config.yaml` — confirms `hybrid` mode, frontend not detected, Vitest plan
- `openspec/changes/backend-go-{proposal,spec,design,tasks}.md` — pattern reference for the frontend change set (read partial)
- `AGENTS.md`, `Prueba Técnica Desarrollador II - Simon Movilidad.md` — rubric weights, structure expected

---

## Skill Resolution

Skills loaded: `nextjs-15`, `react-19`, `tailwind-4`, `typescript`, `zustand-5` (all five injected by the orchestrator). Loaded before task work as instructed.
