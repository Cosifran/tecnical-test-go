# Proposal: Frontend Web Dashboard for IoT Fleet Monitoring

## Intent

The backend exposes a fully working REST + WebSocket API, but fleet operators cannot see or act on the data without a UI. This frontend provides a real-time dashboard where admins and users monitor vehicle location, fuel levels, temperature, and alerts. It turns raw sensor streams into actionable visibility.

## Scope

### In Scope
- **Auth**: Login page, token refresh, role-based route guards, 401 retry
- **Vehicle List**: Table with masked device IDs for non-admins
- **Vehicle Detail**: Map (MapLibre) + fuel/temperature charts (Recharts) + sensor history
- **Real-time**: WebSocket hook with auto-reconnect, live marker updates, toast alerts
- **Alerts page**: Admin-only, predictive low-fuel list
- **Offline resilience**: Zustand-persist cache, stale-data banner, graceful degradation
- **Tests**: Vitest + Testing Library for apiClient, masking utility, store logic

### Out of Scope
- Mobile app (React Native) — deferred to a later change
- i18n / a11y audit — beyond 3-day budget
- E2E with Playwright — stretch goal only
- PWA / Service Worker tile caching

## Capabilities

### New Capabilities
- `vehicle-dashboard`: List, detail, map, and chart views for fleet data
- `auth-session`: Login, token refresh, role-aware navigation
- `real-time-updates`: WebSocket connection, live sensor broadcasts, reconnect logic
- `offline-cache`: Persisted fleet state, stale indicators, history replay

### Modified Capabilities
- None (backend is frozen; frontend is all new)

## Target Users

| Persona | Views | Permissions |
|---------|-------|-------------|
| **Admin** | Vehicles, Alerts, raw device IDs, predictive warnings | Full read + sensor ingestion |
| **User** | Vehicles only, masked IDs (`DEV-****-XC54`), no alerts | Read-only fleet data |

Role is read from the JWT payload; no `/me` endpoint is required.

## Approach

**Stack**: Vite 5 + React 19 + TypeScript strict + Tailwind 4 + Zustand 5 + MapLibre GL + Recharts + Vitest.

**Why Vite over Next.js**: The backend is a separate origin (`:8080`). SSR adds no value here, and a SPA gives a simpler mental model, faster scaffold, and cleaner offline story — all critical for a 3-day test and a 5-minute demo video.

**State**: Zustand 5 with three slices — `auth` (persist to localStorage), `fleet` (vehicles + history cache), `ws` (connection + live buffer). Selectors via `useShallow` to avoid re-renders.

**API client**: Native `fetch` wrapper that injects `Authorization: Bearer`, catches 401, refreshes the token once, and retries the original request.

**WebSocket**: Singleton hook with exponential backoff reconnect (max 30s). The browser handles ping/pong frames automatically. On `sensor_update`, merge into the fleet cache; on `low_fuel`, push a toast if admin.

**Offline**: `fleetStore` is partially persisted. When `navigator.onLine` is false, show a banner and continue serving cached data with a "last updated" timestamp.

## Key Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Build tool | Vite | Faster cold start, simpler config than Next.js for a SPA |
| Map | MapLibre GL + react-map-gl | User preference; GPU-accelerated, free OSM tiles |
| Charts | Recharts | Native React components, smaller bundle than Chart.js |
| Forms | React 19 `useActionState` | Avoids new dependency for a single login form |
| Offline storage | localStorage via Zustand `persist` | Enough capacity for vehicle list + recent history; simpler than IndexedDB |
| Test runner | Vitest + Testing Library | Matches Vite pipeline, fast HMR, aligned with `openspec/config.yaml` |

## Rubric Alignment

| Criterion | Weight | How we hit it |
|-----------|--------|---------------|
| Code quality | 30% | Strict TS, flat interfaces, const types, Clean-ish store slices, no `any` |
| Functionality | 25% | All backend endpoints consumed, RBAC gated, real-time updates working |
| Offline / resilience | 20% | Zustand persist, `useOnline` banner, WS auto-reconnect, stale indicators |
| Documentation | 15% | `DESIGN.md` and `SETUP.md` updated with frontend dev flow |
| Testing | 10% | Unit tests for apiClient retry, masking, fuel-alert utility, store reducers |

## First Slice / MVP Boundary

The minimum demo-worthy frontend includes:
1. Scaffold (Vite + TS + Tailwind + router)
2. Login page + auth store with persist
3. Protected layout with role-based sidebar
4. Vehicle list page (verify masking for non-admin)
5. Vehicle detail page (map + last-known location)
6. WebSocket hook connected, showing a live toast when data arrives
7. Offline banner + cache surviving a hard reload

Charts and the dedicated Alerts page are **slice 2**; they improve the rubric but are not required for the first demo run.

## Dependencies & Risks

| Dependency | Status | Risk |
|------------|--------|------|
| Backend CORS | ✅ Resolved | None |
| Backend WS broadcast | ✅ Resolved | None |
| Backend running at `:8080` | ✅ Available | None |
| React 19 ecosystem maturity | ⚠️ Monitor | MapLibre wrapper must support React 19; pin exact versions |
| Time budget | ⚠️ Tight | 3-day test; frontend gets ~1.5 days. Scope must be ruthlessly enforced. |

## Rollback Plan

The frontend is a net-new directory (`frontend-web/`). Rollback is `rm -rf frontend-web/` and reverting any `docs/*.md` updates. No backend changes are required.

## Open Questions

1. **Mobile scope**: Confirm deferred to a separate change.
2. **Playwright E2E**: Confirm stretch goal; not in MVP.
3. **Demo video script**: Should the frontend proposal include a suggested narrative, or is that owned by `docs/estudio/`?

## Success Criteria

- [ ] `npm run dev` starts the dashboard on a free port
- [ ] Login works against `localhost:8080` and stores tokens in localStorage
- [ ] Vehicle list renders; non-admin sees masked IDs
- [ ] Map shows last-known vehicle positions
- [ ] WebSocket connects, reconnects after kill, and updates live markers
- [ ] Offline banner appears when network is disabled; cached data remains visible
- [ ] `npm run test` passes with coverage for apiClient, masking, and store logic
- [ ] `docs/SETUP.md` includes frontend install and dev instructions
