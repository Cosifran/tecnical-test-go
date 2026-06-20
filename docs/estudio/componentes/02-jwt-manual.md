# Ficha: JWT Manual con HMAC-SHA256

## ¿Qué es un JWT?

JSON Web Token es un estándar (RFC 7519) para transmitir claims de forma compacta y segura.

**Estructura:**
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
   ↑ header                        ↑ payload                         ↑ signature
```

Cada parte está en **base64url** sin padding, separada por puntos.

---

## Proceso de Generación (paso a paso)

### Paso 1: Header
```json
{"alg": "HS256", "typ": "JWT"}
```
Siempre es el mismo. Dice qué algoritmo usamos.

### Paso 2: Payload (Claims)
```json
{
  "sub": "user-uuid",
  "email": "admin@example.com",
  "role": "admin",
  "iat": 1700000000,
  "exp": 1700000900,
  "type": "access"
}
```
- `sub`: subject — el ID del usuario
- `email`: para mostrar en frontend sin query extra
- `role`: para RBAC sin query extra
- `iat`: issued at — cuándo se creó
- `exp`: expires at — cuándo vence
- `type`: "access" o "refresh"

### Paso 3: Signature
```
HMAC-SHA256(base64url(header) + "." + base64url(payload), secret)
```

**El secreto debe tener al menos 32 bytes** (256 bits) para que HMAC-SHA256 sea seguro.

### Paso 4: Unir
```
base64url(header) + "." + base64url(payload) + "." + base64url(signature)
```

---

## Proceso de Validación

1. **Split**: dividir el token por los puntos → deben ser exactamente 3 partes
2. **Recompute**: calcular HMAC-SHA256 del header.payload con el secreto
3. **Compare**: comparar la signature recibida con la recomputada usando `hmac.Equal`
4. **Decode**: decodificar el payload a Claims
5. **Expire**: verificar que `exp > now`

**Si FALLA cualquier paso → token inválido.**

---

## Por Qué `hmac.Equal` y No `bytes.Equal`

```go
// ❌ INSEGURO: timing attack posible
if bytes.Equal(expected, actual) { ... }

// ✅ SEGURO: tiempo constante
if hmac.Equal(expected, actual) { ... }
```

**El problema:** `bytes.Equal` retorna `false` TAN PRONTO como encuentra un byte diferente. Si los primeros 10 bytes coinciden pero el 11º no, tarda más que si el 1º ya es diferente.

**El ataque:** Un atacante envía miles de tokens con signatures variadas y mide el tiempo de respuesta. Si un token con "A..." tarda 1ms y uno con "B..." tarda 2ms, sabe que "A" es el primer byte correcto. Repite byte por byte hasta reconstruir la signature.

**La solución:** `hmac.Equal` compara TODOS los bytes SIEMPRE, sin early return. El tiempo es constante independientemente de cuántos bytes coincidan.

---

## Access Token vs Refresh Token

| Campo | Access Token | Refresh Token |
|-------|-------------|---------------|
| `type` | `"access"` | `"refresh"` |
| Duración | 15 minutos | 7 días |
| Uso | Acceder a APIs protegidas | Obtener nuevos access tokens |
| Guardado en | Memory/short-term storage | Secure HTTP-only cookie |

**Refresh Token Rotation:**
Cuando usás un refresh token para obtener uno nuevo, el sistema genera un access token NUEVO y un refresh token NUEVO. El viejo sigue válido hasta su expiración natural, pero el cliente debería descartarlo y usar el nuevo.

**Por qué:** Si alguien roba el refresh token, tiene 7 días máximo para usarlo. Y cada vez que el usuario legítimo hace refresh, el atacante pierde el token viejo (si implementamos invalidación, que en este caso no hicimos por simplicidad).

---

## Inyección de Clock para Tests

```go
type TokenService struct {
    secret []byte
    now func() time.Time  // ← injectable!
}
```

En producción: `now = time.Now`
En tests: `now = func() time.Time { return fixedTime }`

**Por qué:** Para testear expiración sin dormir 15 minutos. Podemos "congelar" el tiempo, generar un token, avanzar el reloj 16 minutos, y verificar que falla.

---

## Tests Cubiertos

| Test | Qué verifica |
|------|-------------|
| `TestGenerateAndValidate` | Round-trip básico |
| `TestExpiredToken` | Token pasado exp → error |
| `TestTokenValidation/tampered_payload` | Payload modificado → error |
| `TestTokenValidation/wrong_secret` | Secreto incorrecto → error |
| `TestTokenValidation/malformed_token` | No tiene 3 partes → error |
| `TestRefreshTokenTTL` | Refresh token dura 7 días, no 15 min |

---

## Pregunta Clave para el Video

> "¿Por qué no usamos github.com/golang-jwt/jwt? Porque el enunciado lo pide manual. Y además, implementarlo a mano demuestra que entendemos el protocolo: sabemos que es base64url, sabemos qué es HMAC-SHA256, sabemos por qué el timing attack importa. No estamos usando una caja negra — estamos construyendo la caja."

---

## Archivos para Revisar

- `internal/infrastructure/jwt/token.go` — Implementación
- `internal/infrastructure/jwt/token_test.go` — Tests
- `internal/domain/entity.go` (líneas 121-143) — Claims struct
