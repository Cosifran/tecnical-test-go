# Design: Frontend Web Dashboard

## Technical Approach

Vite 5 SPA with React 19, TypeScript strict, Tailwind 4, Zustand 5 (persist), MapLibre GL, and Recharts. The backend at `:8080` is a separate origin; SSR adds no value. The frontend is a fully client-rendered dashboard that authenticates via JWT, consumes REST + WebSocket, caches fleet data in localStorage via Zustand persist, and degrades gracefully offline.

## Architecture Decisions

### Decision: Vite SPA over Next.js App Router

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Vite SPA | Simple mental model, fast HMR, clean offline story, no SSR complexity | **Chosen** |
| Next.js 15 | Server components, API routes, streaming | Rejected — backend is separate origin, SSR provides no benefit for a dashboard that requires auth |

### Decision: Zustand 5 slices with persist

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Zustand 5 slices + persist | 3KB, built-in localStorage persist, `useShallow` selectors, clean slice pattern | **Chosen** |
| React Context | Zero-cost but no persist, no selectors, nested providers | Rejected |
| Redux Toolkit | Powerful but heavy for this scope | Rejected |

Three slices: `authStore` (persisted), `fleetStore` (persisted, recent history only), `wsStore` (ephemeral, connection status + buffer).

### Decision: Native fetch wrapper (apiClient)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Custom `apiClient` module | Zero deps, full control over retry/refresh logic, easy to test | **Chosen** |
| Axios | Interceptors are nice, but 14KB for one login form | Rejected |
| TanStack Query | Caching + retry built-in, but adds complex cache invalidation for WS-driven data | Deferred to future |

The apiClient is a thin `fetch` wrapper: injects `Authorization: Bearer`, catches 401, refreshes once via `POST /api/v1/auth/refresh`, retries the original request. If refresh fails, clear auth and redirect to `/login`.

### Decision: Singleton WebSocket manager

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `WebSocketManager` class + `useWebSocket` hook | Singleton ensures one connection; hook subscribes to events via EventEmitter pattern; testable with mock WS | **Chosen** |
| React-only hook | Reconnect state lost on unmount; harder to test | Rejected |

The manager handles: connect with `?token=`, exponential backoff (initial 1s, max 30s, factor 2), message dispatch, disconnect, and reconnect. The hook bridges it to React state.

### Decision: react-router-dom v7

| Option | Tradeoff | Decision |
|--------|----------|----------|
| react-router-dom v7 | Industry standard, data routers, lazy routes, layout routes | **Chosen** |
| TanStack Router | Type-safe routes but newer ecosystem | Rejected — familiarity wins for 3-day test |

### Decision: localStorage via Zustand persist (not IndexedDB)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| localStorage + persist | ~5MB cap (sufficient for vehicle list + last 50 history points per vehicle), sync API, built-in Zustand support | **Chosen** |
| IndexedDB | Unlimited but async API, no built-in Zustand middleware | Rejected for v1 |

Cap: `fleetStore` persists at most 50 history points per vehicle. On overflow, oldest points are evicted.

## Data Flow

### Auth

```
Login form → apiClient.post('/auth/login', {email, password})
           → authStore.login(tokens)
           → persist to localStorage
           → decode JWT payload (base64) → extract role
           → redirect to /vehicles
```

### Token refresh on 401

```
apiClient request → 401 response
                  → authStore.refresh()
                  → POST /api/v1/auth/refresh with refreshToken
                  → success: update tokens, retry original request (once)
                  → failure: authStore.logout(), redirect /login
```

### Vehicle list

```
VehicleList mount → fleetStore.fetchVehicles()
                 → GET /api/v1/vehicles (Bearer token)
                 → store in fleetStore.vehicles
                 → component re-renders via useShallow selector
                 → client-side masking: if role !== 'admin', mask deviceId
```

### WebSocket

```
authStore.authenticated → WebSocketManager.connect(url, token)
                        → onopen: wsStore.connected = true
                        → onmessage: dispatch by type
                          sensor_update → merge into fleetStore
                          low_fuel → if admin, show toast
                        → onclose/onerror: wsStore.connected = false, start backoff
                        → reconnect → wsStore.reconnectAttempt++
                        → flushed pending messages if any
```

### Offline

```
navigator.onLine changes → useOnline hook
                         → false: show OfflineBanner, serve fleetStore cache with stale timestamp
                         → true: hide banner, refetch vehicles, reconnect WS
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `frontend-web/package.json` | Create | Dependencies: react ^19.0.0, react-dom ^19.0.0, react-router-dom ^7.0.0, zustand ^5.0.0, **react-map-gl ^8.1.1**, **maplibre-gl ^5.24.0** (NOT 5.17.0 — Babel regression), recharts ^2.0.0, clsx, tailwind-merge |
| `frontend-web/vite.config.ts` | Create | Dev proxy `/api` → `http://localhost:8080`, WS proxy. **Note:** MapLibre 5+ is pure ESM — no `optimizeDeps.include` needed. Pin vite@^7.0.0 (avoid Vite 8.0.x GeoJSON diff bug). |
| `frontend-web/tsconfig.json` | Create | strict: true, paths: `@/*` → `src/*` |
| `frontend-web/tailwind.config.ts` | Create | Tailwind 4 config with theme variables |
| `frontend-web/src/main.tsx` | Create | App entry, router, providers |
| `frontend-web/src/App.tsx` | Create | Layout shell with Outlet |
| `frontend-web/src/stores/authStore.ts` | Create | Auth slice with persist (tokens, role, login, logout, refresh) |
| `frontend-web/src/stores/fleetStore.ts` | Create | Fleet slice with persist (vehicles, history cache, fetch actions) |
| `frontend-web/src/stores/wsStore.ts` | Create | WS slice (connected, reconnectAttempt, pendingMessages) |
| `frontend-web/src/api/apiClient.ts` | Create | Fetch wrapper with Bearer injection, 401 retry, typed responses |
| `frontend-web/src/api/endpoints.ts` | Create | Typed API functions (login, refreshToken, getVehicles, getHistory, getAlerts) |
| `frontend-web/src/websocket/WebSocketManager.ts` | Create | Singleton WS class with reconnect, event dispatch |
| `frontend-web/src/websocket/useWebSocket.ts` | Create | React hook bridging WS manager to wsStore |
| `frontend-web/src/hooks/useOnline.ts` | Create | navigator.onLine listener returning boolean |
| `frontend-web/src/hooks/useAuth.ts` | Create | Convenience wrapper around authStore selectors |
| `frontend-web/src/utils/masking.ts` | Create | Client-side maskDeviceId function mirroring backend logic |
| `frontend-web/src/utils/jwt.ts` | Create | Decode JWT payload (base64) to extract role, without external lib |
| `frontend-web/src/pages/LoginPage.tsx` | Create | Login form with useActionState |
| `frontend-web/src/pages/VehicleListPage.tsx` | Create | Table of vehicles with RBAC-aware masking |
| `frontend-web/src/pages/VehicleDetailPage.tsx` | Create | Map + charts + history for one vehicle |
| `frontend-web/src/pages/AlertsPage.tsx` | Create | Admin-only alerts list with route guard |
| `frontend-web/src/components/Layout.tsx` | Create | Sidebar + header shell |
| `frontend-web/src/components/OfflineBanner.tsx` | Create | Banner shown when navigator.onLine is false |
| `frontend-web/src/components/VehicleCard.tsx` | Create | Reusable vehicle row/card |
| `frontend-web/src/components/MapView.tsx` | Create | MapLibre GL map with vehicle markers |
| `frontend-web/src/components/ChartView.tsx` | Create | Recharts wrapper for fuel/temperature |
| `frontend-web/src/components/Toast.tsx` | Create | Toast notification for low_fuel alerts |
| `frontend-web/src/guards/AuthGuard.tsx` | Create | Redirect to /login if not authenticated |
| `frontend-web/src/guards/AdminGuard.tsx` | Create | Redirect to /vehicles if role !== admin |
| `frontend-web/src/types/api.ts` | Create | API request/response TypeScript interfaces |
| `frontend-web/src/types/domain.ts` | Create | Domain types mirroring backend |
| `frontend-web/src/types/ws.ts` | Create | WebSocket message types |
| `frontend-web/src/styles/globals.css` | Create | Tailwind 4 directives + theme variables |
| `frontend-web/src/__tests__/apiClient.test.ts` | Create | Unit tests for 401 retry, token refresh |
| `frontend-web/src/__tests__/masking.test.ts` | Create | Unit tests for device ID masking |
| `frontend-web/src/__tests__/jwt.test.ts` | Create | Unit tests for JWT payload decode |
| `frontend-web/src/__tests__/stores/authStore.test.ts` | Create | Auth store unit tests |
| `frontend-web/vitest.config.ts` | Create | Vitest config with jsdom environment |

## Interfaces / Contracts

### Domain Types (`src/types/domain.ts`)

```typescript
const VEHICLE_STATUS = {
  ACTIVE: "active",
  INACTIVE: "inactive",
  MAINTENANCE: "maintenance",
} as const;

type VehicleStatus = (typeof VEHICLE_STATUS)[keyof typeof VEHICLE_STATUS];

interface Vehicle {
  id: string;
  device_id: string;
  name: string;
  status: VehicleStatus;
  last_seen: string;
  fuel_level: number;
  temperature: number;
  latitude: number;
  longitude: number;
  created_at: string;
}

interface SensorHistoryPoint {
  timestamp: string;
  fuel_level: number;
  temperature: number;
}

interface Alert {
  id: string;
  vehicle_id: string;
  type: string;
  severity: string;
  details: Record<string, unknown>;
  created_at: string;
}
```

### API Types (`src/types/api.ts`)

```typescript
interface LoginRequest {
  email: string;
  password: string;
}

interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: "Bearer";
  expires_in: number;
}

interface RefreshRequest {
  refresh_token: string;
}

interface VehiclesResponse {
  vehicles: Vehicle[];
}

interface HistoryResponse {
  vehicle: Vehicle;
  history: SensorData[];
}

interface AlertsResponse {
  alerts: Alert[];
}

interface ApiError {
  error: string;
  message: string;
}
```

### WebSocket Types (`src/types/ws.ts`)

```typescript
interface SensorUpdateMessage {
  type: "sensor_update";
  vehicle_id: string;
}

interface LowFuelMessage {
  type: "low_fuel";
  vehicle_id: string;
  estimated_minutes: number;
}

type WsMessage = SensorUpdateMessage | LowFuelMessage;
```

### Store Types

```typescript
// authStore
interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  role: "admin" | "user" | null;
  email: string | null;
  userId: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refresh: () => Promise<boolean>;
}

// fleetStore
interface FleetState {
  vehicles: Vehicle[];
  selectedVehicleId: string | null;
  history: Record<string, SensorHistoryPoint[]>;
  alerts: Alert[];
  lastFetched: number | null;
  isLoading: boolean;
  error: string | null;
  fetchVehicles: () => Promise<void>;
  fetchHistory: (vehicleId: string) => Promise<void>;
  fetchAlerts: () => Promise<void>;
  selectVehicle: (id: string | null) => void;
  updateVehiclePosition: (vehicleId: string, lat: number, lng: number) => void;
}

// wsStore
interface WsState {
  connected: boolean;
  reconnectAttempt: number;
  lastMessage: WsMessage | null;
}
```

## Key Patterns

### RBAC Enforcement

- **Route guards**: `AuthGuard` redirects unauthenticated users to `/login`. `AdminGuard` redirects non-admins to `/vehicles`. Applied to `/alerts` route.
- **Conditional UI**: `maskDeviceId()` is called in `VehicleListPage` and `VehicleDetailPage` based on `authStore.role`.
- **Server validation**: Backend enforces RBAC; frontend gates are UX-only.

### Device ID Masking (`src/utils/masking.ts`)

```typescript
function maskDeviceId(raw: string): string {
  if (raw.length < 4) return "DEV-****-????";
  return `DEV-****-${raw.slice(-4)}`;
}
```

Mirrors the backend `MaskDeviceID` function exactly. Applied in render, not in the store — raw IDs are always stored.

### Token Refresh on 401 (apiClient)

```typescript
let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

async function apiClient<T>(url: string, options: RequestInit = {}): Promise<T> {
  const response = await fetchWithAuth(url, options);
  if (response.status !== 401) return handleResponse<T>(response);

  // Refresh once
  if (!isRefreshing) {
    isRefreshing = true;
    refreshPromise = authStore.refresh();
  }
  const refreshed = await refreshPromise;
  isRefreshing = false;
  refreshPromise = null;

  if (!refreshed) {
    authStore.logout();
    window.location.href = "/login";
    throw new Error("Session expired");
  }

  // Retry original request with new token
  return fetchWithAuth(url, options).then(r => handleResponse<T>(r));
}
```

Queue pattern: concurrent 401s share a single refresh call.

### WebSocket Reconnect

```typescript
class WebSocketManager {
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempt = 0;
  private maxReconnectDelay = 30_000; // 30s

  private scheduleReconnect(): void {
    const delay = Math.min(1000 * 2 ** this.reconnectAttempt, this.maxReconnectDelay);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempt++;
      this.connect();
    }, delay);
  }

  private onOpen(): void {
    this.reconnectAttempt = 0;
    // ...
  }
}
```

### Offline Cache Eviction

`fleetStore` persists via Zustand `persist` with a `partialize` option that caps history to 50 points per vehicle:

```typescript
persist(fleetSlice, {
  name: "fleet-cache",
  partialize: (state) => ({
    vehicles: state.vehicles,
    history: Object.fromEntries(
      Object.entries(state.history).map(([id, points]) => [
        id,
        points.slice(-50), // keep last 50 points per vehicle
      ])
    ),
    lastFetched: state.lastFetched,
  }),
}),
```

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `apiClient` 401 retry + refresh queue | Mock `globalThis.fetch`; assert retry-once behavior, queue concurrency |
| Unit | `maskDeviceId` function | Mirror backend tests: short strings, normal IDs, edge cases |
| Unit | JWT payload decode (`decodePayload`) | Test base64 decode, type extraction, malformed input |
| Unit | Store logic (auth login/logout, fleet fetch) | Mock apiClient; assert state transitions |
| Component | `LoginPage` | Render, fill form, mock apiClient, assert redirect |
| Component | `VehicleListPage` | Mock fleetStore, verify masked vs raw IDs |
| Integration | WebSocket reconnect | Mock WebSocket; verify backoff timing and reconnect count |
| E2E (stretch) | Login → vehicle list → WS update | Playwright if time permits |

Mock strategy: Vitest `vi.fn()` for `fetch`; custom `MockWebSocket` class for WS tests.

## Build & Dev Configuration

### vite.config.ts

```typescript
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/api/v1/ws": {
        target: "ws://localhost:8080",
        ws: true,
      },
    },
  },
});
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "paths": { "@/*": ["./src/*"] }
  },
  "include": ["src"]
}
```

### Tailwind 4 Conventions

- **Never** `var()` or hex in className — use semantic Tailwind classes `bg-primary`, `text-slate-400`.
- **Never** `useMemo` / `useCallback` (React Compiler handles it).
- Dynamic values → `style` prop. Conditional classes → `cn()` from `src/utils/cn.ts`.
- Recharts color constants use `var(--color-*)` in the `style` prop, NOT in className.

## Resolved Risks

- ✅ **react-map-gl React 19 compatibility**: `react-map-gl@^8.1.1` confirmed working with React 19 in Vite SPAs. Import via `react-map-gl/maplibre` subpath. Open React 19 issues are Next.js-specific edge cases only.
- ✅ **MapLibre + Vite ESM**: `maplibre-gl@^5.24.0` is pure ESM, no `optimizeDeps` needed. Avoid `5.17.0` (Babel regression). Avoid `vite@8.0.x` (GeoJSON diff bug). Use `vite@^7.0.0`.
- ✅ **CORS**: Backend CORS middleware is enabled (`Access-Control-Allow-Origin: *`).
- ✅ **WebSocket broadcast**: Backend broadcasts `sensor_update` after successful ingest.

## Open Questions

- [ ] Decide whether `AlertsPage` fetches on mount or subscribes to WS-only alerts — spec says GET `/api/v1/alerts` exists, so fetch on mount is simpler.