# AGENTS.md — Simon Movilidad Prueba Técnica

> Contexto: este repo es una prueba técnica de 3 días para Desarrollador II. El sistema es un monitoreo IoT para flotas vehiculares. Cada decisión debe priorizar los criterios de evaluación: calidad de código (30%), funcionalidad (25%), resiliencia/offline (20%), documentación (15%), testing (10%).

## Stack Sugerido (definir en DESIGN.md)

- **Backend**: Golang o C# (o tu lenguaje de expertise). API REST + WebSockets.
- **Frontend Web**: React o NextJS.
- **Mobile (opcional, valorado)**: React Native.
- **Base de datos**: PostgreSQL o SQLite.

## Restricciones Críticas del Enunciado

- **JWT manual**: implementar autenticación JWT SIN librerías externas de validación. Usar crypto estándar de tu lenguaje.
- **Cálculo predictivo de combustible**: alertar si el nivel baja a < 1 hora de autonomía. Esto es lógica crítica que requiere tests unitarios.
- **Privacidad**: enmascarar IDs de dispositivos para usuarios no administradores. Formato ejemplo: `DEV-****-XC54`.
- **Alertas predictivas**: visibles SOLO para administradores.
- **Offline**: frontend debe funcionar con caché (localStorage/IndexedDB). Mobile también requiere sincronización offline.
- **WebSockets**: actualizaciones en tiempo real de ubicación/sensores.

## Entregables Obligatorios

1. `DESIGN.md`: elección de stack y trade-offs técnicos.
2. `SETUP.md`: guía de despliegue local paso a paso.
3. Tests automatizados, especialmente para:
   - Lógica de cálculo de combustible/autonomía.
   - Autenticación JWT.
4. Video explicativo (≤ 5 min) — no generable por agente, pero el código debe ser fácil de demo.

## Estructura Esperada del Repo

```
/
├── backend/          # API REST + WebSockets
├── frontend-web/     # Dashboard React/NextJS
├── mobile/           # React Native (opcional)
├── docs/
│   ├── DESIGN.md
│   └── SETUP.md
└── AGENTS.md
```

## Convenciones y Quirks

- **No usar librerías JWT externas**: validar tokens a mano con HMAC + base64. Es un requisito explícito del enunciado.
- **Tests unitarios obligatorios** para lógica crítica. No omitir.
- **Mascarar IDs**: siempre verificar rol antes de exponer IDs crudos. El formato de máscara debe ser consistente.
- **Offline-first**: el frontend debe degradar gracefulmente sin conexión. Cachear datos históricos y estado del mapa.
- **WebSockets**: usar para push de datos de sensores, no solo polling.

## Checklist antes de entregar

- [ ] `DESIGN.md` escrito y actualizado.
- [ ] `SETUP.md` con instrucciones claras de levantamiento local.
- [ ] Backend corre y expone API REST + WebSockets.
- [ ] Frontend Web corre, muestra mapa y gráficos.
- [ ] Tests unitarios pasan (especialmente auth y combustible).
- [ ] Mobile funciona (si se implementó).
- [ ] No hay IDs de dispositivos expuestos a usuarios no-admin.

## Notas para Agentes Futuros

- Este es un repo de prueba técnica: la calidad del código y la documentación pesan más que features extra.
- No agregar features fuera de scope hasta que los requisitos base estén 100% implementados y testeados.
- Si hay conflicto entre una librería cómoda y el requisito de "manual JWT", gana el requisito.
- Priorizar la resiliencia/offline: es 20% de la nota.
