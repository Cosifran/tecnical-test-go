# Ficha: Cálculo Predictivo de Combustible

## El Problema de Negocio

Tenemos camiones con sensores de combustible. El sistema recibe lecturas periódicas. Necesitamos **predecir** si un camión se va a quedar sin nafta en menos de 1 hora.

**No es suficiente mostrar el nivel actual.** Un camión con 10 litros yendo a 2L/h tiene 5 horas de autonomía. Otro con 10 litros yendo a 15L/h tiene 40 minutos. El segundo necesita alerta crítica.

---

## Las Fórmulas

### 1. Tasa de Consumo
```
rate = (nivel_antes - nivel_ahora) / horas_transcurridas
```

**Ejemplo:**
- Hace 4 horas: 24 litros
- Ahora: 4 litros
- Consumidos: 24 - 4 = 20 litros
- Tiempo: 4 horas
- Rate: 20 / 4 = **5 litros/hora**

### 2. Autonomía Proyectada
```
autonomy = nivel_actual / rate
```

**Ejemplo:**
- Nivel actual: 4 litros
- Rate: 5 L/h
- Autonomía: 4 / 5 = **0.8 horas = 48 minutos**

### 3. Decisión
```
if autonomy < 1.0 hora:
    generar alerta "low_fuel" severity "critical"
```

---

## Guards Matemáticos

El cálculo puede fallar de varias maneras. Tenemos protecciones:

### Guard 1: Pocos Datos
```go
if len(readings) < 3 {
    return nil // No hay suficiente historia para una tendencia confiable
}
```
**Por qué:** Con 2 puntos, la "tendencia" podría ser ruido. Con 3+ puntos, tenemos al menos 2 intervalos para validar.

### Guard 2: Rate Cero
```go
if rate <= 0 {
    return nil // No está consumiendo (apagado) o está cargando nafta
}
```
**Por qué:** Si rate = 0, la autonomía sería infinita (4/0 = Inf). Si rate < 0, está cargando combustible — ¡no necesita alerta!

### Guard 3: Valores Inválidos
```go
if math.IsNaN(rate) || math.IsInf(rate, 0) {
    return nil // Datos corruptos o división por cero
}
```
**Por qué:** NaN puede aparecer si los timestamps son idénticos (Δt = 0 → división por cero). Inf puede aparecer si el rate es cero.

### Guard 4: Autonomía Inválida
```go
if math.IsNaN(autonomy) || math.IsInf(autonomy, 0) {
    return nil
}
```
**Por qué:** Doble chequeo. Si por alguna razón el cálculo de autonomía da NaN/Inf, no generamos alerta.

---

## Implementación en Código

```go
func (fs *FuelService) CheckAutonomy(ctx context.Context, vehicleID string, readingsCount int) error {
    // 1. Obtener últimas N lecturas de combustible
    data, err := fs.sensorRepo.FindByVehicleID(ctx, vehicleID, zeroTime, zeroTime, "fuel", readingsCount)
    
    // 2. Ordenar por timestamp (más vieja primero)
    sort.Slice(data, func(i, j int) bool {
        return data[i].Timestamp.Before(data[j].Timestamp)
    })
    
    // 3. Extraer niveles de combustible
    readings := make([]domain.FuelReadingWithTime, 0, len(data))
    for _, point := range data {
        var fuel domain.FuelReading
        json.Unmarshal(point.Value, &fuel)
        readings = append(readings, domain.FuelReadingWithTime{
            Level:     fuel.Level,
            Timestamp: point.Timestamp,
        })
    }
    
    // 4. Guards
    if len(readings) < 3 { return nil }
    
    // 5. Calcular rate y autonomía
    oldest := readings[0]
    newest := readings[len(readings)-1]
    
    deltaLiters := oldest.Level - newest.Level
    deltaHours := newest.Timestamp.Sub(oldest.Timestamp).Hours()
    rate := deltaLiters / deltaHours
    
    if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
        return nil
    }
    
    autonomy := newest.Level / rate
    
    if autonomy >= 1.0 {
        return nil // Tiene más de 1 hora — todo bien
    }
    
    // 6. Generar alerta
    alert := &domain.Alert{
        VehicleID: vehicleID,
        Type:      "low_fuel",
        Severity:  "critical",
        Details:   json.Marshal(map[string]interface{}{
            "autonomy":      autonomy,
            "rate":          rate,
            "current_level": newest.Level,
        }),
    }
    
    return fs.alertRepo.Create(ctx, alert)
}
```

---

## Tests Cubiertos

| Test | Escenario | Esperado |
|------|-----------|----------|
| `< 3 readings` | Solo 2 lecturas | No alerta (datos insuficientes) |
| `High consumption` | 5L/h, 4L actual | Alerta (0.8h < 1h) |
| `Low consumption` | 2L/h, 10L actual | No alerta (5h >= 1h) |
| `Zero rate` | Nivel no cambia | No alerta (rate = 0) |
| `Negative rate` | Está cargando nafta | No alerta (rate < 0) |
| `Repo error` | Base de datos falla | Error devuelto |

---

## Por Qué `readingsCount` como Parámetro

Inicialmente `readingsCount` era un parámetro que no se usaba. Lo corregimos agregándolo al repositorio:

```go
// Antes: traía TODOS los datos históricos (ineficiente)
data, err := fs.sensorRepo.FindByVehicleID(ctx, vehicleID, from, to, "fuel")

// Después: trae solo las últimas N (eficiente)
data, err := fs.sensorRepo.FindByVehicleID(ctx, vehicleID, from, to, "fuel", readingsCount)
```

**Impacto:** Si un camión tiene 2 años de datos, antes traíamos TODO a memoria. Ahora la base de datos limita a 10 registros. Escala mejor.

---

## Analogía para el Video

> "Es como calcular el rendimiento de tu auto. Si en la última semana gastaste 35 litros en 500 kilómetros, tu rendimiento es 14 km/l. Si te quedan 20 litros en el tanque, podés recorrer 280 km más. Si la próxima estación de servicio está a 300 km, el sistema te alerta ANTES de que te quedes sin nafta en medio de la ruta."

---

## Preguntas que te pueden hacer

**Q: "¿Por qué 1 hora y no 30 minutos o 2 horas?"**
A: "El enunciado especificó < 1 hora de autonomía. Es un umbral de negocio que define 'crítico'. Podría ser configurable en el futuro."

**Q: "¿Qué pasa si el sensor manda datos erráticos?"**
A: "Los guards matemáticos protegen contra eso. NaN, Infinito, rate negativo — todos se descartan. Además, el service ordena por timestamp y usa las lecturas más recientes, así que un dato espurado viejo no afecta el cálculo actual."

**Q: "¿Por qué usan linear regression en vez de promedio simple?"**
A: "En este caso usamos rate promedio simple porque es suficiente para el requisito. Con más datos y variabilidad, un modelo más sofisticado (regresión lineal o incluso ML) podría dar mejores predicciones. Pero para una prueba de 3 días, simple y testeado > complejo y no testeado."

---

## Archivos para Revisar

- `internal/application/fuel_service.go` — Implementación
- `internal/application/fuel_service_test.go` — Tests
- `internal/domain/entity.go` (líneas 101-119) — FuelReading, FuelReadingWithTime
