# Ficha: WebSockets en Tiempo Real

## ¿Por Qué WebSockets?

HTTP es **request-response**: el cliente pregunta, el servidor responde. Para actualizaciones en tiempo real, las alternativas son:

1. **Polling**: el cliente pregunta cada X segundos. Problema: consume batería, datos, y es lento.
2. **Long-polling**: el servidor espera hasta tener datos. Mejor, pero complejo y no verdadero tiempo real.
3. **WebSockets**: conexión bidireccional persistente. El servidor PUEDE enviar datos al cliente sin que este pregunte.

**En nuestro caso:** Cuando un sensor envía datos de GPS, queremos que TODOS los dashboards conectados vean el movimiento del camión en el mapa INMEDIATAMENTE, sin recargar la página.

---

## Arquitectura: Hub + Clientes

```
┌─────────────────────────────────────────┐
│              Hub Central                │
│  ┌─────────┐  ┌─────────┐  ┌────────┐ │
│  │ Client 1│  │ Client 2│  │Client 3│ │
│  │ (Web)   │  │ (Web)   │  │(Mobile)│ │
│  └────┬────┘  └────┬────┘  └───┬────┘ │
│       └─────────────┴───────────┘       │
│              broadcast channel          │
└─────────────────────────────────────────┘
         ↑
    Sensor envía datos
    → Handler llama hub.Broadcast()
    → Hub envía a todos los clients
```

---

## El Hub

```go
type Hub struct {
    clients    map[*Client]bool  // Set de clientes conectados
    register   chan *Client      // Canal para registrar nuevos
    unregister chan *Client      // Canal para desregistrar
    broadcast  chan []byte       // Canal para mensajes a difundir
}
```

**Por qué channels y no mutex:** En Go, los channels son la forma idiomática de comunicación entre goroutines. El Hub tiene una goroutine `Run()` que escucha los tres canales en un `select`:

```go
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            delete(h.clients, client)
            close(client.send)
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:  // Enviar si hay espacio
                default:
                    // Buffer lleno → cerrar cliente lento
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

**Non-blocking broadcast:** Si `client.send` está lleno (el cliente no lee rápido), el `default` cierra la conexión en lugar de bloquear a todos los demás. Esto evita que un cliente lento frene el sistema.

---

## El Cliente

Cada conexión WebSocket tiene DOS goroutines:

### ReadPump
```go
func (c *Client) ReadPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadLimit(4096)  // Máximo 4KB por mensaje
    
    for {
        _, _, err := c.conn.ReadMessage()
        if err != nil {
            break  // Error de lectura → salir
        }
        // Procesar mensaje del cliente (si aplica)
    }
}
```

**Read limit:** 4096 bytes protege contra payloads enormes que podrían saturar memoria.

### WritePump
```go
func (c *Client) WritePump() {
    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            c.conn.WriteMessage(websocket.TextMessage, message)
        }
    }
}
```

**Separación:** ReadPump lee DEL cliente, WritePump escribe HACIA el cliente. Van en goroutines separadas porque una puede bloquear sin afectar la otra.

---

## Autenticación en WebSocket

WebSocket no soporta headers de HTTP estándar después del handshake. La forma de autenticar es via query parameter:

```
ws://localhost:8080/api/v1/ws?token=<jwt_token>
```

**En el servidor:**
```go
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, tokenService *jwt.TokenService) {
    token := r.URL.Query().Get("token")
    claims, err := tokenService.Validate(token)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Upgrade a WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    
    // Crear cliente y registrar
    client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
    hub.register <- client
    
    // Iniciar goroutines
    go client.WritePump()
    go client.ReadPump()
}
```

**Por qué query param y no header:** El protocolo WebSocket permite headers en el handshake inicial, pero muchos clientes (navegadores, librerías) no exponen forma de setear headers personalizados. El query param funciona universalmente.

---

## Ciclo de Vida de una Conexión

```
1. Cliente abre ws://.../ws?token=xxx
2. Servidor valida token
3. Upgrade HTTP → WebSocket
4. Hub.register ← client
5. ReadPump y WritePump arrancan
6. [En algún momento] Sensor envía datos → Handler → hub.Broadcast(msg)
7. Hub itera clients → envía a cada send channel
8. WritePump recibe → escribe al websocket
9. Cliente recibe mensaje en tiempo real
10. [Al desconectar] ReadPump detecta error → hub.unregister ← client
```

---

## Tests

| Test | Qué verifica |
|------|-------------|
| `TestHubRegisterUnregister` | Cliente se registra y desregistra correctamente |
| `TestHubBroadcast` | Mensaje broadcast llega a todos los clientes |
| `TestHubSlowClientDrop` | Cliente lento se cierra sin bloquear a otros |

---

## Analogía para el Video

> "Imaginen una radio FM. El Hub es la emisora. Cada cliente es un radio en una casa. Cuando el locutor habla (sensor envía datos), la emisora transmite a TODOS los radios al mismo tiempo. Si un radio está roto o muy lejos, la emisora no deja de transmitir — simplemente deja de enviarle señal a ese radio en particular."

---

## Preguntas que te pueden hacer

**Q: "¿Por qué no usan SSE (Server-Sent Events) en vez de WebSockets?"**
A: "SSE es más simple para unidireccional (servidor → cliente), pero WebSockets permiten bidireccional. En el futuro podríamos querer que el dashboard envíe comandos al servidor (ej. 'solicitar ubicación actual'). WebSocket nos da esa flexibilidad."

**Q: "¿Qué pasa si se cae el Hub?"**
A: "El Hub corre como goroutine en el mismo proceso que el servidor HTTP. Si el proceso muere, todo se cae. Para producción a escala, se podría extraer el Hub a un servicio separado con Redis Pub/Sub para distribuir entre múltiples instancias."

**Q: "¿Cómo escala esto a 10.000 clientes?"**
A: "Go maneja goroutines muy eficientemente — 10.000 goroutines consumen pocos MB de memoria. El cuello de botella sería el broadcast: iterar 10.000 clients en un loop. Para escalar más, se podría particionar en múltiples Hubs o usar un message broker como Redis Streams."

---

## Archivos para Revisar

- `internal/infrastructure/websocket/hub.go` — Hub
- `internal/infrastructure/websocket/client.go` — Cliente y ServeWS
- `internal/infrastructure/websocket/websocket_test.go` — Tests
