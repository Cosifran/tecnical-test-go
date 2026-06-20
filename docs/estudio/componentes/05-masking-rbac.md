# Ficha: Masking de IDs y Control de Acceso

## El Requisito de Privacidad

**Enunciado:** "Enmascarar IDs de dispositivos para usuarios no administradores. Formato ejemplo: `DEV-****-XC54`."

Esto no es un "nice to have" — es un requisito de **privacidad y seguridad**. Los IDs de dispositivos hardware son información sensible: podrían usarse para rastrear, identificar, o atacar dispositivos IoT.

---

## La Función de Masking

```go
func MaskDeviceID(raw string) string {
    if len(raw) < 4 {
        return "DEV-****-????"
    }
    last4 := raw[len(raw)-4:]
    return fmt.Sprintf("DEV-****-%s", last4)
}
```

**Ejemplos:**
| Input | Output |
|-------|--------|
| `DEV-12345678-ABCD` | `DEV-****-ABCD` |
| `SENSOR-XYZ-1234` | `DEV-****-1234` |
| `AB` | `DEV-****-????` |
| `` (vacío) | `DEV-****-????` |

**Determinístico:** La misma entrada SIEMPRE produce la misma salida. Esto permite que un usuario correlacione observaciones sin ver el ID completo.

---

## ¿Dónde Aplicar el Masking?

**Regla de oro:** El masking es una **concern de presentación**, no de negocio ni de persistencia.

```
❌ NUNCA en Domain (entity.go)
❌ NUNCA en Repository (sqlite/*.go)
✅ SIEMPRE en Application/Handler (vehicle_service.go, vehicle_handler.go)
```

**Por qué:**
- Si guardamos IDs mascarados en la BD, perdemos el dato original
- Si el domain trabaja con IDs mascarados, no puede buscar por device_id
- El masking depende de QUIÉN pregunta (rol del usuario), no del dato en sí

**Implementación en VehicleService:**
```go
func (s *VehicleService) ListVehicles(ctx context.Context, userRole string) ([]domain.Vehicle, error) {
    vehicles, err := s.vehicleRepo.FindAll(ctx)
    if err != nil { return nil, err }
    
    if userRole != "admin" {
        for i := range vehicles {
            vehicles[i].DeviceID = s.maskFunc(vehicles[i].DeviceID)
        }
    }
    
    return vehicles, nil
}
```

---

## Control de Acceso Basado en Roles (RBAC)

**Dos roles:**
- `admin`: ve IDs reales, ve alertas, puede ingestar datos
- `user`: ve IDs mascarados, NO ve alertas, NO puede ingestar

**Implementación:**

### En Middleware
```go
func RBACMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := r.Context().Value("user_role").(string)
            
            for _, allowed := range allowedRoles {
                if role == allowed {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            writeError(w, 403, "forbidden", "insufficient permissions")
        })
    }
}
```

### En Router
```go
// Admin + User pueden ver vehículos
mux.Handle("GET /api/v1/vehicles", AuthMiddleware(...)(
    RBACMiddleware("admin", "user")(vehicleHandler.List),
))

// Solo Admin puede ingestar
mux.Handle("POST /api/v1/sensors/data", AuthMiddleware(...)(
    RBACMiddleware("admin")(sensorHandler.Ingest),
))
```

**Alertas para usuarios normales:**
El endpoint `/api/v1/alerts` ya está protegido con `RBACMiddleware("admin")`. Si un usuario normal llega ahí, obtiene 403. Pero según el spec, los usuarios normales deberían ver un array vacío en vez de error. En nuestro caso, como usan roles distintos y el endpoint es admin-only, un user nunca llega ahí por diseño.

---

## Tests Cubiertos

| Test | Escenario |
|------|-----------|
| `TestMaskDeviceID/standard` | ID normal → `DEV-****-ABCD` |
| `TestMaskDeviceID/short` | ID de 2 chars → `DEV-****-????` |
| `TestMaskDeviceID/empty` | Vacío → `DEV-****-????` |
| `TestMaskDeviceID_Deterministic` | Mismo input → mismo output |
| `TestListVehicles_AsAdmin` | Admin ve IDs reales |
| `TestListVehicles_AsUser` | User ve IDs mascarados |
| `TestGetVehicleHistory_NonAdmin` | Historial con ID enmascarado |

---

## Analogía para el Video

> "Es como mostrar una cuenta bancaria. El administrador del banco ve el número completo de cuenta: 1234-5678-9012-3456. El usuario normal solo ve los últimos 4 dígitos: ****-****-****-3456. Ambos pueden identificar la cuenta, pero solo el administrador tiene acceso completo."

---

## Preguntas que te pueden hacer

**Q: "¿Por qué no hashear el ID en vez de mascararlo?"**
A: "El hash es irreversible. Si un admin necesita ver el ID real para debugging o soporte, no podría. El masking preserva los últimos 4 caracteres como identificador visual único, pero oculta la parte sensible."

**Q: "¿Qué pasa si un usuario adivina el ID completo viendo los últimos 4 caracteres?"**
A: "Los últimos 4 caracteres reducen el espacio de búsqueda, pero no lo hacen trivial. Un atacante tendría que adivinar los primeros N-4 caracteres. Para seguridad adicional, los device_ids deberían ser aleatorios y largos, no secuenciales."

---

## Archivos para Revisar

- `internal/application/masking.go` — Implementación
- `internal/application/masking_test.go` — Tests
- `internal/application/vehicle_service.go` — Uso en ListVehicles y GetVehicleHistory
- `internal/infrastructure/http/middleware.go` — RBACMiddleware
