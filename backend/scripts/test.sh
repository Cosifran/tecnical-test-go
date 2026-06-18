#!/bin/bash
# test.sh — Script para ejecutar tests del backend
# Uso: ./scripts/test.sh (desde cualquier directorio)

set -e

# Obtener el directorio donde está este script, sin importar desde dónde se ejecute
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Ir al directorio padre (donde está go.mod)
cd "$SCRIPT_DIR/.."

echo "=== Fleet Monitor Backend Tests ==="
echo "Directorio: $(pwd)"
echo ""

# Verificar que Go esté instalado
if ! command -v go &> /dev/null; then
    echo "ERROR: Go no está instalado."
    echo "Instalalo desde: https://go.dev/dl/"
    exit 1
fi

echo "Go version: $(go version)"
echo ""

# Verificar que exista go.mod
if [ ! -f "go.mod" ]; then
    echo "ERROR: No se encontró go.mod en $(pwd)"
    echo "Asegurate de ejecutar este script desde el directorio del proyecto."
    exit 1
fi

# Ejecutar todos los tests con verbose
echo "Ejecutando tests..."
go test ./... -v

echo ""
echo "=== TODOS LOS TESTS PASARON ✅ ==="
