# Guion para Video — Backend IoT Fleet Monitor

> Duración objetivo: 5 minutos exactos
> Audiencia: Evaluadores técnicos (Simon Movilidad)
> Objetivo: Demostrar comprensión profunda de arquitectura, calidad de código, y cumplimiento de requisitos

---

## Segmento 1: Introducción y Contexto (0:00 - 0:30)

**Qué mostrar:**
- Tu cara (o pantalla con tu nombre/título)
- Slide/título: "Backend IoT Fleet Monitor — Prueba Técnica"

**Qué decir:**
> "Hola, soy [tu nombre]. Desarrollé el backend de un sistema de monitoreo IoT para flotas vehiculares. El stack es Go, SQLite, y WebSockets. Voy a mostrarles la arquitectura, los componentes críticos, y una demo en vivo."

**Tips:**
- Sonreí, hablá claro
- No leas — hablá como si explicaras a un colega

---

## Segmento 2: Arquitectura Clean — La Estructura (0:30 - 1:30)

**Qué mostrar:**
- Terminal con `tree backend/internal` o VS Code con el árbol de carpetas
- Resaltar: `domain/`, `application/`, `infrastructure/`

**Qué decir:**
> "Usé Clean Architecture con tres capas. Domain define los modelos de negocio — User, Vehicle, SensorData, Alert. No importa NADA de HTTP ni de base de datos. Application tiene la lógica de negocio: autenticación, cálculo de combustible, validación de sensores. Infrastructure tiene todo lo técnico: SQLite, HTTP handlers, WebSockets, JWT."

**Punto clave a enfatizar:**
> "La regla de oro es: las dependencias apuntan SIEMPRE hacia adentro. Infrastructure depende de Application, que depende de Domain. Nunca al revés. Esto significa que puedo cambiar SQLite por PostgreSQL mañana sin tocar la lógica de negocio."

**Transición:**
> "El requisito más interesante del enunciado es el JWT manual. Veamos eso."

---

## Segmento 3: JWT Manual — Seguridad desde Cero (1:30 - 2:30)

**Qué mostrar:**
- Archivo `internal/infrastructure/jwt/token.go` (scrolleá lentamente)
- Resaltar: `Generate`, `Validate`, `base64urlEncode`, `computeHMAC`

**Qué decir:**
> "El enunciado pedía explícitamente JWT sin librerías externas. Implementé HMAC-SHA256 a mano. Un JWT tiene tres partes: header, payload, y signature. El header dice 'uso HS256'. El payload tiene los claims: quién es el usuario, su rol, cuándo expira. Y la signature es un HMAC-SHA256 del header más el payload, firmado con un secreto de 32 bytes mínimo."

**Demostración en vivo:**
1. Abrí una terminal
2. Corré: `curl -X POST http://localhost:8080/api/v1/auth/login -d '{"email":"admin@example.com","password":"admin123"}'`
3. Copiá el token que devuelve
4. Mostrá: `echo "<token>" | tr '.' '\n' | head -2 | base64 -d` (o explicá que son base64url)

**Punto clave:**
> "Usé `hmac.Equal` para comparar signatures, no `bytes.Equal`. ¿Por qué? Porque `bytes.Equal` tarda más cuando los primeros bytes coinciden. Un atacante podría medir el tiempo y adivinar la signature byte por byte. `hmac.Equal` siempre tarda lo mismo, sin importar cuántos bytes matcheen. Esto se llama 'constant-time comparison' y es fundamental en criptografía."

**Transición:**
> "El otro componente crítico es el cálculo predictivo de combustible."

---

## Segmento 4: Cálculo Predictivo de Combustible (2:30 - 3:30)

**Qué mostrar:**
- `internal/application/fuel_service.go`
- `internal/application/fuel_service_test.go` (scrolleá los test cases)

**Qué decir:**
> "El sistema recibe lecturas de nivel de combustible cada cierto tiempo. Con las últimas 10 lecturas, calculo la tasa de consumo: litros consumidos dividido horas transcurridas. Después, con el nivel actual, proyecto la autonomía: nivel actual dividido tasa. Si da menos de una hora, genero una alerta crítica."

**Mostrá el test:**
> "Acá tengo un test que demuestra esto: 5 lecturas que van de 24 litros a 4 litros en 4 horas. Eso es 5 litros por hora de consumo. Con 4 litros restantes, la autonomía es 0.8 horas. Como es menor a 1, se genera una alerta `low_fuel` de severidad `critical`."

**Mostrá los guards:**
> "Tengo protecciones: si hay menos de 3 lecturas, no calculo — no hay suficiente datos. Si la tasa es cero o negativa — por ejemplo, están cargando nafta — no alerto. Y si hay NaN o Infinito, también salto. Esto evita falsos positivos."

**Transición:**
> "Los datos llegan en tiempo real por WebSocket. Veamos eso."

---

## Segmento 5: WebSockets y Tiempo Real (3:30 - 4:15)

**Qué mostrar:**
- `internal/infrastructure/websocket/hub.go`
- `internal/infrastructure/websocket/client.go`

**Qué decir:**
> "Cuando un sensor envía datos, no solo se guardan en la base de datos — también se difunden en tiempo real a todos los clientes conectados por WebSocket. El Hub central maneja tres canales: registrar cliente, desregistrar cliente, y broadcast. Cada cliente tiene dos goroutines: ReadPump lee del navegador, WritePump escribe hacia el navegador."

**Demostración en vivo:**
1. Abrí el navegador (o una extensión de WebSocket cliente)
2. Conectate a `ws://localhost:8080/api/v1/ws?token=<token>`
3. En otra terminal, enviá datos de sensor con curl
4. Mostrá que llegan en la conexión WebSocket

**Punto clave:**
> "El broadcast es non-blocking. Si un cliente es lento y su buffer está lleno, se dropea el mensaje. No podemos dejar que un cliente lento frene a todo el sistema."

**Transición:**
> "Antes de cerrar, quiero mostrar que todo esto funciona y está testeado."

---

## Segmento 6: Tests y Cierre (4:15 - 5:00)

**Qué mostrar:**
- Terminal corriendo `go test ./... -v` (o `go test ./...` si es muy largo)
- Mostrá el count de tests que pasan
- Opcional: `go test ./... -cover` para mostrar coverage

**Qué decir:**
> "Tengo 108 tests automatizados cubriendo JWT, cálculo de combustible, autenticación, repositorios SQLite, middleware HTTP, y WebSockets. Usé table-driven tests para probar múltiples escenarios de una sola pasada. Los mocks permiten testear services sin base de datos, y los tests de integración usan SQLite en memoria para verificar el stack completo."

**Mostrá el checklist del enunciado:**
> "Revisando los requisitos del enunciado: JWT manual sin librerías externas — check. Cálculo predictivo de combustible con tests — check. Privacidad: IDs enmascarados para usuarios no-admin — check. WebSockets para tiempo real — check. Offline-first en frontend — eso es parte del frontend web. Documentación: tenemos DESIGN.md con decisiones arquitectónicas y SETUP.md con instrucciones paso a paso."

**Cierre:**
> "Este sistema no solo muestra dónde están los camiones — predice cuándo se van a quedar sin combustible antes de que pase. Eso es IoT con inteligencia de negocio. Gracias por su tiempo."

---

## Checklist Técnico para Grabar

### Antes de empezar:
- [ ] Terminal lista con el server corriendo (`go run ./cmd/api`)
- [ ] Base de datos seedeada (`go run ./cmd/api -seed` ya corrido)
- [ ] Postman/curl listo con requests guardados
- [ ] Cliente WebSocket abierto (extensión del navegador o `wscat`)
- [ ] VS Code con los archivos clave abiertos en tabs
- [ ] Cronómetro visible (para controlar los 5 minutos)

### Durante la grabación:
- [ ] Grabá en 1080p mínimo
- [ ] Aumentá el tamaño de fuente de la terminal (al menos 16px)
- [ ] Cerrá notificaciones y aplicaciones innecesarias
- [ ] Si te equivocás, no pares — seguí. Se edita después.
- [ ] Mirá a cámara cuando hagas introducción y cierre

### Después de grabar:
- [ ] Revisá que el audio se escuche claro
- [ ] Verificá que el código se lea bien en pantalla
- [ ] Si te pasaste de 5 minutos, cortá secciones menos críticas (ej. el detalle de los tests)

---

## Variaciones según tu estilo

**Si sos más técnico/detalista:**
- Agregá 30 segundos mostrando el `middleware.go` y cómo funciona RBAC
- Mostrá la estructura de la base de datos (`migrations/001_init.sql`)

**Si sos más práctico/demo-oriented:**
- Acortá la parte de JWT a 45 segundos
- Extendé la demo en vivo: mostrá el mapa con vehículos, el gráfico de combustible, la alerta apareciendo

**Si te queda tiempo al final:**
- Mostrá `scripts/simulate.go` corriendo y enviando datos automáticamente
- Mostrá cómo un usuario normal vs. admin ven IDs diferentes

---

## Frases que Suman Puntos

- "Clean Architecture nos permite testear la lógica de negocio sin base de datos."
- "El JWT manual demuestra que entendemos la criptografía, no solo cómo usar una librería."
- "Constant-time comparison previene timing attacks — es un detalle de seguridad que muchos overlook."
- "El cálculo de autonomía tiene guards matemáticos: NaN, Infinito, rate negativo."
- "Non-blocking broadcast en WebSockets: un cliente lento no frena al resto."
- "108 tests automatizados, desde unitarios hasta integración con SQLite en memoria."
