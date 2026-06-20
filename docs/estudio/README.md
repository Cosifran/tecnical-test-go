# Índice de Estudio — Backend IoT Fleet Monitor

> Material de preparación

---

## Roadmap Principal

1. **[Roadmap de Estudio](01-roadmap-estudio.md)** — Guía completa de qué estudiar y en qué orden. 10 bloques, ~2-3 horas totales.

---

## Fichas por Componente

### Fundamentos
- **[Arquitectura Clean](componentes/01-arquitectura-clean.md)** — Capas, dependencias, analogía del restaurante

### Seguridad
- **[JWT Manual con HMAC-SHA256](componentes/02-jwt-manual.md)** — Generación, validación, timing attacks, access vs refresh

### Lógica de Negocio
- **[Cálculo Predictivo de Combustible](componentes/03-calculo-combustible.md)** — Fórmulas, guards matemáticos, tests
- **[Masking y RBAC](componentes/05-masking-rbac.md)** — Privacidad, control de acceso, determinismo

### Infraestructura
- **[WebSockets](componentes/04-websockets.md)** — Hub, clientes, non-blocking broadcast, auth via query param

### Calidad
- **[Testing](componentes/06-testing.md)** — Unit, integration, table-driven, inyección de dependencias

### Demo
- **[Comandos y Demo](componentes/07-demo-comandos.md)** — Todos los curls, WebSocket, simulador, troubleshooting

## Orden Recomendado de Estudio

```
Día 1 (1 hora):
  ├─ Roadmap de Estudio (lectura rápida, 15 min)
  ├─ Arquitectura Clean (comprensión profunda, 30 min)
  └─ JWT Manual (entender HMAC y estructura, 15 min)

Día 2 (1 hora):
  ├─ Cálculo Predictivo (fórmulas + tests, 30 min)
  ├─ WebSockets (hub + cliente, 20 min)
  └─ Masking y RBAC (rápido, 10 min)

Día 3 (1 hora):
  ├─ Testing (patrones, 20 min)
  ├─ Demo y Comandos (practicar curls, 20 min)
  └─ Guion del Video (ensayar 2-3 veces, 20 min)
```

---

## Frases Clave para Memorizar

1. *"El dominio no sabe nada de HTTP ni de base de datos — es puro negocio."*
2. *"Implementé JWT manual con HMAC-SHA256 y constant-time comparison para prevenir timing attacks."*
3. *"Con 4 litros y 5 litros por hora de consumo, la autonomía es 0.8 horas — alerta crítica."*
4. *"Un usuario normal ve `DEV-****-ABCD`, un admin ve `DEV-12345678-ABCD` — privacidad por diseño."*
5. *"El broadcast es non-blocking: un cliente lento no frena al resto."*
6. *"108 tests automatizados, desde unitarios con mocks hasta integración con SQLite en memoria."*

---
