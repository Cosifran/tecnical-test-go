# Frontend Web Dashboard Specification

## Capabilities

### vehicle-dashboard
**Purpose:** List, detail, map, and chart views for fleet data.

#### Requirements

| ID | Requirement | Type |
|---|---|---|
| VD-01 | The system MUST render a paginated vehicle list with columns: ID, status, last seen, fuel %, temperature. | Functional |
| VD-02 | The system MUST mask device IDs for non-admin users as `DEV-****-{last4}`. | Functional |
| VD-03 | The system MUST show a vehicle detail view with an interactive map and historical sensor charts. | Functional |
| VD-04 | Map tiles MUST degrade gracefully when offline. | Non-functional |
| VD-05 | Chart renders SHOULD complete within 100ms for 500 data points. | Non-functional |

#### Scenarios

**Admin views vehicle list**
- GIVEN an authenticated admin
- WHEN they navigate to `/vehicles`
- THEN they see all vehicles with raw device IDs

**User views masked list**
- GIVEN an authenticated non-admin user
- WHEN they navigate to `/vehicles`
- THEN they see device IDs in `DEV-****-{last4}` format

**Admin opens vehicle detail**
- GIVEN an admin on the vehicle list
- WHEN they click a vehicle row
- THEN they navigate to `/vehicles/:id` and see a map with the last location and a fuel/temp chart

### auth-session
**Purpose:** Login, token refresh, role-aware navigation.

#### Requirements

| ID | Requirement | Type |
|---|---|---|
| AS-01 | The system MUST provide a login form submitting to `POST /api/auth/login`. | Functional |
| AS-02 | The system MUST store access and refresh tokens in `localStorage` via Zustand persist. | Functional |
| AS-03 | The system MUST attach `Authorization: Bearer <token>` to every API request. | Functional |
| AS-04 | The system MUST auto-refresh the access token once on 401 and retry the original request. | Functional |
| AS-05 | The system MUST redirect to `/login` if token refresh fails. | Functional |
| AS-06 | The system MUST derive the user role from the JWT payload. | Functional |
| AS-07 | The system MUST implement route guards redirecting unauthenticated users to login. | Functional |
| AS-08 | Password fields MUST mask input. | Non-functional |

#### Scenarios

**User logs in**
- GIVEN a user on `/login`
- WHEN they submit valid credentials
- THEN the system stores tokens and redirects to `/vehicles`

**Token expires during API call**
- GIVEN an authenticated user with an expired access token
- WHEN they trigger an API call
- THEN the system refreshes the token and retries transparently

**Refresh token invalid**
- GIVEN a user with an expired refresh token
- WHEN they trigger an API call
- THEN the system clears auth state and redirects to `/login`

**User accesses Alerts page**
- GIVEN a non-admin user
- WHEN they navigate to `/alerts`
- THEN they are redirected to `/vehicles`

### real-time-updates
**Purpose:** WebSocket connection, live sensor broadcasts, reconnect logic.

#### Requirements

| ID | Requirement | Type |
|---|---|---|
| RT-01 | The system MUST open a WebSocket connection to `/ws` using the access token. | Functional |
| RT-02 | The system MUST implement exponential backoff reconnect with max delay 30s. | Functional |
| RT-03 | On `sensor_update`, the system MUST merge the payload into the fleet cache and update the map marker. | Functional |
| RT-04 | On `low_fuel`, the system MUST show a toast notification ONLY for admins. | Functional |
| RT-05 | The system MUST buffer incoming WS messages during reconnect and apply them once connected. | Functional |
| RT-06 | WS connection MUST not block the UI thread. | Non-functional |

#### Scenarios

**Live sensor update moves marker**
- GIVEN an open WS connection and a vehicle on the map
- WHEN a `sensor_update` arrives with new coordinates
- THEN the map marker animates to the new position

**Admin receives low fuel alert**
- GIVEN an admin with an open WS connection
- WHEN a `low_fuel` message arrives
- THEN a toast appears with the vehicle ID and estimated autonomy

**WebSocket reconnects after server restart**
- GIVEN a connected dashboard
- WHEN the server restarts and the connection drops
- THEN the system reconnects automatically and resumes live updates

### offline-cache
**Purpose:** Persisted fleet state, stale indicators, history replay.

#### Requirements

| ID | Requirement | Type |
|---|---|---|
| OC-01 | The system MUST persist the vehicle list and recent history to `localStorage` via Zustand persist. | Functional |
| OC-02 | When `navigator.onLine` is false, the system MUST show a persistent offline banner. | Functional |
| OC-03 | The system MUST continue serving cached data when offline. | Functional |
| OC-04 | The system MUST display a "last updated" timestamp on cached data. | Functional |
| OC-05 | When connectivity returns, the system MUST hide the banner and attempt WS reconnect. | Functional |
| OC-06 | The system MUST gracefully handle `localStorage` being full or disabled. | Functional |

#### Scenarios

**Network goes offline**
- GIVEN a user viewing the vehicle list
- WHEN the network disconnects
- THEN an offline banner appears and the cached list remains visible

**Network comes back online**
- GIVEN a user in offline mode
- WHEN the network reconnects
- THEN the banner hides, API calls resume, and the WS reconnects

## Edge Cases & Error Handling

| Case | Behavior |
|---|---|
| Empty vehicle list | Show empty state message; do not crash |
| WS connection fails permanently | Show "Realtime unavailable" badge; allow manual reconnect |
| Invalid/expired token + refresh fails | Clear auth state, redirect to `/login` |
| Map tiles fail to load | Show placeholder grid; do not crash |
| API returns 500 | Show generic error toast; allow retry |
| `localStorage` full/disabled | Disable persist middleware; show warning banner |
| Browser lacks WebSocket support | Show "Realtime unavailable" badge; polling fallback OPTIONAL |

## Data Contracts

### API Types

```typescript
interface LoginRequest { email: string; password: string; }
interface LoginResponse { accessToken: string; refreshToken: string; }

interface Vehicle {
  id: string;
  deviceId: string;
  status: "active" | "inactive" | "maintenance";
  lastSeen: string;
  fuelLevel: number;
  temperature: number;
  latitude: number;
  longitude: number;
}

interface SensorHistoryPoint {
  timestamp: string;
  fuelLevel: number;
  temperature: number;
}

interface ApiError { error: string; code: string; }
```

### Zustand Store State

```typescript
interface AuthSlice {
  accessToken: string | null;
  refreshToken: string | null;
  role: "admin" | "user" | null;
  isAuthenticated: boolean;
  login: (c: LoginRequest) => Promise<void>;
  logout: () => void;
}

interface FleetSlice {
  vehicles: Vehicle[];
  selectedVehicle: Vehicle | null;
  history: Record<string, SensorHistoryPoint[]>;
  lastFetched: number | null;
  fetchVehicles: () => Promise<void>;
  fetchHistory: (id: string) => Promise<void>;
}

interface WsSlice {
  connected: boolean;
  reconnectAttempt: number;
  pendingMessages: WsMessage[];
}

type Store = AuthSlice & FleetSlice & WsSlice;
```

### WebSocket Messages

```typescript
type WsMessage =
  | { type: "sensor_update"; payload: { deviceId: string; latitude: number; longitude: number; fuelLevel: number; temperature: number } }
  | { type: "low_fuel"; payload: { deviceId: string; estimatedMinutes: number } };
```

### Component Props

```typescript
interface VehicleListProps {
  vehicles: Vehicle[];
  role: "admin" | "user";
  onSelect: (id: string) => void;
}

interface MapViewProps {
  vehicles: Vehicle[];
  selectedId?: string;
}

interface OfflineBannerProps {
  isOnline: boolean;
  lastUpdated: number | null;
}
```

## RBAC Matrix

| Feature | Admin | User |
|---|---|---|
| `/vehicles` list | ✅ Raw IDs | ✅ Masked IDs |
| `/vehicles/:id` detail | ✅ Map + Charts | ✅ Map + Charts |
| `/alerts` | ✅ Full access | ❌ Redirect to `/vehicles` |
| `low_fuel` toast | ✅ Shown | ❌ Hidden |
| Sensor history | ✅ Full history | ✅ Full history |
| Device ID in API | ✅ Raw | ✅ Raw (masking is UI-only) |

## Acceptance Criteria

- [ ] Admin sees raw device IDs; users see masked IDs.
- [ ] Login stores tokens; 401 triggers refresh-and-retry; refresh failure redirects to login.
- [ ] WebSocket connects, reconnects, and updates map markers live.
- [ ] Admin sees `low_fuel` toasts; users do not.
- [ ] Offline banner shows on disconnect; cached data survives reload.
- [ ] All critical paths have corresponding unit tests.
