# Tasks: Frontend Web Dashboard

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,500 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-always |
| Suggested split | 6 PRs (one per batch) |
| Decision needed before apply | Yes |
| Chain strategy | pending |

## Suggested Work Units

| Unit | Goal | PR | Notes |
|------|------|-----|-------|
| 1 | Scaffold: Vite+TS+Tailwind+router | PR 1 | Base: main. ~150 lines, self-contained |
| 2 | Types, auth, login | PR 2 | Base: PR 1. ~350 lines |
| 3 | Fleet store, vehicle pages, map | PR 3 | Base: PR 2. ~400 lines |
| 4 | WebSocket, real-time, offline | PR 4 | Base: PR 3. ~230 lines |
| 5 | Alerts page + polish | PR 5 | Base: PR 4. ~60 lines |
| 6 | Tests + docs | PR 6 | Base: PR 5. ~300 lines |

## Batch 1: Scaffold

- [x] T1.1 Create `frontend-web/` with `package.json` (react 19, react-dom 19, react-router-dom 7, zustand 5, react-map-gl 8, maplibre-gl 5.24, recharts 2, clsx, tailwind-merge, vitest, @testing-library/react). *Files: package.json*
- [x] T1.2 Create `vite.config.ts` with React plugin, `@/` alias, `/api` proxy to `localhost:8080`, WS proxy. Pin vite@^7.0.0. *Files: vite.config.ts*
- [x] T1.3 Create `tsconfig.json` with `strict: true`, `noUncheckedIndexedAccess`, `paths: {"@/*": ["./src/*"]}`. *Files: tsconfig.json*
- [x] T1.4 Create `vitest.config.ts` with jsdom environment and `@/` alias. *Files: vitest.config.ts*
- [x] T1.5 Create `src/styles/globals.css` with Tailwind 4 `@import "tailwindcss"` directive. *Files: src/styles/globals.css*
- [x] T1.6 Create `src/utils/cn.ts` (clsx + tailwind-merge wrapper). *Files: src/utils/cn.ts*
- [x] T1.7 Create `src/main.tsx` with `createBrowserRouter`, route definitions, `RouterProvider`. *Files: src/main.tsx*
- [x] T1.8 Create `src/App.tsx` as layout shell rendering `<Outlet />`. *Files: src/App.tsx*
- [x] T1.9 Verify: `npm run dev` starts, shows blank page at `localhost:5173`. *AC: dev server boots, no TS errors.*

## Batch 2: Auth & API Foundation

- [x] T2.1 Create flat TypeScript interfaces in `src/types/domain.ts` (Vehicle, SensorHistoryPoint, Alert, VehicleStatus const+type). *Files: src/types/domain.ts*
- [x] T2.2 Create API types in `src/types/api.ts` (LoginRequest, LoginResponse, VehiclesResponse, HistoryResponse, AlertsResponse, ApiError). *Files: src/types/api.ts*
- [x] T2.3 Create WS message types in `src/types/ws.ts` (SensorUpdateMessage, LowFuelMessage, WsMessage union). *Files: src/types/ws.ts*
- [x] T2.4 Implement `src/utils/jwt.ts` — decode JWT payload from base64 without external lib; extract role and email. *Files: src/utils/jwt.ts*
- [x] T2.5 Implement `src/utils/masking.ts` — `maskDeviceId(raw)` returns `DEV-****-{last4}`. *Files: src/utils/masking.ts*
- [x] T2.6 Implement `src/api/apiClient.ts` — fetch wrapper injecting Bearer token, catching 401, refreshing once with queue dedup, retrying. *Files: src/api/apiClient.ts*
- [x] T2.7 Implement `src/api/endpoints.ts` — typed functions: login, refreshToken, getVehicles, getHistory, getAlerts. *Files: src/api/endpoints.ts*
- [x] T2.8 Implement `src/stores/authStore.ts` — Zustand slice with persist: tokens, role, login/logout/refresh actions. *Files: src/stores/authStore.ts*
- [x] T2.9 Create `src/hooks/useAuth.ts` — convenience wrapper around authStore selectors. *Files: src/hooks/useAuth.ts*
- [x] T2.10 Create `src/guards/AuthGuard.tsx` — redirects to `/login` if not authenticated. *Files: src/guards/AuthGuard.tsx*
- [x] T2.11 Create `src/guards/AdminGuard.tsx` — redirects to `/vehicles` if role !== admin. *Files: src/guards/AdminGuard.tsx*
- [x] T2.12 Implement `src/pages/LoginPage.tsx` — useActionState form, email+password, calls authStore.login, redirects to `/vehicles`. *Files: src/pages/LoginPage.tsx*
- [x] T2.13 Create `src/components/Layout.tsx` — header + sidebar with role-aware nav links + `<Outlet />`. *Files: src/components/Layout.tsx*
- [x] T2.14 Verify: login flow works against backend, tokens persisted in localStorage, role decoded. *AC: login → stored tokens → redirect to /vehicles.*

## Batch 3: Fleet Store & Vehicle Pages

- [x] T3.1 Implement `src/stores/fleetStore.ts` — Zustand slice with persist (partialize: cap 50 history points/vehicle), fetchVehicles, fetchHistory, selectVehicle, updateVehiclePosition. *Files: src/stores/fleetStore.ts*
- [x] T3.2 Create `src/components/VehicleCard.tsx` — reusable row: status badge, fuel %, temperature, masked device ID (when role !== admin). *Files: src/components/VehicleCard.tsx*
- [x] T3.3 Implement `src/pages/VehicleListPage.tsx` — renders fleetStore.vehicles via useShallow selector, wraps with AuthGuard, calls maskDeviceId for non-admins. *Files: src/pages/VehicleListPage.tsx*
- [x] T3.4 Create `src/components/MapView.tsx` — react-map-gl/maplibre map, markers for vehicles, click-to-select. *Files: src/components/MapView.tsx*
- [x] T3.5 Create `src/components/ChartView.tsx` — Recharts line chart for fuel/temperature history; color constants via style props (var()). *Files: src/components/ChartView.tsx*
- [x] T3.6 Implement `src/pages/VehicleDetailPage.tsx` — map + chart + history for one vehicle, raw/masked device ID by role. *Files: src/pages/VehicleDetailPage.tsx*
- [x] T3.7 Verify: admin sees raw IDs, user sees masked IDs; map renders markers; chart renders history. *AC: RBAC masking correct, map + chart visible.*

## Batch 4: Real-time & Offline

- [x] T4.1 Implement `src/stores/wsStore.ts` — ephemeral Zustand slice: connected, reconnectAttempt, lastMessage. *Files: src/stores/wsStore.ts*
- [x] T4.2 Implement `src/websocket/WebSocketManager.ts` — singleton class: connect with token param, onmessage dispatch by type, exponential backoff reconnect (1s–30s). *Files: src/websocket/WebSocketManager.ts*
- [x] T4.3 Implement `src/websocket/useWebSocket.ts` — React hook bridging manager to wsStore; subscribes to sensor_update (merge into fleetStore) and low_fuel (toast for admins). *Files: src/websocket/useWebSocket.ts*
- [x] T4.4 Implement `src/hooks/useOnline.ts` — navigator.onLine listener returning boolean. *Files: src/hooks/useOnline.ts*
- [x] T4.5 Create `src/components/OfflineBanner.tsx` — persistent banner when offline, disappears on reconnect. *Files: src/components/OfflineBanner.tsx*
- [x] T4.6 Create `src/components/Toast.tsx` — toast notification component for low_fuel alerts (admin-only). *Files: src/components/Toast.tsx*
- [x] T4.7 Verify: WS connects, reconnects after server restart, offline banner shows on disconnect, cached data survives reload. *AC: reconnect within 30s, offline banner visible, toast for admin.*

## Batch 5: Alerts & Polish

- [x] T5.1 Implement `src/pages/AlertsPage.tsx` — admin-only route, fetches GET /api/v1/alerts, renders alert list with severity badges. *Files: src/pages/AlertsPage.tsx, src/stores/fleetStore.ts (added alerts state/actions)*
- [x] T5.2 Wire AlertsPage route behind AdminGuard in router. *Files: src/main.tsx*
- [x] T5.3 Verify: non-admin redirected to /vehicles; admin sees alert list. *AC: RBAC enforced, alerts render correctly.*
- [x] T5.4 Create `src/components/LoadingSpinner.tsx` — reusable loading spinner with sm/md/lg sizes and optional label.
- [x] T5.5 Refactor VehicleListPage to use LoadingSpinner instead of inline loading text.

## Batch 6: Tests & Docs

- [x] T6.1 Write `src/__tests__/apiClient.test.ts` — mock fetch, verify 401 triggers refresh-once, queue dedup, retry, and logout on refresh failure. *Files: src/__tests__/apiClient.test.ts*
- [x] T6.2 Write `src/__tests__/masking.test.ts` — test short IDs, normal IDs, edge cases matching backend mask pattern. *Files: src/__tests__/masking.test.ts*
- [x] T6.3 Write `src/__tests__/jwt.test.ts` — test valid decode, tampered payload, malformed input. *Files: src/__tests__/jwt.test.ts*
- [x] T6.4 Write `src/__tests__/stores/authStore.test.ts` — test login/logout state transitions, token persistence, token refresh. *Files: src/__tests__/stores/authStore.test.ts*
- [x] T6.5 Update `docs/SETUP.md` with frontend install, dev, and test instructions. *Files: docs/SETUP.md*
- [x] T6.6 Verify: `npm run test` passes all suites; `npm run dev` serves full dashboard. *AC: all tests green, full user flow functional.*
