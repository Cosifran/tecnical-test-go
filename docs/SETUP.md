# SETUP.md — IoT Fleet Monitoring System

## Prerequisites

- **Go 1.21+** (tested with Go 1.23+)
- **GCC/CGO** (required by the SQLite driver `mattn/go-sqlite3`)
- **Git**

## Quick Start

### 1. Clone the Repository

```bash
git clone <repo-url>
cd prueba-tecnica/backend
```

### 2. Set Up Environment Variables

```bash
cp .env.example .env
```

Edit `.env` and set `JWT_SECRET` to a secure random string (at least 32 characters):

```bash
# Generate a secure secret:
openssl rand -base64 32
```

Then update the `.env` file:

```
JWT_SECRET=<paste-your-secret-here>
```

### 3. Install Dependencies

```bash
go mod tidy
```

### 4. Seed the Database

The seed command creates two users and three vehicles:

| User | Email | Password | Role |
|------|-------|----------|------|
| Admin | `admin@example.com` | `admin123` | admin |
| User | `user@example.com` | `user123` | user |

| Vehicle | Device ID | Name |
|---------|-----------|------|
| Truck 01 | `DEV-11111111-AAAA` | Truck 01 |
| Truck 02 | `DEV-22222222-BBBB` | Truck 02 |
| Truck 03 | `DEV-33333333-CCCC` | Truck 03 |

```bash
# Make sure JWT_SECRET is set in your environment
export JWT_SECRET=$(grep JWT_SECRET .env | cut -d= -f2)

go run ./cmd/api -seed
```

You should see output like:

```
INFO seeded admin user id=... email=admin@example.com
INFO seeded regular user id=... email=user@example.com
INFO seeded vehicle id=... device_id=DEV-11111111-AAAA name=Truck 01
INFO seeded vehicle id=... device_id=DEV-22222222-BBBB name=Truck 02
INFO seeded vehicle id=... device_id=DEV-33333333-CCCC name=Truck 03
INFO database seeded successfully
```

### 5. Start the Server

```bash
export JWT_SECRET=$(grep JWT_SECRET .env | cut -d= -f2)
go run ./cmd/api
```

You should see:

```
INFO WebSocket Hub started
INFO server starting addr=:8080
```

### 6. Verify It Works

**Login as admin:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'
```

Expected response:

```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**List vehicles (save the access token from the login response):**

```bash
TOKEN="<paste-access-token-here>"

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/vehicles
```

**Ingest sensor data:**

```bash
curl -X POST http://localhost:8080/api/v1/sensors/data \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '[
    {"device_id":"DEV-11111111-AAAA","timestamp":"2024-01-15T10:30:00Z","type":"gps","value":{"lat":-34.6037,"lng":-58.3816}},
    {"device_id":"DEV-11111111-AAAA","timestamp":"2024-01-15T10:30:00Z","type":"fuel","value":{"level":45.5,"unit":"liters"}}
  ]'
```

Expected response:

```json
{"inserted": 2}
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific package
go test -v ./internal/infrastructure/jwt/...
go test -v ./internal/application/...
go test -v ./internal/infrastructure/websocket/...
```

## Running the Simulation Script

The simulation script authenticates as admin and sends batches of sensor data every 5 seconds.

```bash
export JWT_SECRET=$(grep JWT_SECRET .env | cut -d= -f2)

# Start the server in one terminal
go run ./cmd/api

# In another terminal, run the simulation
go run scripts/simulate.go
```

**With alert mode** (fuel levels drop faster to trigger `low_fuel` alerts):

```bash
go run scripts/simulate.go -alert-mode
```

**Custom options:**

```bash
go run scripts/simulate.go -server http://localhost:8080 -iterations 50 -alert-mode
```

## WebSocket Connection

Connect using a WebSocket client (e.g., `wscat`, Postman, browser):

```bash
# Install wscat
npm install -g wscat

# Connect with token (use the access token from login)
wscat -c "ws://localhost:8080/api/v1/ws?token=<your-access-token>"
```

The server broadcasts sensor data and alerts to all connected WebSocket clients. You'll see JSON messages like:

```json
{"type":"sensor_update","vehicle_id":"..."}
{"type":"low_fuel","vehicle_id":"..."}
```

## API Endpoints

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| POST | `/api/v1/auth/login` | No | - | Login with email+password |
| POST | `/api/v1/auth/refresh` | No | - | Refresh access token |
| GET | `/api/v1/vehicles` | Bearer | admin, user | List all vehicles |
| GET | `/api/v1/vehicles/{id}/history` | Bearer | admin, user | Get vehicle sensor history |
| POST | `/api/v1/sensors/data` | Bearer | admin | Ingest sensor data batch |
| GET | `/api/v1/alerts` | Bearer | admin | List all alerts |
| GET | `/api/v1/ws` | ?token= | admin, user | WebSocket real-time updates |

## Project Structure

```
backend/
├── cmd/api/main.go                        # Entrypoint + dependency wiring
├── internal/
│   ├── config/config.go                   # Environment variable configuration
│   ├── domain/
│   │   ├── entity.go                      # Domain entities (User, Vehicle, SensorData, Alert)
│   │   ├── repository.go                 # Repository interfaces
│   │   └── errors.go                      # Sentinel errors
│   ├── application/
│   │   ├── auth_service.go                # Login + refresh logic
│   │   ├── sensor_service.go             # Sensor data ingestion
│   │   ├── vehicle_service.go            # Vehicle listing + history
│   │   ├── fuel_service.go               # Fuel autonomy calculation
│   │   └── masking.go                    # Device ID masking
│   └── infrastructure/
│       ├── jwt/token.go                  # Manual JWT implementation
│       ├── persistence/
│       │   ├── db.go                     # SQLite connection + migrations
│       │   └── sqlite/                   # Repository implementations
│       ├── http/
│       │   ├── router.go                 # Route definitions
│       │   ├── middleware.go             # Auth, RBAC, logging middleware
│       │   ├── httputil/helpers.go       # Context keys, JSON helpers
│       │   └── handler/                  # HTTP handlers
│       └── websocket/
│           ├── hub.go                    # WebSocket Hub (broadcast manager)
│           └── client.go                # WebSocket Client (ReadPump + WritePump)
├── migrations/
│   └── 001_init.sql                     # Database schema
├── scripts/
│   └── simulate.go                      # Simulation script
├── .env.example                          # Environment variable template
├── go.mod
└── go.sum
```

## Frontend Web Dashboard

### Prerequisites

- **Node.js 18+** (tested with 20+)
- **npm** (comes with Node.js)
- Backend API running at `http://localhost:8080`

### Install Frontend Dependencies

```bash
cd frontend-web
npm install
```

### Start Development Server

```bash
cd frontend-web
npm run dev
```

The dev server starts at `http://localhost:5173` and proxies API/WS requests to the backend at `:8080`.

### Run Tests

```bash
cd frontend-web
npm test
```

For watch mode during development:

```bash
npm run test:watch
```

For coverage:

```bash
npm run test:coverage
```

### Build for Production

```bash
cd frontend-web
npm run build
```

Output goes to `frontend-web/dist/`. Serve with any static file server.

### Environment Variables

The frontend uses Vite's dev proxy to forward API and WebSocket requests to the backend. No environment variables are needed during development — the proxy handles `/api` and `/ws` routes automatically.

For production deployments, configure the API base URL via the `VITE_API_URL` environment variable.

### Full Stack Startup Sequence

1. **Seed the database** (first time only):

   ```bash
   cd backend
   export JWT_SECRET=$(grep JWT_SECRET .env | cut -d= -f2)
   go run ./cmd/api -seed
   ```

2. **Start the backend**:

   ```bash
   cd backend
   export JWT_SECRET=$(grep JWT_SECRET .env | cut -d= -f2)
   go run ./cmd/api
   ```

3. **Start the frontend** (in a separate terminal):

   ```bash
   cd frontend-web
   npm run dev
   ```

4. **Open browser**: `http://localhost:5173`

### Test Users

| Role  | Email               | Password  |
|-------|---------------------|-----------|
| Admin | `admin@example.com` | `admin123` |
| User  | `user@example.com`  | `user123`  |

> **Note**: Admin sees raw device IDs and the Alerts page. Regular users see masked IDs (`DEV-****-XXXX`) and cannot access `/alerts`.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `JWT_SECRET must be at least 32 characters` | Set `JWT_SECRET` in `.env` or environment to a string of 32+ characters |
| `CGO_ENABLED=0` error with SQLite | Install GCC and run `CGO_ENABLED=1 go run ./cmd/api` |
| `database is locked` | This shouldn't happen with WAL mode. If it does, close other connections to the DB file |
| WebSocket 401 errors | Make sure you're passing the access token (not refresh token) as `?token=` |
| Port already in use | Change `PORT` in `.env` or kill the process using port 8080 |
| `npm install` fails in frontend-web | Ensure Node.js 18+ is installed (`node -v`). Delete `node_modules` and retry |
| Vite proxy 404s to backend | Make sure the backend is running at `:8080` before starting the frontend |
| MapLibre tiles don't load offline | Expected — map tiles require network. The map shows a placeholder grid when offline |