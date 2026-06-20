# Roadmap de Estudio — Backend IoT Fleet Monitor

> Objetivo: Dominar cada componente del backend para explicarlo con soltura en el video de 5 minutos.
> Tiempo estimado de estudio: 2-3 horas (lectura + comprensión + ensayo)

---

## Arquitectura Mental del Sistema

Antes de mirar código, entendé ESTO:

```
HTTP Request → Router → Middleware (Auth/RBAC) → Handler → Service → Repository → SQLite
                                    ↓
                              WebSocket Hub ←→ Clients (tiempo real)
```

**Flujo de datos:**
1. **Ingesta**: Sensor envía datos → API POST → SensorService valida → SQLite → WebSocket broadcast
2. **Consulta**: Frontend pide vehículos → API GET → VehicleService → SQLite → Masking según rol → JSON
3. **Alerta**: FuelService lee últimos datos → calcula autonomía → si < 1h → crea alerta en SQLite
4. **Auth**: Login → bcrypt verify → JWT manual (HMAC-SHA256) → access_token (15min) + refresh_token (7d)

---

## Roadmap por Bloques (orden recomendado)

### Bloque 1: Fundamentos — 30 minutos
**Archivos a leer:**
- `internal/domain/entity.go` — Los objetos del negocio
- `internal/domain/repository.go` — Los contratos (interfaces)
- `internal/domain/errors.go` — Errores predefinidos

**Qué tenés que saber explicar:**
- [ ] Por qué `json.RawMessage` en `SensorData.Value`
- [ ] Qué es un "sentinel error" y por qué usamos `errors.Is()`
- [ ] Diferencia entre `Vehicle.ID` (UUID interno) y `Vehicle.DeviceID` (hardware)
- [ ] Qué información guarda un JWT claim (`sub`, `email`, `role`, `iat`, `exp`, `type`)

**Pregunta clave para el video:** *"El dominio no sabe nada de HTTP ni de base de datos — es puro negocio. ¿Por qué? Porque si mañana cambiamos SQLite por PostgreSQL, el dominio no se entera."*

---

### Bloque 2: JWT Manual — 45 minutos
**Archivos a leer:**
- `internal/infrastructure/jwt/token.go` (257 líneas, bien comentado)
- `internal/infrastructure/jwt/token_test.go`

**Qué tenés que saber explicar:**
- [ ] Estructura de un JWT: `header.payload.signature`
- [ ] Qué es HMAC-SHA256 y por qué usamos `hmac.Equal` (timing attacks)
- [ ] Base64url encoding SIN padding (`RawURLEncoding`)
- [ ] Por qué inyectamos `now func() time.Time` (testabilidad)
- [ ] Diferencia entre access token (15 min) y refresh token (7 días)
- [ ] Qué pasa si alguien modifica el payload (tampering)

**Demostración sugerida:**
1. Generar un token con `curl`
2. Copiar el token, cambiar un carácter del payload, intentar validar → falla
3. Dejar pasar 15 minutos (o cambiar el clock inyectado en test) → expira

**Pregunta clave:** *"¿Por qué no usamos una librería de JWT? Porque el enunciado lo pide explícito. Y además demuestra que entendemos CÓMO funciona JWT, no solo CÓMO usarlo."*

---

### Bloque 3: Cálculo Predictivo de Combustible — 30 minutos
**Archivos a leer:**
- `internal/application/fuel_service.go`
- `internal/application/fuel_service_test.go`

**Qué tenés que saber explicar:**
- [ ] La fórmula: `rate = (oldest - newest) / delta_time_hours`
- [ ] La fórmula: `autonomy = newest_level / rate`
- [ ] Guards: < 3 lecturas → skip; rate <= 0 → no alerta; NaN/Inf → skip
- [ ] Por qué usamos `readingsCount` como `limit` en el repo (eficiencia)
- [ ] Qué guarda el campo `details` de la alerta (autonomy, rate, current_level)

**Demostración sugerida:**
- Mostrar el test `TestCheckAutonomy_HighConsumption_TriggersAlert`
- Explicar: "24L → 19L → 14L → 9L → 4L en 4 horas = 5L/h. Con 4L restantes: 4/5 = 0.8h de autonomía. Como es < 1h, alerta crítica."

**Pregunta clave:** *"Esto es lo que le da valor al sistema: no solo muestra datos, sino que predice cuándo un camión se va a quedar sin nafta."*

---

### Bloque 4: Masking y Privacidad — 15 minutos
**Archivos a leer:**
- `internal/application/masking.go`
- `internal/application/masking_test.go`

**Qué tenés que saber explicar:**
- [ ] Formato: `DEV-****-{last4}`
- [ ] Por qué está en application layer y no en domain (es presentación, no negocio)
- [ ] Edge case: ID < 4 caracteres → `DEV-****-????`
- [ ] Por qué es determinístico (mismo input → mismo output)

**Pregunta clave:** *"Un usuario normal nunca ve el ID real del dispositivo. Solo los administradores. Esto es privacidad por diseño."*

---

### Bloque 5: SensorService y Validación — 30 minutos
**Archivos a leer:**
- `internal/application/sensor_service.go`
- `internal/application/sensor_service_test.go`

**Qué tenés que saber explicar:**
- [ ] Flujo de `IngestBatch`: validar → resolver device_id → persistir → trigger fuel check
- [ ] Cache de `vehicleCache` (evita N+1 queries)
- [ ] Atomicidad: si UN punto falla, se rechaza TODO el batch
- [ ] Validación de schema según tipo (GPS → lat/lng, fuel → level, temp → celsius)
- [ ] Por qué el fuel check corre DESPUÉS de persistir
- [ ] Qué pasa si `CheckAutonomy` falla (se ignora, no rompe el batch)

---

### Bloque 6: Repositorios SQLite — 20 minutos
**Archivos a leer:**
- `internal/infrastructure/persistence/sqlite/helpers.go`
- `internal/infrastructure/persistence/sqlite/user_repo.go`
- `internal/infrastructure/persistence/sqlite/sensor_repo.go`

**Qué tenés que saber explicar:**
- [ ] Interfaces en domain, implementaciones en sqlite (Dependency Inversion)
- [ ] `BulkInsert` usa transacciones (`BEGIN` → inserts → `COMMIT`/`ROLLBACK`)
- [ ] `FindByVehicleID` usa SQL dinámico según filtros
- [ ] UUID generado con `crypto/rand` (16 bytes → hex = 32 chars)
- [ ] `isUniqueConstraintError` detecta duplicados por mensaje de error

---

### Bloque 7: HTTP Layer — 25 minutos
**Archivos a leer:**
- `internal/infrastructure/http/middleware.go`
- `internal/infrastructure/http/router.go`
- `internal/infrastructure/http/handler/auth_handler.go`
- `internal/infrastructure/http/handler/vehicle_handler.go`

**Qué tenés que saber explicar:**
- [ ] Cómo AuthMiddleware extrae el Bearer token y lo valida
- [ ] Cómo RBACMiddleware permite/deniega según rol
- [ ] Qué pasa si un token expira vs. si es inválido (ambos 401, pero códigos distintos)
- [ ] Cómo el vehicle handler aplica masking antes de responder
- [ ] Por qué sensor ingestion es solo para admin (protección de datos)

---

### Bloque 8: WebSockets — 20 minutos
**Archivos a leer:**
- `internal/infrastructure/websocket/hub.go`
- `internal/infrastructure/websocket/client.go`

**Qué tenés que saber explicar:**
- [ ] Pattern: Hub central con channels (register/unregister/broadcast)
- [ ] ReadPump y WritePump como goroutines separadas
- [ ] Autenticación vía `?token=` query param en el upgrade
- [ ] Non-blocking broadcast (drop si el cliente es lento)
- [ ] Read limit: 4096 bytes (protección contra payloads enormes)

---

### Bloque 9: main.go y Wiring — 15 minutos
**Archivos a leer:**
- `cmd/api/main.go`

**Qué tenés que saber explicar:**
- [ ] Orden de inicialización: config → DB → migrations → repos → services → router
- [ ] Graceful shutdown (`signal.NotifyContext` + `server.Shutdown`)
- [ ] Qué hace `-seed` (crea admin/user + 3 vehículos con bcrypt)
- [ ] Por qué `hub.Run()` va en una goroutine

---

### Bloque 10: Tests — 20 minutos
**Archivos a leer:**
- `internal/application/*_test.go` (todos)
- `internal/infrastructure/jwt/token_test.go`
- `internal/infrastructure/persistence/sqlite/*_test.go`

**Qué tenés que saber explicar:**
- [ ] Table-driven tests (un struct con casos, un loop con `t.Run`)
- [ ] Mocks en memoria (sin base de datos real)
- [ ] Tests de integración con `:memory:` SQLite
- [ ] Inyección de clock para testear expiración sin esperar
- [ ] Coverage: ¿qué porcentaje tenemos? (`go test -cover ./...`)

---

## Checklist Final (antes de grabar)

- [ ] Corré `go test ./...` y mostrá que pasa
- [ ] Corré `go run ./cmd/api -seed` y mostrá los logs
- [ ] Hacé login con curl/Postman y mostrá los tokens
- [ ] Enviá un batch de sensores y mostrá la respuesta 201
- [ ] Conectá un cliente WebSocket y mostrá que recibe datos
- [ ] Mostrá un vehículo con admin (ID completo) vs. con user (mascarado)
- [ ] Abrí el dashboard y mostrá una alerta de combustible

---

## Comandos Útiles para la Demo

```bash
# Seed
cd backend && go run ./cmd/api -seed

# Start server
JWT_SECRET=change-this-to-a-secure-random-secret-at-least-32-characters-long go run ./cmd/api

# Login como admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'

# Listar vehículos (usar token del paso anterior)
curl http://localhost:8080/api/v1/vehicles \
  -H "Authorization: Bearer <token>"

# Enviar datos de sensor
curl -X POST http://localhost:8080/api/v1/sensors/data \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "DEV-11111111-AAAA",
    "type": "fuel",
    "timestamp": "2024-01-15T10:00:00Z",
    "value": {"level": 4.0, "unit": "liters"}
  }'

# Ver alertas
curl http://localhost:8080/api/v1/alerts \
  -H "Authorization: Bearer <token>"

# Tests
go test ./... -v
go test ./... -cover
```

---

## Tips para el Video

1. **Empezá con el problema**: "Tenemos una flota de camiones y necesitamos saber en tiempo real dónde están, cuánta nafta tienen, y si se van a quedar sin combustible."

2. **Mostrá arquitectura primero**: Abrí el árbol de carpetas y explicá por qué están separadas.

3. **JWT manual es tu AS**: Es el diferenciador más fuerte. Mostrá el código, explicá HMAC, mostrá que funciona.

4. **Los tests son prueba de calidad**: Decí "108 tests pasando" y mostrá el comando corriendo.

5. **Demo en vivo**: Siempre más impactante que slides. Mostrá curl/Postman interactuando con la API.

6. **No te quedes en lo obvio**: En 5 minutos no podés mostrar TODO. Priorizá: arquitectura → JWT → combustible → WebSocket → demo.

7. **Cerrá con valor de negocio**: "El sistema no solo muestra datos, sino que predice problemas antes de que pasen. Eso es IoT con inteligencia."
