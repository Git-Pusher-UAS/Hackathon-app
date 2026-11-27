# Cabeceras Swaggo para nuevos endpoints

Plantilla de anotaciones para copiar **encima del método handler**. No debe haber líneas en blanco entre el último `// @...` y la línea `func`.

## 1. Endpoint público (GET)

```go
// NombreHandler describe brevemente qué hace el endpoint.
//
// @Summary      Resumen corto
// @Description  Descripción detallada del comportamiento
// @Tags         nombre-del-dominio
// @Produce      json
// @Success      200  {object}  MiResponse
// @Failure      500  {object}  httpx.ErrorResponse
// @Router       /ruta [get]
func (h *MiHandler) NombreHandler(c *fiber.Ctx) error {
```

## 2. Endpoint público con body (POST)

```go
// NombreHandler describe brevemente qué hace el endpoint.
//
// @Summary      Resumen corto
// @Description  Descripción detallada del comportamiento
// @Tags         nombre-del-dominio
// @Accept       json
// @Produce      json
// @Param        request  body      MiRequest  true  "Datos de entrada"
// @Success      201      {object}  MiResponse
// @Failure      400      {object}  httpx.ErrorResponse
// @Failure      500      {object}  httpx.ErrorResponse
// @Router       /ruta [post]
func (h *MiHandler) NombreHandler(c *fiber.Ctx) error {
```

## 3. Endpoint protegido (JWT)

Añade `@Security BearerAuth` al bloque del endpoint:

```go
// @Security     BearerAuth
// @Failure      401  {object}  httpx.ErrorResponse
```

## 4. Parámetros

```go
// @Param id      path   string     true   "ID del recurso"
// @Param limit   query  int        false  "Tamaño de página"
// @Param cursor  query  string     false  "Cursor opaco"
// @Param request body   MiRequest  true   "Cuerpo de la petición"
```

## 5. Respuestas

```go
// @Success 200 {object} MiResponse
// @Success 200 {array}  models.Expediente
// @Success 204 "Sin contenido"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 429 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
```

## 6. Cabecera general de la API

Ya está definida en `cmd/main.go`; no se repite por endpoint:

```go
// @title           SAI API
// @version         1.0
// @description     Documentación de endpoints de la API.
// @host            localhost:3000
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
```

## 7. Reglas del proyecto

- Todo el contrato vive bajo `/api` (`@BasePath /api`): usar ruta relativa, por ejemplo `@Router /case-files [get]` (sin `/api` ni `/v1`).
- Rutas dentro de `priv` (protegidas): añadir `@Security BearerAuth`. Las escrituras (POST/PATCH/PUT/DELETE) llevan además `@Param X-CSRF-Token header string true "..."`.
- `GET /health` se sirve en `/health` (healthchecks de Docker y CI) y en `/api/health` (contrato).
- Rutas públicas (login, verificación MFA, directorio institucional): se registran en `cmd/main.go` **antes** del grupo `priv`, porque ese grupo se monta como middleware sobre todo `/api`.
- Usar los DTOs reales en `@Success`, nunca `{object} object`: el front genera sus tipos a partir de ellos. Si el handler devuelve `fiber.Map`, crear un struct o usar `object{campo=tipo}`.
- Para errores usar `{object} httpx.ErrorResponse` (sobre `{"error": {"code", "message", "requestId"}}`, el único formato de error de la API).
- Si un DTO vive en un paquete nuevo de `internal/models/<modulo>`, agregar su nombre legible en `pkgNames` de `cmd/openapi/main.go`.
- Después de agregar o cambiar un endpoint, registrar la ruta en `cmd/main.go` y regenerar Swagger **y** el contrato:

```bash
swag init -g cmd/main.go -d . --parseDependency --parseInternal
go run ./cmd/openapi          # escribe api/openapi.yaml (OpenAPI 3)
go run ./cmd/openapi -check   # verifica que api/openapi.yaml está al día
```

`api/openapi.yaml` es el contrato que el front sincroniza con `make sync-contract REF=contract-vN`. No se edita a mano.
