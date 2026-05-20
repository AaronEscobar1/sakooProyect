# Sakoo Backend - Clean Architecture Go Server

Este repositorio contiene la API REST y el servidor backend para el **Proyecto Sakoo**, desarrollado en **Go** siguiendo los principios de **Arquitectura Limpia (Clean Architecture)**.

---

## 🛠️ Tecnologías y Herramientas

- **Lenguaje**: Go (v1.22+)
- **Base de Datos**: PostgreSQL (con el driver `pgx/v5` y pool de conexiones `pgxpool`)
- **Documentación API**: Swagger/OpenAPI (v3) mediante `swag` y `http-swagger`
- **Gestión de Entornos y Tareas**:
  - Live Reloading: `air`
  - Variables de entorno: `godotenv`
  - Automatización de scraping e intervalos: Administrador de cron interno

---

## 📁 Estructura del Proyecto

El backend está diseñado siguiendo una separación estricta de capas para garantizar robustez, mantenibilidad y facilidad de prueba:

```mermaid
graph TD
    cmd[cmd/api/main.go] --> API(Capa de Presentación / API - internal/api)
    API --> Domain(Capa de Dominio / Casos de Uso - internal/usecase)
    Domain --> Entities(Entidades Core - internal/domain)
    Infrastructure(Infraestructura / Persistencia - internal/infrastructure) --> Domain
    Infrastructure --> pool[(Base de Datos PostgreSQL)]
```

* **`cmd/api/`**: Punto de entrada de la aplicación (`main.go`). Registra las rutas de la API, carga la configuración y conecta la infraestructura.
* **`internal/domain/`**: Entidades del dominio de negocio (p.ej., `BankAccount`, `Message`, `PaymentCommitment`, `Comment`) e interfaces de contratos.
* **`internal/usecase/`**: Lógica de aplicación que coordina e implementa las operaciones principales del sistema.
* **`internal/api/`**: Controladores HTTP, decodificadores y middleware de autenticación (JWT) y de logs asíncronos.
* **`internal/infrastructure/`**: Implementaciones técnicas (Repositorios SQL en `repository/`, Web Scraping en `scraper/`, notificaciones de correo en `email/`, y cron en `cron/`).
* **`docs/`**: Especificación de Swagger/OpenAPI autogenerada dinámicamente.
* **`bruno/`**: Colección de peticiones HTTP en formato Bruno para pruebas en caliente.

---

## 🚀 Instalación y Uso Local

### Requisitos Previos
1. Tener instalado [Go (1.22 o superior)](https://go.dev/).
2. Una base de datos PostgreSQL activa.
3. El archivo `.env` configurado en el directorio raíz.

### Configuración del Entorno `.env`
Crea un archivo `.env` en la raíz del proyecto con la siguiente estructura:
```env
PORT=8080
DATABASE_URL=postgres://usuario:contraseña@localhost:5432/sakoo?sslmode=disable
JWT_SECRET=tu-clave-secreta-para-firmar-tokens-jwt
```

### Ejecutar con Recarga en Caliente (Air)
Si tienes `air` instalado, ejecuta:
```bash
air
```

Si no está en tu PATH global pero está en tu `go/bin`, ejecuta:
```powershell
& "$env:USERPROFILE\go\bin\air.exe"
```

El servidor compilará automáticamente ante cualquier cambio de código y estará escuchando peticiones en:
👉 **[http://localhost:8080](http://localhost:8080)**

---

## 📚 Documentación Interactiva de la API (Swagger)

Toda la API está documentada usando **Swaggo**. Para visualizar la documentación interactiva, probar en vivo cada uno de los 34 endpoints y autenticarte con tu token JWT:

1. Levanta el servidor.
2. Abre en tu navegador de preferencia:
   👉 **[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

### Actualización de la especificación
Si realizas cambios en los controladores de la API y deseas regenerar la documentación de Swagger:
```powershell
& "$env:USERPROFILE\go\bin\swag.exe" init -g cmd/api/main.go --parseDependency --parseInternal
```

---

## 🧪 Pruebas con Bruno
En la carpeta `/bruno` encontrarás la colección con 30 peticiones HTTP organizadas por módulos (Autenticación, Mensajes, Tasas, Scraping, etc.). Importa la carpeta directamente en el cliente de API **Bruno** para interactuar de forma inmediata con los endpoints de desarrollo.
