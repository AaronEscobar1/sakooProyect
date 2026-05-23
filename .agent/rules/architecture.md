# Reglas de Arquitectura y Negocio — Sakoo Backend

Este documento define las reglas de diseño inamovibles y restricciones de negocio que rigen el desarrollo del backend del Proyecto Sakoo. Cualquier modificación al código debe apegarse estrictamente a estas especificaciones.

---

## 1. Contexto Unificado con Repomix (LECTURA OBLIGATORIA)

> [!IMPORTANT]
> **Repomix como Fuente Primaria de Contexto**: Existe un archivo unificado y estructurado generado por Repomix en la raíz del proyecto llamado `repomix-output.xml` (o en su defecto `repomix-output.txt`).
> 
> * **Acción Obligatoria para Agentes**: Antes de analizar el proyecto, proponer refactorizaciones o realizar cambios, **debes leer y examinar primero este archivo**. Esto garantiza que tengas en memoria el estado más actualizado y completo de todos los archivos fuente de la base de código en un solo paso, eliminando la necesidad de realizar búsquedas fragmentadas e ineficientes.

---

## 2. Identificadores Secuenciales (NO UUIDs)

> [!IMPORTANT]
> **Identificadores Secuenciales obligatorios**: Todos los identificadores (IDs) en el sistema (usuarios, logs, cuentas, banners, etc.) deben ser estrictamente numéricos enteros secuenciales (`BIGSERIAL` en la base de datos PostgreSQL, `int64` en Go). 
> **Bajo ninguna circunstancia se deben usar UUIDs o strings como identificadores primarios.**

---


## 3. Privacidad de Detalles Técnicos en la API

> [!WARNING]
> **Seguridad y Opacidad de Errores**: Ningún error nativo de la base de datos SQL o del pool de conexiones (como restricciones de unicidad de clave duplicada, fallos del driver, consultas sintácticamente erróneas) debe ser expuesto directamente en el JSON de respuesta devuelto al cliente HTTP. 
> 
> * **Acción Requerida**: Los errores deben ser interceptados en la capa API/Handlers, registrados en consola usando `slog.Error` con su respectivo `track_code`, y convertidos a un código y mensaje de error de cara al usuario a través del sistema estandarizado de `response.Error`.

---

## 4. Políticas de CORS y Preflights (OPTIONS)

> [!CAUTION]
> **Manejo Centralizado de Preflights**: No agregues manejadores para el método `OPTIONS` directamente dentro del enrutador principal de Go (`net/http.ServeMux`) para endpoints individuales.
> 
> * **Razón**: Si se registran manejadores `OPTIONS` en el mux, estos interceptarán la petición antes de que pasen por la inyección de encabezados de CORS del middleware principal. Esto provocará que los navegadores (como Chrome) bloqueen el request por falta de headers CORS.
> * **Solución**: Deja que la lógica del middleware global de CORS en `internal/api/middleware/cors.go` maneje e intercepte todos los preflights respondiendo de inmediato con un `204 No Content` con todos los headers inyectados.

---

## 5. Respuestas Estandarizadas Fuertemente Tipadas

Todas las peticiones HTTP de la API Sakoo deben retornar obligatoriamente un formato consistente `APIResponse[T any]` fuertemente tipado mediante genéricos de Go:

```json
{
  "code": 1000,
  "message": "Operación exitosa",
  "data": { ... },
  "track_code": "wKTxFoZqfVLFGGkZ"
}
```

* **Código de Éxito (SUCCESS / CREATED)**: Devuelve exactamente el código numérico **`1000`**.
* **Códigos de Fallos**: Códigos distintos a 1000 semánticos para el cliente (ej: `1001` JSON inválido, `1002` Parámetros inválidos, `1003` No autorizado, `1004` Recurso duplicado).
