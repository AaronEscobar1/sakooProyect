# Skill de Desarrollo en Go (Golang) — Sakoo Backend

Esta guía especifica las mejores prácticas, modismos del compilador y pautas del lenguaje Go aplicadas directamente al backend de Sakoo para garantizar un código eficiente, seguro y libre de fallos en producción.

---

## 1. Cumplimiento del Enrutamiento en Go 1.22+

El backend de Sakoo aprovecha las optimizaciones nativas de Go 1.22 para el enrutamiento HTTP mediante `net/http.ServeMux`.

* **Método y Ruta Explícitos**: Define el método al inicio del patrón separado por un espacio.
  ```go
  mux.HandleFunc("POST /api/auth/login", authHandler.HandleLogin)
  ```
* **Variables de Ruta Dinámicas**: Define variables de segmento usando llaves `{}`.
  ```go
  mux.Handle("PUT /api/v1/accounts/own/{id}", ...)
  ```
* **Extracción de Variables**: Recupera las variables dinámicas de forma limpia dentro del handler usando `r.PathValue`:
  ```go
  idStr := r.PathValue("id")
  ```

---

## 2. Gestión de Conexiones PostgreSQL con `pgx/v5`

El backend utiliza `pgx/v5` como driver y gestor de conexiones con PostgreSQL por su rendimiento nativo.

* **Concurrencia Segura**: Pasa siempre la referencia a `*pgxpool.Pool` para inyectar la base de datos en los repositorios de la capa de infraestructura.
* **Control Estricto de Contextos**: Nunca ejecutes queries sin pasar el contexto HTTP. Esto asegura que si el cliente cancela la petición o el timeout se dispara, la base de datos libere los recursos inmediatamente.
  ```go
  ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
  defer cancel()
  
  err := db.QueryRow(ctx, query, args...).Scan(&dest)
  ```
* **Limpieza de Cursores**: Si ejecutas consultas con múltiples filas (`Query`), cierra siempre el cursor con un bloque deferido y verifica el error residual del cursor:
  ```go
  rows, err := db.Query(ctx, query, args...)
  if err != nil {
      return nil, err
  }
  defer rows.Close()
  
  for rows.Next() {
      // scan...
  }
  if err = rows.Err(); err != nil {
      return nil, err
  }
  ```

---

## 3. Robustez en Concurrencia y Goroutines

El backend realiza registro de logs y otras tareas en segundo plano de manera asíncrona usando goroutines dedicadas.

* **Prevención de Pánicos en Cascadas**: Cualquier goroutine asíncrona iniciada con la instrucción `go` debe contar obligatoriamente con un mecanismo de recuperación ante pánicos (`recover`) para evitar que un fallo inesperado tire el servidor completo:
  ```go
  go func() {
      defer func() {
          if r := recover(); r != nil {
              slog.Error("Pánico recuperado en tarea asíncrona", 
                  "panic", r,
                  "stack", string(debug.Stack()),
              )
          }
      }()
      // Código de la goroutine aquí...
  }()
  ```

---

## 4. Estricta Higiene del Compilador de Go

* **Cero Imports no Usados**: El compilador de Go fallará y detendrá cualquier build de desarrollo o CI/CD si existen paquetes importados en la cabecera del archivo que no se utilicen en el código.
* **Limpieza proactiva**: Al refactorizar lógica o remover llamadas a funciones (como `os.Getenv` o `fmt.Sprintf`), elimina inmediatamente el paquete correspondiente del bloque `import (...)`.
* **Verificación de Compilación**: Antes de dar por finalizada cualquier edición, valida localmente:
  ```powershell
  go build ./cmd/api
  ```
