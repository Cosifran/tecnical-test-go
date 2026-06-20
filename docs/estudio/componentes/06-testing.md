# Ficha: Testing en el Proyecto

## Filosofía de Testing

> "Si no está testeado, no funciona."

En esta prueba técnica, los tests pesan **10% de la evaluación** explícitamente, pero impactan mucho más:
- Demuestran que el código es confiable
- Permiten refactorizar sin miedo
- Documentan el comportamiento esperado
- Aceleran el debugging

---

## Tipos de Tests

### 1. Unit Tests (Services con Mocks)
**Ubicación:** `internal/application/*_test.go`
**Qué testean:** Lógica de negocio sin base de datos
**Cómo:** Mocks que satisfacen las interfaces del dominio

```go
type mockSensorRepo struct {
    data []domain.SensorData
    err  error
}

func (m *mockSensorRepo) FindByVehicleID(ctx context.Context, ...) ([]domain.SensorData, error) {
    return m.data, m.err  // Retorna lo que configuramos en el test
}
```

**Ventaja:** Corren en milisegundos. No necesitamos SQLite levantado.

---

### 2. Integration Tests (Con SQLite en Memoria)
**Ubicación:** `internal/infrastructure/persistence/sqlite/*_test.go`
**Qué testean:** Repositorios reales con base de datos real (pero `:memory:`)
**Cómo:** Cada test abre una BD nueva, corre migraciones, ejecuta queries, verifica resultados.

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, _ := sql.Open("sqlite3", ":memory:")
    persistence.RunMigrations(db, "../../../../../migrations")
    return db
}
```

**Ventaja:** Verifican que las queries SQL son correctas. Detectan errores de schema.

---

### 3. HTTP Handler Tests (Con Servidor Real)
**Ubicación:** `internal/infrastructure/http/handler/*_test.go`
**Qué testean:** Endpoints completos: request → middleware → handler → response
**Cómo:** `httptest.NewServer` crea un servidor HTTP real en un puerto aleatorio.

```go
server := httptest.NewServer(router)
defer server.Close()

resp, _ := http.Post(server.URL+"/api/v1/auth/login", ...)
// Verificar status code, body, headers
```

**Ventaja:** Testean el stack completo: JSON encoding, routing, middleware, error handling.

---

## Patrón: Table-Driven Tests

Es la forma idiomática de testear múltiples casos en Go:

```go
func TestTokenValidation(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        secret  string
        wantErr bool
        errType error
    }{
        {
            name:    "valid_access_token",
            token:   generateValidToken(),
            secret:  "correct-secret-minimum-32-characters",
            wantErr: false,
        },
        {
            name:    "tampered_payload",
            token:   tamperPayload(generateValidToken()),
            secret:  "correct-secret-minimum-32-characters",
            wantErr: true,
            errType: domain.ErrTokenInvalid,
        },
        {
            name:    "wrong_secret",
            token:   generateValidToken(),
            secret:  "wrong-secret-minimum-32-characters!!",
            wantErr: true,
            errType: domain.ErrTokenInvalid,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ts, _ := jwt.NewTokenService(tt.secret, ...)
            _, err := ts.Validate(tt.token)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.True(t, errors.Is(err, tt.errType))
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**Por qué es poderoso:** Agregar un nuevo caso es agregar una línea al slice. No copiar-pegar código.

---

## Inyección de Dependencias para Testear

### Clock Injectable (Tiempo)
```go
type TokenService struct {
    now func() time.Time  // Producción: time.Now, Test: fixedTime
}

// Test:
fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
ts, _ := jwt.NewTokenService(secret, 15*time.Minute, 7*24*time.Hour, func() time.Time {
    return fixedTime
})
```

### BCrypt Injectable
```go
type AuthService struct {
    bcryptCompare func(hashedPassword, password []byte) error
}

// Test (sin calcular bcrypt real):
mockBcrypt := func(hashed, plain []byte) error {
    if string(plain) == "correct" { return nil }
    return errors.New("wrong password")
}
```

**Por qué:** Los tests deben ser rápidos. BCrypt con cost 12 tarda ~200ms por hash. En 50 tests, eso son 10 segundos. Con mock, es instantáneo.

---

## Métricas de Tests

```bash
# Contar tests
go test ./... 2>&1 | grep -E "^(ok|FAIL)"

# Ver cobertura
go test ./... -cover

# Ver cobertura detallada por paquete
go test ./internal/application/ -cover

# Generar reporte HTML
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Resultados actuales:**
- `internal/application`: 37 tests
- `internal/infrastructure/jwt`: 10 tests
- `internal/infrastructure/http`: 30 tests
- `internal/infrastructure/http/handler`: 7 tests
- `internal/infrastructure/persistence/sqlite`: 21 tests
- `internal/infrastructure/websocket`: 3 tests
- **Total: ~108 tests**

---

## Tests Críticos que Demuestran Calidad

### JWT
- ✅ Tampered payload → signature mismatch
- ✅ Expired token → token_expired error
- ✅ Wrong secret → invalid
- ✅ Malformed token (2 segments) → error

### Fuel Calculation
- ✅ < 3 readings → no calculation
- ✅ 5L/h + 4L → alert (0.8h)
- ✅ 2L/h + 10L → no alert (5h)
- ✅ Rate 0 → no alert
- ✅ Rate negative (cargando) → no alert

### Sensor Ingestion
- ✅ Valid batch → persisted
- ✅ Empty batch → validation error
- ✅ >100 items → validation error
- ✅ Unknown device_id → batch rejected
- ✅ Invalid GPS schema → batch rejected
- ✅ Future timestamp → batch rejected

### Auth & RBAC
- ✅ Valid token → access granted
- ✅ Missing token → 401
- ✅ Expired token → 401
- ✅ Admin accessing admin endpoint → 200
- ✅ User accessing admin endpoint → 403

---

## Analogía para el Video

> "Los tests son como los cinturones de seguridad de un auto. No los usás porque esperes chocar, los usás porque SI chocás, querés salir ileso. En código, los tests te permiten cambiar cosas — refactorizar, agregar features, optimizar — sin miedo a romper lo que ya funciona."

---

## Preguntas que te pueden hacer

**Q: "¿Cuál es el coverage de tu proyecto?"**
A: "Corré `go test ./... -cover` y mostrá el número. Si es >70%, está bien. Si es >80%, excelente."

**Q: "¿Por qué no usas testify/assert?"**
A: "Usamos testing estándar de Go. Es más verboso pero cero dependencias. Para una prueba técnica, demostrar que sabés usar el testing nativo es un plus."

**Q: "¿Cómo testeás el cálculo de combustible sin datos reales de sensores?"**
A: "Con mocks. Creo un mockSensorRepo que retorna lecturas predefinidas: [24L, 19L, 14L, 9L, 4L]. El FuelService calcula con esos datos. No necesito una base de datos, un sensor real, ni siquiera el servidor levantado."

---

## Archivos para Revisar

- `internal/application/*_test.go` — Tests de services
- `internal/infrastructure/jwt/token_test.go` — Tests de JWT
- `internal/infrastructure/persistence/sqlite/*_test.go` — Tests de repos
- `internal/infrastructure/http/middleware_test.go` — Tests de middleware
- `internal/infrastructure/http/handler/auth_handler_test.go` — Tests de integración
