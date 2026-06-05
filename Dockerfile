# --- Stage 1: Build (Compilación Inmutable) ---
FROM golang:1.26-alpine AS builder

# Instalar dependencias necesarias para la compilación y zona horaria
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copiar la totalidad del código fuente (incluyendo la carpeta vendor/)
COPY . .

# Compilar utilizando la carpeta vendor local
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o main ./cmd/api

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
