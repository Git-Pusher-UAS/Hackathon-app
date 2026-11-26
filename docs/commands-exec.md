# Guía de comandos — Backend Hackathon

Comandos para levantar, migrar, sembrar y probar la API en local y en Docker.

## 1. Requisitos

- Go **1.27+**
- Docker y Docker Compose v2
- Puertos libres:
  - **3001** → API
  - **5433** → PostgreSQL (en el host; dentro de Docker usa 5432)


## 3. Flujo local (Go + Postgres en Docker)

```bash
# 1. Levantar solo la base de datos
docker compose up -d db
docker compose ps

# 2. Crear esquemas, tablas, vistas e índices
go run ./cmd/migrate

# 3. Crear rol, permiso y cuenta inicial (idempotente)
go run ./cmd/seed

# 4. Arrancar la API
go run ./cmd/
```

La API queda en <http://localhost:3001>.

> No ejecutes `go run ./cmd/` si el contenedor `hackathon-api` está levantado: ambos usan el puerto 3001.

## 4. Flujo 100% Docker

```bash
# Construir y levantar db + backend
docker compose up -d --build

# Aplicar migraciones
docker compose --profile tools run --rm migrate

# Crear datos iniciales
docker compose --profile tools run --rm seed

# Estado de los contenedores
docker compose ps
```

En un volumen nuevo, el orden es **migrate → seed → backend**.

Contenedores esperados:

```text
hackathon-db    Up (healthy)   5433->5432
hackathon-api   Up (healthy)   3001->3001
```

## 5. Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| GET | `/health` | — | Liveness (no toca la BD) |
| GET | `/api/status` | — | Readiness + ping a la BD |
| POST | `/api/auth/login` | — | Login con usuario/contraseña |
| GET | `/api/auth/me` | Bearer | Cuenta autenticada y permisos |
| POST | `/api/auth/logout` | Bearer | Revoca la sesión actual |

### Códigos de respuesta

| Código | Cuándo |
|---|---|
| 200 | Operación correcta |
| 204 | Logout correcto |
| 400 | Cuerpo de la petición inválido |
| 401 | Credenciales o token inválidos |
| 403 | Cuenta suspendida, MFA activo (`mfa_requerido`) o permiso insuficiente |
| 429 | Cuenta bloqueada o demasiados intentos |
| 500 | Error interno |

## 6. Probar la autenticación

Requiere `jq` (opcional, para extraer el token):

```bash
# Login
curl -s -X POST http://localhost:3001/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"usuario":"admin","password":"admin123"}'

# Guardar el token en una variable
TOKEN=$(curl -s -X POST http://localhost:3001/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"usuario":"admin","password":"admin123"}' | jq -r .token)

# Consultar la cuenta
curl -s http://localhost:3001/api/auth/me \
  -H "Authorization: Bearer $TOKEN"

# Logout (revoca la sesión)
curl -s -o /dev/null -w '%{http_code}\n' -X POST http://localhost:3001/api/auth/logout \
  -H "Authorization: Bearer $TOKEN"

# Reutilizar el token debe devolver 401
curl -s http://localhost:3001/api/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

Notas:

- Tras **5 intentos fallidos** en 15 minutos, el login responde **429**.
- Una cuenta con MFA activo responde **403** `mfa_requerido` (MFA aún no implementado).

## 7. Base de datos

Entrar al contenedor:

```bash
docker exec -it hackathon-db psql -U postgres -d sai
```

Consultas útiles:

```sql
-- Esquemas y tablas
\dn
\dt *.*

-- Vistas
SELECT table_schema, table_name
FROM information_schema.views
WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY 1, 2;

-- Cuentas
SELECT usuario, estado, fallos_consecutivos, bloqueado_hasta
FROM identidad.personal;

-- Sesiones
SELECT id, titular_id, revocado_en, vence_absoluto
FROM identidad.sesiones_personal
ORDER BY creado_en DESC;

-- Auditoría
SELECT secuencia, accion, resultado, ocurrido_en
FROM auditoria.eventos
ORDER BY secuencia DESC
LIMIT 10;
```

## 8. Mantenimiento

```bash
# Logs
docker compose logs -f backend
docker compose logs -f db

# Reiniciar solo la API
docker compose restart backend

# Apagar conservando datos
docker compose down

# Apagar y borrar la base de datos (DESTRUCTIVO)
docker compose down -v

# Reconstruir la imagen sin caché
docker compose build --no-cache backend

# Reaplicar migraciones (idempotente)
go run ./cmd/migrate
# o
docker compose --profile tools run --rm migrate
```

## 9. Pruebas y calidad

```bash
go build ./...
go vet ./...
gofmt -l cmd internal
go test ./...
```

## 10. Problemas comunes

- **`address already in use` en 3001**: hay un contenedor o proceso usando el puerto. Detén uno de los dos:
  ```bash
  docker compose stop backend
  # o mata el proceso local ```
- **`Warning: .env file not found`** dentro de un contenedor: es normal. El `.env` no se copia a la imagen; las variables llegan por `env_file`.
- **Login 401 después del seed**: confirma que corriste `migrate` antes de `seed` y que usuario/contraseña coinciden con `INITIAL_ADMIN_*`.
- **429 al probar muchas veces**: espera 15 minutos o limpia el estado en desarrollo:
  ```sql
  DELETE FROM identidad.control_intentos;
  UPDATE identidad.personal
  SET fallos_consecutivos = 0, bloqueado_hasta = NULL;
  ```
- **La base no responde**: revisa `docker compose ps` y que `hackathon-db` esté `healthy`.

## 11. Estructura relevante

```text
cmd/
  main.go          API (Fiber)
  migrate/main.go  esquemas + AutoMigrate + vistas + índice único
  seed/main.go     rol admin + permiso + cuenta inicial

internal/
  auth/            Argon2id, JWT, servicio de login/logout
  config/          carga de .env
  handlers/        health, status, auth
  middleware/      AuthMiddleware, RequirePermiso
  models/          modelos GORM
  repository/
    gorm/          implementación GORM + UnitOfWork
    repositories/  interfaces y contratos
```
