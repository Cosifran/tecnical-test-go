## Prueba Técnica Full Stack - Simon Movilidad 

Candidato: Desarrollador II Duración: 3 días 

Entrega: Enviar repositorio y video dentro de las 72 horas siguientes al envío de esta prueba. 

## Objetivo 

Desarrollar un sistema de monitoreo IoT para flotas vehiculares con: 

1. Backend: API para ingesta/procesamiento de datos. 

2. Frontend Web: Dashboard interactivo. 

3. Mobile: App para monitoreo en movimiento (opcional pero valorado). 

## Requisitos Técnicos 

A. Backend (Lenguaje sugiero: Golang o C# aunque puede usar otro donde sea experto) 

   - Implementar una API REST con: 

      - Endpoint de autenticación JWT manual (sin librerías externas para validación). 

      - Ingesta de datos de sensores (ubicación GPS, combustible, temperatura). 

      - Cálculo predictivo de combustible: Alerta si el nivel baja a <1 hora de autonomía. 

   - Persistencia en base de datos (PostgreSQL/SQLite). 

   - WebSockets para actualizaciones en tiempo real. 

- B. Frontend Web (sugerido: React/NextJS aunque puede usar otro) 

   - Dashboard con: 

      - Mapa interactivo (Google Maps/MapLibre) mostrando ubicaciones en vivo. 

      - Gráficos históricos (velocidad/combustible). 

   - Sistema de alertas predictivas visible solo para admin. 

   - Funcionalidad offline usando caché (localStorage/IndexedDB). 

- C. Mobile (React Native - Opcional) - altamente valorado 

   - Réplica del dashboard web con: 

      - Soporte para notificaciones push de alertas. 

      - Sincronización offline. 

- D. Requisitos Generales 

   - Privacidad: Enmascarar IDs de dispositivos para usuarios no administradores (ej: DEV-****-XC54). 

   - Documentación: 

      - DESIGN.md explicando elección de stack y trade-offs técnicos. 

      - SETUP.md con guía de despliegue local. 

   - Testing: Unit tests para lógica crítica (ej: cálculo de combustible, autenticación). 

## Entrega 

1. Repositorio Público con: 

   - Código fuente (backend + frontend + mobile si aplica). 

   - Documentación (DESIGN.md, SETUP.md). 

   - Tests automatizados. 

2. Video Explicativo (≤ 5 min): 

   - Muestra el sistema funcionando (pantalla + voz). 

   - Explica: 

      - Flujo de autenticación. 

      - Mecanismo de alertas predictivas. 

      - Estrategia offline. 

## Criterios de Evaluación 

|Área|Peso|
|---|---|
|Calidad del código|30%|
|Funcionalidad|25%|
|Resiliencia/Offine|20%|
|Documentación|15%|
|Testing|10%|



