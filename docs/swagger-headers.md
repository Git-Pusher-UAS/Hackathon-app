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
// @Failure      500  {object}  ErrorResponse
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
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /ruta [post]
func (h *MiHandler) NombreHandler(c *fiber.Ctx) error {
```

## 3. Endpoint protegido (JWT)

Añade `@Security BearerAuth` al bloque del endpoint:

```go
// @Security     BearerAuth
// @Failure      401  {object}  ErrorResponse
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
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

- Rutas dentro de `api := app.Group("/api")`: usar ruta relativa, por ejemplo `@Router /auth/login [post]` (sin `/api`).
- Rutas dentro de `protected`: añadir `@Security BearerAuth`.
- `/health` está fuera de `/api`: se documenta como `@Router /health [get]` y se aclara en la descripción.
- Usar los DTOs reales. Si el handler devuelve `fiber.Map`, crear un struct con tags `json` y `example`.
- Para errores usar `{object} ErrorResponse`.
- Después de agregar el endpoint, registrar la ruta en `cmd/main.go` y regenerar:

```bash
swag init -g cmd/main.go -d . --parseDependency --parseInternal
go build ./cmd/main.go
```
