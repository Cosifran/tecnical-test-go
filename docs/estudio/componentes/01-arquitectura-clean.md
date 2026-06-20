# Ficha: Arquitectura Clean en el Proyecto

## Concepto Central

Clean Architecture (o Hexagonal/Onion) organiza el código en capas concéntricas donde las dependencias apuntan SIEMPRE hacia el centro.

```
         ┌─────────────────────────────┐
         │      Infrastructure         │  ← HTTP, DB, JWT, WS
         │   (frameworks, drivers)     │    Depende de Application
         ├─────────────────────────────┤
         │       Application           │  ← Services, use cases
         │      (business rules)       │    Depende de Domain
         ├─────────────────────────────┤
         │         Domain              │  ← Entities, interfaces
         │    (enterprise rules)       │    NO depende de nadie
         └─────────────────────────────┘
```

---

## En Nuestro Proyecto

### `internal/domain/` — El Núcleo
**Archivos:** `entity.go`, `repository.go`, `errors.go`

**Qué hay:**
- `User`, `Vehicle`, `SensorData`, `Alert` — los objetos del negocio
- Interfaces como `UserRepository`, `VehicleRepository` — los CONTRATOS
- Errores como `ErrNotFound`, `ErrUnauthorized` — el lenguaje del negocio

**Qué NO hay:**
- Nada de `net/http`
- Nada de `database/sql`
- Nada de JWT
- Nada de WebSockets

**Por qué:** Si mañana cambiamos SQLite por PostgreSQL, o Go por Python, el dominio sigue igual.

---

### `internal/application/` — La Lógica
**Archivos:** `auth_service.go`, `fuel_service.go`, `sensor_service.go`, `vehicle_service.go`, `masking.go`

**Qué hay:**
- Servicios que orquestan operaciones del negocio
- Validación de reglas (batch size, timestamp futuro, schema de sensores)
- Cálculos (autonomía de combustible)
- Transformaciones (masking de IDs)

**Depende de:** `domain/` (interfaces y entidades)

**NO depende de:** `infrastructure/` (no sabe si usamos SQLite o qué librería HTTP)

---

### `internal/infrastructure/` — La Tecnología
**Subpaquetes:** `http/`, `persistence/sqlite/`, `jwt/`, `websocket/`

**Qué hay:**
- `http/`: handlers, middleware, router — traduce HTTP a llamadas a services
- `persistence/sqlite/`: implementaciones concretas de los repositories
- `jwt/`: generación y validación de tokens usando `crypto/hmac`
- `websocket/`: hub, clientes, upgrade de conexiones

**Depende de:** `application/` y `domain/` (usa los services y satisface las interfaces)

---

## Regla de Oro: La Flecha de Dependencias

```go
// ❌ MAL: Domain importando infraestructura
package domain
import "github.com/francisco/fleet-monitor/internal/infrastructure/jwt" // ¡NUNCA!

// ✅ BIEN: Infraestructura importando domain
package jwt
import "github.com/francisco/fleet-monitor/internal/domain" // Esto sí
```

---

## Ventajas para Esta Prueba Técnica

| Ventaja | Cómo se demuestra |
|---------|-------------------|
| **Testabilidad** | Services testeados con mocks, sin base de datos real |
| **Flexibilidad** | Podríamos cambiar SQLite por PostgreSQL sin tocar services |
| **Claridad** | Cada capa tiene UNA responsabilidad |
| **Mantenibilidad** | Si hay un bug en JWT, sabés exactamente dónde buscar |

---

## Analogía

> "Imaginen un restaurante. El `domain` es el menú: define qué platos existen y qué ingredientes llevan. El `application` es el chef: sabe CÓMO cocinar cada plato siguiendo las recetas. El `infrastructure` es la cocina: los hornos, las ollas, los cuchillos. El chef no le dice al horno cómo calentar — solo le pasa la temperatura. Y el menú no sabe ni que existe un horno."

## Archivos Clave para Revisar

1. `internal/domain/repository.go` — ver las interfaces
2. `internal/application/fuel_service.go` — ver cómo usa interfaces
3. `internal/infrastructure/persistence/sqlite/sensor_repo.go` — ver cómo satisface la interfaz
4. `cmd/api/main.go` — ver cómo se cablea todo
