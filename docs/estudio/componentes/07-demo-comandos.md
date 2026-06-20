# Ficha: Comandos y Demo para el Video

## Preparación Antes de Grabar

### 1. Variables de entorno
```bash
cd /home/francisco/Documents/prueba-tecnica/backend
export JWT_SECRET="change-this-to-a-secure-random-secret-at-least-32-characters-long"
```

### 2. Seed de datos (solo una vez)
```bash
go run ./cmd/api -seed
```
**Esperado:**
```
INFO seeded admin user id=... email=admin@example.com
INFO seeded regular user id=... email=user@example.com
INFO seeded vehicle id=... device_id=DEV-11111111-AAAA
INFO seeded vehicle id=... device_id=DEV-22222222-BBBB
INFO seeded vehicle id=... device_id=DEV-33333333-CCCC
INFO database seeded successfully
```

### 3. Levantar el servidor
```bash
go run ./cmd/api
```
**Esperado:**
```
INFO server starting addr=:8080
INFO websocket hub started
```

---

## Demo 1: Login y Token

### Admin login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'
```

**Respuesta:**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Para el video:** Copiá el `access_token` en una variable:
```bash
TOKEN="eyJhbGc..."
```

---

## Demo 2: Listar Vehículos (Admin vs User)

### Como Admin
```bash
curl http://localhost:8080/api/v1/vehicles \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta:**
```json
{
  "vehicles": [
    {"id": "...", "device_id": "DEV-11111111-AAAA", "name": "Truck 01"},
    {"id": "...", "device_id": "DEV-22222222-BBBB", "name": "Truck 02"},
    {"id": "...", "device_id": "DEV-33333333-CCCC", "name": "Truck 03"}
  ]
}
```

**Para el video:** Resaltá que los IDs son completos.

### Como User
```bash
# Login como user primero
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"user123"}'

# Guardar el token
USER_TOKEN="eyJhbGc..."

# Listar vehículos
curl http://localhost:8080/api/v1/vehicles \
  -H "Authorization: Bearer $USER_TOKEN"
```

**Respuesta:**
```json
{
  "vehicles": [
    {"id": "...", "device_id": "DEV-****-AAAA", "name": "Truck 01"},
    {"id": "...", "device_id": "DEV-****-BBBB", "name": "Truck 02"},
    {"id": "...", "device_id": "DEV-****-CCCC", "name": "Truck 03"}
  ]
}
```

**Para el video:** Resaltá la diferencia. Mismo endpoint, mismos datos, presentación distinta según rol.

---

## Demo 3: Enviar Datos de Sensor

```bash
curl -X POST http://localhost:8080/api/v1/sensors/data \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "DEV-11111111-AAAA",
    "type": "fuel",
    "timestamp": "2024-01-15T10:00:00Z",
    "value": {"level": 4.0, "unit": "liters"}
  }'
```

**Respuesta:**
```json
{"inserted": 1}
```

**Para el video:** Explicá que esto simula un sensor IoT enviando una lectura de combustible.

---

## Demo 4: Ver Alertas

### Primero, enviar datos históricos que disparen alerta

Necesitamos enviar MÚLTIPLES lecturas de combustible decrecientes. Podés usar el simulador:

```bash
# En otra terminal
cd /home/francisco/Documents/prueba-tecnica/backend
go run scripts/simulate.go -alert-mode
```

**O manualmente:**
```bash
# Enviar 5 lecturas decrecientes (24→19→14→9→4)
for level in 24 19 14 9 4; do
  curl -X POST http://localhost:8080/api/v1/sensors/data \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"device_id\": \"DEV-11111111-AAAA\",
      \"type\": \"fuel\",
      \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"value\": {\"level\": $level, \"unit\": \"liters\"}
    }"
  sleep 1
done
```

### Ver alertas
```bash
curl http://localhost:8080/api/v1/alerts \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta esperada:**
```json
{
  "alerts": [
    {
      "id": "...",
      "vehicle_id": "...",
      "type": "low_fuel",
      "severity": "critical",
      "details": {
        "autonomy": 0.8,
        "rate": 5.0,
        "current_level": 4.0
      }
    }
  ]
}
```

---

## Demo 5: WebSocket en Tiempo Real

### Conectar con wscat
```bash
# Instalar si no tenés
npm install -g wscat

# Conectar con token
wscat -c "ws://localhost:8080/api/v1/ws?token=$TOKEN"
```

**Esperado:**
```
Connected (press CTRL+C to quit)
>
```

### Enviar datos y ver broadcast
En otra terminal, enviá un sensor:
```bash
curl -X POST http://localhost:8080/api/v1/sensors/data \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "DEV-11111111-AAAA",
    "type": "gps",
    "timestamp": "2024-01-15T10:00:00Z",
    "value": {"lat": -34.6037, "lng": -58.3816}
  }'
```

**En la terminal de wscat deberías ver:**
```json
{"type":"gps","vehicle_id":"...","lat":-34.6037,"lng":-58.3816}
```

**Para el video:** "El dato llegó al servidor, se guardó en SQLite, y se difundió en tiempo real a todos los clientes conectados."

---

## Demo 6: Tests Pasando

```bash
cd /home/francisco/Documents/prueba-tecnica/backend
go test ./... -v
```

**Bonus:** Mostrá coverage:
```bash
go test ./... -cover
```

---

## Demo 7: Simulador Automático

```bash
cd /home/prueba-tecnica/backend
go run scripts/simulate.go -iterations 5
```

**Esperado:**
```
Authenticated as admin@example.com
Iteration 1/5: Sent 3 sensor points → 201 Created
Iteration 2/5: Sent 3 sensor points → 201 Created
...
```

---

## Troubleshooting Rápido

| Problema | Solución |
|----------|----------|
| `JWT_SECRET must be at least 32 characters` | `export JWT_SECRET="change-this-to-a-secure-random-secret-at-least-32-characters-long"` |
| `database is locked` | SQLite con WAL mode debería manejarlo, pero si persiste, cerrá otras conexiones |
| `connection refused` | El servidor no está corriendo. Verificá con `go run ./cmd/api` |
| `401 Unauthorized` | Token expirado (15 min) o inválido. Hacé login de nuevo |
| `403 Forbidden` | El endpoint requiere rol admin y tu token es de user |

