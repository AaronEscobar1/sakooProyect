# --- Stage 1: Build (Compilación Inmutable) ---
FROM golang:1.22-alpine AS builder

# Instalar dependencias necesarias para la compilación y zona horaria
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copiar archivos de dependencias go.mod y go.sum primero para optimizar la caché de Docker
COPY go.mod go.sum ./
RUN go mod download

# Copiar la totalidad del código fuente
COPY . .

# Compilar un binario estático y minimalista
# CGO_ENABLED=0 elimina dependencias de librerías dinámicas de C, garantizando máxima portabilidad.
# -ldflags="-w -s" elimina información de depuración y símbolos, reduciendo el tamaño del binario en ~40%.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api

# --- Stage 2: Runtime (Entorno de Ejecución Seguro y Minimalista) ---
FROM alpine:3.19

# Instalar ca-certificates actualizados para conexiones TLS seguras externas
# e importar tzdata para soportar manejo de zonas horarias en los Cron Jobs del backend
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar el binario altamente optimizado desde la fase builder
COPY --from=builder /app/main .

# Copiar la carpeta de migraciones (Crítico para que golang-migrate aplique los esquemas al arrancar)
COPY --from=builder /app/migrations ./migrations

# Exponer el puerto por defecto de la API
EXPOSE 8080

# Definir variables de entorno de fallback seguras
ENV PORT=8080 \
    GO_ENV=production

# Ejecutar el binario del backend
ENTRYPOINT ["./main"]
