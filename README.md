# SAI API — Sistema de Ayuda Integral

<div align="center">

**Universidad Autónoma de Sinaloa**  
**Facultad de Ingenieria Mochis" (FIM UAS)**  
**Hackathon 2 FIM UAS**

### Equipo: Git Pushers

Backend del **Sistema de Ayuda Integral (SAI)**, una API para la gestión segura de expedientes, canalizaciones, acompañamiento, bolsa de trabajo, portal ciudadano y analítica institucional.

[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Fiber](https://img.shields.io/badge/Fiber-v2-00ACD7)](https://gofiber.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Swagger](https://img.shields.io/badge/Swagger-OpenAPI-85EA2D?logo=swagger&logoColor=black)](http://localhost:3001/docs/)

</div>

---

## Descripción

**SAI API** es el backend del proyecto desarrollado por el equipo **Git Pushers** para el **Hackathon 2 FIM UAS** de la **Universidad Autónoma de Sinaloa**.

El sistema centraliza procesos de atención y seguimiento mediante una API REST segura, con separación de identidades (personal interno, portal ciudadano y empleadores), control de acceso por permisos, auditoría trazable, cifrado de datos sensibles y observabilidad.

## Funcionalidades principales

- **Autenticación y sesiones**
  - Login de personal con JWT y sesiones revocables.
  - MFA TOTP con enrolamiento y verificación.
  - CSRF en operaciones de escritura.
  - Bloqueo por intentos fallidos y protección anti-enumeración.

- **Gestión de expedientes**
  - Creación, consulta, actualización y línea de tiempo.
  - Verificación de duplicados.
  - Asignaciones, notas por departamento, consentimientos y revelación controlada de datos sensibles.
  - Canalizaciones internas y solicitudes de acceso a expediente.

- **Portal ciudadano**
  - Acceso por OTP.
  - Consulta/actualización de perfil.
  - CV, postulaciones, respuesta a entrevistas y retiro de postulaciones.
  - Alertas de ayuda y notificación a contactos de confianza.

- **Empleadores y bolsa de trabajo**
  - Registro y autenticación OTP para empleadores.
  - Publicación y gestión de vacantes.
  - Revisión administrativa, liberación de contacto y seguimiento de contrataciones.

- **Cursos, encuestas y vinculación**
  - Gestión de cursos, inscripciones y asistencia.
  - Encuestas pendientes, respuesta y resultados.
  - Visitas de vinculación institucional.

- **Auditoría y analítica**
  - Bitácora de auditoría con cadena de integridad.
  - Alertas de auditoría.
  - Resumen analítico y mapa de calor.

- **Observabilidad y seguridad**
  - Métricas Prometheus en `/metrics`.
  - Trazas OpenTelemetry OTLP opcionales.
  - Logs estructurados JSON con `request_id`.
  - Reglas de alerta y dashboard en `deploy/observability/`.

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Lenguaje | Go |
| Framework HTTP | Fiber v2 |
| Base de datos | PostgreSQL 17 |
| ORM | GORM |
| Auth | JWT HS256, Argon2id, TOTP, CSRF |
| Cifrado | AES-256-GCM (envelope encryption), HMAC-SHA256 |
| Documentación API | Swagger / OpenAPI (`swaggo`) |
| Observabilidad | Prometheus, OpenTelemetry, slog JSON |
| Contenedores | Docker, Docker Compose |

## Estructura del proyecto

```text
.
├── cmd/
│   ├── main.go              # Punto de entrada de la API
│   ├── migrate/             # Migraciones/esquema
│   ├── seed/                # Bootstrap inicial
│   └── seed-demo/           # Datos demo
├── deploy/
│   └── observability/       # Alertas Prometheus, dashboard Grafana, README
├── docs/                    # Swagger generado y documentación técnica
├── git/                     # Guías de flujo Git del equipo
├── internal/
│   ├── auth/                # Autenticación, MFA, CSRF, tokens, contraseñas
│   ├── audit/               # Registro de auditoría
│   ├── config/              # Carga de configuración/env
│   ├── handlers/            # Handlers HTTP por dominio
│   ├── httpx/               # Errores, JSON estricto, ETag, utilidades HTTP
│   ├── metrics/             # Contadores in-memory
│   ├── middleware/          # Auth, RBAC, CSRF, rate limit, portal/employer
│   ├── models/              # Modelos y DTOs
│   ├── observability/       # Prometheus y OpenTelemetry
│   ├── repository/          # Interfaces y repositorios GORM
│   ├── security/            # Cifrado, keyring, tokens opacos
│   └── services/            # Lógica de negocio por dominio
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

## Requisitos

- **Go** 1.27+
- **PostgreSQL** 17+ (o Docker)
- **Docker / Docker Compose** (opcional, recomendado)
- **swag** CLI (solo si vas a regenerar Swagger)

## Configuración

1. Copia el archivo de entorno base:

```bash
cp .env.example .env
```

2. Configura las variables necesarias. Como mínimo, revisa:

| Variable | Requerida | Descripción |
|---|---:|---|
| `APP_ENV` | No | `production` por defecto. Usa `dev`, `development` o `local` solo en desarrollo. |
| `PORT` | No | Puerto HTTP (default `3000`). |
| `CORS_ORIGIN` | No | Origen permitido para CORS. |
| `DB_HOST` | No | Host de PostgreSQL. |
| `DB_PORT` | No | Puerto de PostgreSQL. |
| `DB_USER` | No | Usuario de PostgreSQL. |
| `DB_PASSWORD` | No | Contraseña de PostgreSQL. |
| `DB_NAME` | No | Base de datos. |
| `DB_SSLMODE` | No | Modo SSL de PostgreSQL. |
| `JWT_SECRET` | Sí | Secreto para firmar JWT. |
| `JWT_EXPIRES_IN` | No | Duración del JWT (default `12h`). |
| `MFA_ENCRYPTION_KEY` | Sí | Clave base64 de 32 bytes para cifrar semillas TOTP. |
| `KEK_MASTER` | No | Clave maestra base64 de 32 bytes para cifrado envelope. Sin ella, endpoints cifrados responden 503. |
| `KEK_VERSION` | No | Versión de KEK (default `1`). |
| `CURP_HMAC_KEY` | No | Llave HMAC para índice ciego de CURP. |
| `OTP_HMAC_KEY` | No | Llave HMAC para OTP; si falta, usa `JWT_SECRET`. |
| `INITIAL_ADMIN_USERNAME` | No | Usuario admin inicial para seed. |
| `INITIAL_ADMIN_EMAIL` | No | Email admin inicial para seed. |
| `INITIAL_ADMIN_PASSWORD` | No | Password admin inicial para seed. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | Endpoint OTLP/HTTP para trazas. Si no se define, el tracing queda en no-op. |

> No publiques secretos reales en el repositorio. Usa `.env` solo localmente o un gestor de secretos en despliegue.

## Ejecución local

### Opción A: Go directo

1. Levanta PostgreSQL y configura `.env`.
2. Ejecuta migraciones y seed (si aplica en tu flujo):

```bash
go run ./cmd/migrate
go run ./cmd/seed
```

3. Inicia la API:

```bash
go run ./cmd
```

La API quedará disponible en `http://localhost:3000` (o el puerto configurado).

### Opción B: Docker Compose

Levantar base de datos y backend:

```bash
docker compose up --build
```

Ejecutar herramientas de migración/seed:

```bash
docker compose --profile tools run --rm migrate
docker compose --profile tools run --rm seed
```

> Revisa `docker-compose.yml` antes de usarlo: el servicio backend expone `3001:3001` y define `PORT=3001`; valida que los healthchecks y puertos coincidan con tu entorno.

## Documentación de la API

Con la API corriendo, Swagger UI está disponible en:

```text
http://localhost:<PORT>/docs/
```

Para regenerar la documentación OpenAPI:

```bash
swag init -g cmd/main.go -d . -o docs --parseDependency --parseInternal
```

## Verificación

Ejecuta las comprobaciones del proyecto con:

```bash
gofmt -w .
go build ./...
go vet ./...
go test ./...
```

## Observabilidad

- **Métricas Prometheus:** `GET /metrics`
- **Estado/readiness:** `GET /api/status`
- **Salud:** `GET /health`
- **Trazas OTLP:** configurar `OTEL_EXPORTER_OTLP_ENDPOINT`
- **Artefactos:** `deploy/observability/prometheus-alerts.yml`, `deploy/observability/grafana-dashboard.json`

## Seguridad

El proyecto implementa controles de seguridad como:

- Hash de contraseñas con Argon2id.
- JWT con sesiones revocables y validación de versión de permisos.
- MFA TOTP.
- CSRF para operaciones de escritura.
- Rate limiting en endpoints sensibles.
- Cifrado envelope AES-256-GCM para datos sensibles.
- HMAC para OTP e índices ciegos.
- Auditoría con cadena de integridad y `request_id`.
- Separación de dominios de autenticación: staff, portal ciudadano y empleadores.

> Para despliegue real, endurece CORS, TLS, secretos, exposición de `/metrics`/`/docs`, y valida las políticas RLS en PostgreSQL.

## Equipo Git Pushers

Proyecto desarrollado por el equipo **Git Pushers** de la **Universidad Autónoma de Sinaloa, Facultad de Informática Mazatlán**, para el **Hackathon 2 FIM UAS**.

- Equipo: **Git Pushers**
- Institución: **Universidad Autónoma de Sinaloa (UAS)**
- Facultad: **Facultad de Ingenieria Mochis (FIM)**
- Evento: **Hackathon 2 FIM UAS**

## Licencia

Este repositorio fue desarrollado con fines académicos para el Hackathon 2 FIM UAS. Agrega un archivo `LICENSE` si el equipo define una licencia de distribución.
