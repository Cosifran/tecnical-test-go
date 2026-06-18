# Backend Go Specification

## Purpose

Specification for the Go backend of the IoT fleet monitoring system. Covers authentication, sensor data ingestion, predictive fuel alerts, device ID masking, real-time updates, and testing requirements.

## Functional Requirements

### Requirement: Manual JWT Authentication

The system MUST implement JWT authentication without external JWT libraries. Token generation and validation MUST use only Go standard library crypto primitives (`crypto/hmac`, `crypto/sha256`, `encoding/base64`). Tokens MUST use the `HS256` algorithm. Access tokens MUST expire after 15 minutes. Refresh tokens MUST expire after 7 days.

#### Scenario: User logs in with valid credentials

- GIVEN a registered user with email `user@example.com` and password `correctpassword`
- WHEN the client sends a `POST /api/v1/auth/login` request with `{ "email": "user@example.com", "password": "correctpassword" }`
- THEN the system responds with `200 OK` and a JSON body containing `access_token`, `refresh_token`, and `token_type: "Bearer"`
- AND the access token MUST contain claims: `sub` (user ID), `email`, `role`, `iat`, `exp`

#### Scenario: User provides invalid credentials

- GIVEN a registered user with email `user@example.com`
- WHEN the client sends a `POST /api/v1/auth/login` request with `{ "email": "user@example.com", "password": "wrongpassword" }`
- THEN the system responds with `401 Unauthorized` and a JSON body `{ "error": "invalid_credentials" }`

#### Scenario: Token validation succeeds

- GIVEN a valid unexpired access token in the `Authorization: Bearer <token>` header
- WHEN the client sends any authenticated request
- THEN the system MUST validate the token signature using HMAC-SHA256 with the configured secret
- AND the request MUST proceed to the handler with the user context populated

#### Scenario: Token validation fails due to tampering

- GIVEN an access token with an altered payload segment
- WHEN the client sends a request with this token
- THEN the system responds with `401 Unauthorized` and `{ "error": "invalid_token" }`

#### Scenario: Token validation fails due to expiration

- GIVEN an expired access token
- WHEN the client sends a request with this token
- THEN the system responds with `401 Unauthorized` and `{ "error": "token_expired" }`

#### Scenario: Token refresh

- GIVEN a valid unexpired refresh token
- WHEN the client sends a `POST /api/v1/auth/refresh` request with `{ "refresh_token": "<token>" }`
- THEN the system responds with `200 OK` and a new access token
- AND the old refresh token MUST remain valid until its own expiration

### Requirement: Role-Based Access Control

The system MUST support two roles: `admin` and `user`. Admin role MUST grant full access to all endpoints and unmasked device IDs. User role MUST receive masked device IDs and MUST NOT receive predictive fuel alerts.

#### Scenario: Admin accesses vehicle list

- GIVEN an authenticated admin user
- WHEN the client sends `GET /api/v1/vehicles`
- THEN the response MUST include vehicles with raw device IDs

#### Scenario: Regular user accesses vehicle list

- GIVEN an authenticated regular user
- WHEN the client sends `GET /api/v1/vehicles`
- THEN the response MUST include vehicles with masked device IDs in the format `DEV-****-{last4}`
- AND the last 4 characters MUST be derived deterministically from the raw ID

### Requirement: Sensor Data Ingestion

The system MUST expose a `POST /api/v1/sensors/data` endpoint to receive sensor readings. The endpoint MUST accept a JSON array of sensor data points. Each data point MUST contain: `device_id`, `timestamp` (ISO 8601), `type` (`gps`, `fuel`, `temperature`), and `value` (type-specific object). The system MUST persist all valid data points and MUST reject the entire batch if any data point fails schema validation.

#### Scenario: Valid sensor data batch ingestion

- GIVEN a device with ID `DEV-12345678-ABCD` exists in the system
- WHEN the client sends `POST /api/v1/sensors/data` with a JSON array containing valid GPS and fuel readings
- THEN the system responds with `201 Created` and `{ "inserted": 2 }`
- AND the data MUST be persisted in the `sensor_data` table

#### Scenario: Invalid sensor data batch ingestion

- GIVEN a request with a JSON array where one data point has an invalid `type` value
- WHEN the client sends `POST /api/v1/sensors/data`
- THEN the system responds with `400 Bad Request` and `{ "error": "validation_failed", "details": [...] }`
- AND no data from the batch MUST be persisted

#### Scenario: Sensor data with future timestamp

- GIVEN a request with a timestamp 1 hour in the future
- WHEN the client sends `POST /api/v1/sensors/data`
- THEN the system responds with `400 Bad Request` and `{ "error": "invalid_timestamp" }`

### Requirement: Predictive Fuel Autonomy Calculation

The system MUST calculate fuel autonomy based on the last N fuel readings (N >= 3) to estimate consumption rate. The system MUST generate an alert when the projected time-to-empty is less than 1 hour. The calculation MUST be triggered after each fuel sensor data ingestion.

#### Scenario: Fuel drops below 1-hour autonomy threshold

- GIVEN a vehicle with fuel readings showing a consumption rate of 5 liters/hour and current fuel level of 4 liters
- WHEN a new fuel reading is ingested
- THEN the system MUST calculate time-to-empty as 0.8 hours
- AND the system MUST generate a `low_fuel` alert with severity `critical`

#### Scenario: Fuel remains above threshold

- GIVEN a vehicle with fuel readings showing a consumption rate of 2 liters/hour and current fuel level of 10 liters
- WHEN a new fuel reading is ingested
- THEN the system MUST calculate time-to-empty as 5 hours
- AND the system MUST NOT generate a `low_fuel` alert

#### Scenario: Insufficient data for calculation

- GIVEN a vehicle with only 1 fuel reading
- WHEN a new fuel reading is ingested (total 2)
- THEN the system MUST NOT attempt to project autonomy until at least 3 readings exist

### Requirement: Device ID Masking

The system MUST mask device IDs for all non-admin users. The masked format MUST be `DEV-****-{last4}`, where `{last4}` consists of the last 4 characters of the raw device ID. Masking MUST be applied at the HTTP handler/response serialization layer, not at the database layer.

#### Scenario: Non-admin receives masked ID in API response

- GIVEN a regular user requests vehicle details
- WHEN the system serializes the response
- THEN any `device_id` field MUST be transformed to the masked format

#### Scenario: Admin receives raw ID in API response

- GIVEN an admin user requests vehicle details
- WHEN the system serializes the response
- THEN any `device_id` field MUST remain in its raw format

### Requirement: WebSocket Real-Time Updates

The system MUST maintain a WebSocket hub that broadcasts sensor data updates to all connected clients. Clients MUST connect to `ws://host/api/v1/ws`. The connection MUST require a valid JWT access token passed as a query parameter `?token=<jwt>`. Upon successful connection, the server MUST push JSON messages for new sensor data. Clients MUST filter by `device_id` on the client side.

#### Scenario: Client connects with valid token

- GIVEN a client with a valid access token
- WHEN the client opens a WebSocket connection to `/api/v1/ws?token=<valid_jwt>`
- THEN the connection MUST be accepted with `101 Switching Protocols`

#### Scenario: Client connects with invalid token

- GIVEN a client with an invalid access token
- WHEN the client opens a WebSocket connection to `/api/v1/ws?token=<invalid_jwt>`
- THEN the connection MUST be rejected with `401 Unauthorized`

#### Scenario: Sensor data broadcast

- GIVEN two clients are connected to the WebSocket hub
- WHEN a new sensor data point is ingested for device `DEV-123`
- THEN both clients MUST receive a JSON message containing the sensor data
- AND the message MUST include `device_id`, `type`, `value`, and `timestamp`

### Requirement: Admin-Only Alert Visibility

Predictive fuel alerts (`low_fuel`) MUST only be visible to admin users. The system MUST filter alerts in the `GET /api/v1/alerts` endpoint based on the authenticated user's role. Regular users requesting alerts MUST receive an empty list.

#### Scenario: Admin views alerts

- GIVEN an admin user and an active `low_fuel` alert exists
- WHEN the admin sends `GET /api/v1/alerts`
- THEN the response MUST include the `low_fuel` alert

#### Scenario: Regular user views alerts

- GIVEN a regular user and an active `low_fuel` alert exists
- WHEN the user sends `GET /api/v1/alerts`
- THEN the response MUST be `200 OK` with an empty array `[]`

## Non-Functional Requirements

### Requirement: Performance

The REST API MUST respond to authenticated requests within 200ms at the 95th percentile under normal load (up to 100 concurrent connections). Sensor data ingestion endpoint MUST handle batches of up to 100 data points within 500ms. WebSocket broadcast latency MUST be under 50ms for up to 500 concurrent connections.

### Requirement: Security

All endpoints except `/api/v1/auth/login` and `/api/v1/auth/refresh` MUST require authentication. Passwords MUST be hashed using bcrypt with a cost factor of at least 10. The JWT signing secret MUST be at least 256 bits and loaded from environment variables. SQLite database file MUST have filesystem permissions restricted to the application user.

### Requirement: Offline Support

The backend MUST expose a `GET /api/v1/vehicles/{id}/history` endpoint that returns historical sensor data for a time range. This endpoint MUST support query parameters `from` and `to` (ISO 8601). This enables the frontend to cache historical data for offline use.

## Data Models

### User

| Field      | Type     | Constraints                        |
|------------|----------|------------------------------------|
| id         | UUID     | Primary key                        |
| email      | string   | Unique, not null                   |
| password   | string   | Hashed (bcrypt), not null          |
| role       | string   | `admin` or `user`, default `user`  |
| created_at | datetime | Auto-generated                     |

### Vehicle

| Field      | Type   | Constraints                        |
|------------|--------|------------------------------------|
| id         | UUID   | Primary key                        |
| device_id  | string | Unique, not null                   |
| name       | string | Not null                           |
| created_at | datetime | Auto-generated                   |

### SensorData

| Field      | Type     | Constraints                        |
|------------|----------|------------------------------------|
| id         | UUID     | Primary key                        |
| vehicle_id | UUID     | Foreign key -> Vehicle(id)         |
| type       | string   | `gps`, `fuel`, `temperature`       |
| value      | JSON     | Type-specific payload              |
| timestamp  | datetime | Not null, indexed                  |
| created_at | datetime | Auto-generated                     |

#### GPS Value Schema

```json
{ "lat": float, "lng": float }
```

#### Fuel Value Schema

```json
{ "level": float, "unit": "liters" }
```

#### Temperature Value Schema

```json
{ "celsius": float }
```

## Error Handling Requirements

### Requirement: Standard Error Responses

All API error responses MUST use a consistent JSON structure: `{ "error": "<code>", "message": "<human_readable>" }`. The `error` code MUST be machine-readable and stable. The `message` field MAY be localized in the future.

### Requirement: HTTP Status Codes

| Scenario                  | Status Code                  |
|---------------------------|------------------------------|
| Validation failure        | `400 Bad Request`            |
| Authentication failure    | `401 Unauthorized`           |
| Authorization failure     | `403 Forbidden`              |
| Resource not found        | `404 Not Found`              |
| Method not allowed        | `405 Method Not Allowed`     |
| Rate limit exceeded       | `429 Too Many Requests`      |
| Server error              | `500 Internal Server Error`  |

### Requirement: WebSocket Error Handling

If the WebSocket hub encounters an error while broadcasting to a client, it MUST log the error and MUST close that specific client's connection without affecting other connections.

## Testing Requirements

### Requirement: Mandatory Unit Tests

The following logic MUST have unit tests with table-driven test patterns:

1. **JWT manual implementation**: Token generation, signature validation, expiration checking, malformed token rejection, tampered signature rejection.
2. **Fuel autonomy calculation**: Consumption rate estimation, time-to-empty projection, threshold crossing detection, insufficient data handling.
3. **Device ID masking**: Deterministic masking output, format correctness, preservation of last 4 characters.
4. **Role-based access control**: Middleware correctly allows/denies access based on role claims.

### Requirement: Integration Test Coverage

The following flows MUST have integration tests:

1. **Full authentication flow**: Login -> Access protected endpoint -> Refresh token -> Access with new token.
2. **Sensor data ingestion -> Alert generation**: Ingest fuel data -> Verify alert is created -> Verify admin sees it -> Verify regular user does not.
3. **WebSocket connection**: Connect with valid token -> Receive broadcast after data ingestion.

### Requirement: Deterministic Test Data

Tests MUST use deterministic test data. Time-dependent tests (e.g., JWT expiration) MUST use injected clocks or fixed timestamps rather than `time.Now()`.
