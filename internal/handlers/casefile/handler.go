// Package casefile expone los handlers HTTP de /api/case-files.
package casefile

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/casefile"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

func actorOf(c *fiber.Ctx) (repository.ActorContext, error) {
	a, ok := middleware.ActorFrom(c)
	if !ok {
		return a, httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesi\u00f3n inv\u00e1lida")
	}
	return a, nil
}

func uuidParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" inv\u00e1lido")
	}
	return id, nil
}

// X-Operation-Id/Idempotency-Key (header) u operationId (body) hace idempotentes las escrituras.
func operationID(c *fiber.Ctx, required bool) (*uuid.UUID, error) {
	return httpx.OperationIDFromRequest(c, required)
}

func parsePage(c *fiber.Ctx) (repository.PageRequest, error) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return repository.PageRequest{}, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "limit debe estar entre 1 y 100")
		}
		limit = n
	}
	return repository.PageRequest{Limit: limit, Cursor: c.Query("cursor")}, nil
}

func timeQuery(c *fiber.Ctx, name string) (*time.Time, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", name+" debe ser RFC3339")
	}
	return &t, nil
}

func mapError(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	var ve *svc.ValidationError
	if errors.As(err, &ve) {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", ve.Msg)
	}
	switch {
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, svc.ErrVersionMismatch):
		return httpx.E(fiber.StatusPreconditionFailed, "PRECONDITION_FAILED", "versi\u00f3n obsoleta")
	case errors.Is(err, svc.ErrPreconditionRequired):
		return httpx.E(fiber.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match requerido")
	case errors.Is(err, svc.ErrCryptoUnavailable):
		return httpx.E(fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "cifrado no configurado")
	case errors.As(err, new(*svc.DuplicatePersonError)):
		var dupErr *svc.DuplicatePersonError
		errors.As(err, &dupErr)
		e := httpx.E(fiber.StatusConflict, "PERSON_EXISTS", "la persona ya tiene expediente")
		e.Extra = map[string]any{"linkToken": dupErr.LinkToken, "linkExpiresAt": dupErr.LinkExpiresAt, "department": dupErr.Department}
		return e
	case errors.Is(err, svc.ErrDuplicatePerson):
		return httpx.E(fiber.StatusConflict, "PERSON_EXISTS", "la persona ya tiene expediente")
	case errors.Is(err, svc.ErrNoPrivacyNotice):
		return httpx.E(fiber.StatusConflict, "NO_PRIVACY_NOTICE", "no hay aviso de privacidad vigente")
	case errors.Is(err, svc.ErrInvalidTransition):
		return httpx.E(fiber.StatusConflict, "INVALID_TRANSITION", "transici\u00f3n de estado inv\u00e1lida")
	case errors.Is(err, svc.ErrCaseClosed):
		return httpx.E(fiber.StatusConflict, "CASE_CLOSED", "el expediente est\u00e1 cerrado")
	case errors.Is(err, svc.ErrRequestNotPending):
		return httpx.E(fiber.StatusConflict, "REQUEST_NOT_PENDING", "la solicitud ya fue resuelta")
	case errors.Is(err, svc.ErrAlreadyAssigned):
		return httpx.E(fiber.StatusConflict, "ALREADY_ASSIGNED", "ya existe una asignaci\u00f3n vigente")
	case errors.Is(err, svc.ErrAlreadyRevoked):
		return httpx.E(fiber.StatusConflict, "ALREADY_REVOKED", "ya fue revocado")
	case errors.Is(err, svc.ErrLastAssignment):
		return httpx.E(fiber.StatusConflict, "LAST_ASSIGNMENT", "el expediente no puede quedar sin personal asignado")
	case errors.Is(err, svc.ErrNotRevocable):
		return httpx.E(fiber.StatusConflict, "NOT_REVOCABLE", "la finalidad no es revocable")
	case errors.Is(err, svc.ErrReferralExists):
		return httpx.E(fiber.StatusConflict, "REFERRAL_EXISTS", "ya hay una canalizaci\u00f3n activa equivalente")
	case errors.Is(err, repository.ErrConflict):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "conflicto con el estado actual")
	default:
		log.Printf("casefile: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}

// DuplicateCheck verifica si ya existe un expediente asociado a un CURP.
//
// @Summary      Verificar duplicado por CURP
// @Description  Consulta si ya existe un expediente registrado con el CURP proporcionado.
// @Tags         case-files
// @Accept       json
// @Produce      json
// @Param        request  body      casemodels.DuplicateCheckRequest   true  "CURP a verificar"
// @Success      200      {object}  casemodels.DuplicateCheckResponse
// @Failure      400      {object}  httpx.ErrorResponse
// @Failure      401      {object}  httpx.ErrorResponse
// @Failure      422      {object}  httpx.ErrorResponse
// @Failure      500      {object}  httpx.ErrorResponse
// @Router       /case-files/duplicate-check [post]
func (h *Handler) DuplicateCheck(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	var req casemodels.DuplicateCheckRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	res, err := h.service.DuplicateCheck(c.Context(), actor, req.CURP)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Create crea un nuevo expediente de forma idempotente y devuelve su ETag de versión.
//
// @Summary      Crear expediente
// @Description  Crea un expediente nuevo requiriendo X-Operation-Id para idempotencia.
// @Tags         case-files
// @Accept       json
// @Produce      json
// @Param        X-Operation-Id  header    string                 true  "UUID de operación idempotente"
// @Param        request         body      casemodels.CreateRequest  true  "Datos del expediente"
// @Success      201             {object}  object
// @Header       201             {string}  ETag                    "Versión actual del recurso"
// @Failure      400             {object}  httpx.ErrorResponse
// @Failure      401             {object}  httpx.ErrorResponse
// @Failure      409             {object}  httpx.ErrorResponse
// @Failure      422             {object}  httpx.ErrorResponse
// @Failure      500             {object}  httpx.ErrorResponse
// @Failure      503             {object}  httpx.ErrorResponse
// @Router       /case-files [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	op, err := operationID(c, true)
	if err != nil {
		return err
	}
	var req casemodels.CreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	res, err := h.service.Create(c.Context(), actor, *op, req)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(res.VersionTag))
	c.Set(fiber.HeaderLocation, "/api/case-files/"+res.ID.String())
	return c.Status(fiber.StatusCreated).JSON(res)
}

// List lista expedientes con paginación por cursor y filtros opcionales.
//
// @Summary      Listar expedientes
// @Description  Obtiene una página de expedientes filtrable por estado y alias.
// @Tags         case-files
// @Produce      json
// @Param        limit   query     int     false  "Tamaño de página (1-100, default 20)"
// @Param        cursor  query     string  false  "Cursor de paginación"
// @Param        estado  query     string  false  "Estado del expediente"
// @Param        alias   query     string  false  "Alias de búsqueda"
// @Success      200     {object}  object
// @Failure      401     {object}  httpx.ErrorResponse
// @Failure      422     {object}  httpx.ErrorResponse
// @Failure      500     {object}  httpx.ErrorResponse
// @Router       /case-files [get]
func (h *Handler) List(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	page, err := parsePage(c)
	if err != nil {
		return err
	}
	q := casemodels.ListQuery{Estado: models.EstadoExpediente(c.Query("estado")), Alias: c.Query("alias")}
	res, err := h.service.List(c.Context(), actor, q, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Get obtiene el detalle de un expediente por su UUID e incluye su ETag.
//
// @Summary      Obtener expediente
// @Description  Devuelve los datos de un expediente por recordId y su cabecera ETag.
// @Tags         case-files
// @Produce      json
// @Param        recordId  path      string  true  "UUID del expediente"
// @Success      200       {object}  object
// @Header       200       {string}  ETag                 "Versión actual del recurso"
// @Failure      401       {object}  httpx.ErrorResponse
// @Failure      404       {object}  httpx.ErrorResponse
// @Failure      422       {object}  httpx.ErrorResponse
// @Failure      500       {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	res, err := h.service.Get(c.Context(), actor, id)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(res.VersionTag))
	return c.JSON(res)
}

// Update actualiza un expediente validando concurrencia optimista mediante If-Match.
//
// @Summary      Actualizar expediente
// @Description  Actualiza un expediente requiriendo la cabecera If-Match con la versión esperada.
// @Tags         case-files
// @Accept       json
// @Produce      json
// @Param        recordId  path      string                 true  "UUID del expediente"
// @Param        If-Match  header    string                 true  "ETag / versión actual del recurso"
// @Param        request   body      casemodels.UpdateRequest  true  "Campos a actualizar"
// @Success      200       {object}  object
// @Header       200       {string}  ETag                      "Nueva versión del recurso"
// @Failure      400       {object}  httpx.ErrorResponse
// @Failure      401       {object}  httpx.ErrorResponse
// @Failure      404       {object}  httpx.ErrorResponse
// @Failure      409       {object}  httpx.ErrorResponse
// @Failure      412       {object}  httpx.ErrorResponse
// @Failure      422       {object}  httpx.ErrorResponse
// @Failure      500       {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	ifMatch, ok := httpx.ParseIfMatch(c.Get(fiber.HeaderIfMatch))
	var req casemodels.UpdateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	if req.Estado == nil && req.Status != nil {
		req.Estado = req.Status
	}
	if !ok {
		if v := strings.TrimSpace(req.VersionTag); v != "" {
			ifMatch, ok = v, true
		} else if req.Version != nil && *req.Version > 0 {
			ifMatch, ok = strconv.FormatInt(*req.Version, 10), true
		}
	}
	if !ok {
		return httpx.E(fiber.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match requerido")
	}
	res, err := h.service.Update(c.Context(), actor, id, req, ifMatch)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(res.VersionTag))
	return c.JSON(res)
}

// Timeline consulta el historial cronológico de eventos de un expediente.
//
// @Summary      Línea de tiempo del expediente
// @Description  Lista eventos del expediente filtrables por tipos, rango de fechas RFC3339 y paginación.
// @Tags         case-files
// @Produce      json
// @Param        recordId  path      string  true   "UUID del expediente"
// @Param        limit     query     int     false  "Tamaño de página (1-100, default 20)"
// @Param        cursor    query     string  false  "Cursor de paginación"
// @Param        tipos     query     string  false  "Tipos de evento separados por coma"
// @Param        desde     query     string  false  "Fecha inicial en formato RFC3339"
// @Param        hasta     query     string  false  "Fecha final en formato RFC3339"
// @Success      200       {object}  object
// @Failure      401       {object}  httpx.ErrorResponse
// @Failure      404       {object}  httpx.ErrorResponse
// @Failure      422       {object}  httpx.ErrorResponse
// @Failure      500       {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/timeline [get]
func (h *Handler) Timeline(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	page, err := parsePage(c)
	if err != nil {
		return err
	}
	q := casemodels.TimelineQuery{}
	if raw := c.Query("tipos"); raw != "" {
		q.Tipos = strings.Split(raw, ",")
	}
	if q.Desde, err = timeQuery(c, "desde"); err != nil {
		return err
	}
	if q.Hasta, err = timeQuery(c, "hasta"); err != nil {
		return err
	}
	res, err := h.service.Timeline(c.Context(), actor, id, q, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Disclose revela un campo protegido o sensible de un expediente según permisos y reglas.
//
// @Summary      Revelar campo del expediente
// @Description  Solicita la divulgación o consulta de un campo específico del expediente.
// @Tags         case-files
// @Accept       json
// @Produce      json
// @Param        recordId  path      string                    true  "UUID del expediente"
// @Param        request   body      casemodels.DisclosureRequest  true  "Campo a revelar"
// @Success      200       {object}  object
// @Failure      400       {object}  httpx.ErrorResponse
// @Failure      401       {object}  httpx.ErrorResponse
// @Failure      404       {object}  httpx.ErrorResponse
// @Failure      409       {object}  httpx.ErrorResponse
// @Failure      422       {object}  httpx.ErrorResponse
// @Failure      500       {object}  httpx.ErrorResponse
// @Failure      503       {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/disclosures [post]
func (h *Handler) Disclose(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	var req casemodels.DisclosureRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	if strings.TrimSpace(req.Campo) == "" && strings.TrimSpace(req.Field) != "" {
		req.Campo = req.Field
	}
	res, err := h.service.Disclose(c.Context(), actor, id, req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// Assign asigna personal o un responsable al expediente.
//
// @Summary      Asignar personal al expediente
// @Description  Crea una asignación activa sobre el expediente indicado.
// @Tags         case-files-assignments
// @Accept       json
// @Produce      json
// @Param        recordId  path      string                 true  "UUID del expediente"
// @Param        request   body      casemodels.AssignRequest  true  "Datos de asignación"
// @Success      201       {object}  object
// @Failure      400       {object}  httpx.ErrorResponse
// @Failure      401       {object}  httpx.ErrorResponse
// @Failure      404       {object}  httpx.ErrorResponse
// @Failure      409       {object}  httpx.ErrorResponse
// @Failure      422       {object}  httpx.ErrorResponse
// @Failure      500       {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/assignments [post]
func (h *Handler) Assign(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	var req casemodels.AssignRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	if strings.TrimSpace(req.PersonalID) == "" && strings.TrimSpace(req.StaffID) != "" {
		req.PersonalID = req.StaffID
	}
	if req.SolicitudID == nil && req.LinkRequestID != nil {
		req.SolicitudID = req.LinkRequestID
	}
	res, err := h.service.Assign(c.Context(), actor, id, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// RevokeAssignment revoca una asignación existente en el expediente.
//
// @Summary      Revocar asignación
// @Description  Revoca una asignación específica dentro de un expediente.
// @Tags         case-files-assignments
// @Produce      json
// @Param        recordId      path      string  true  "UUID del expediente"
// @Param        assignmentId  path      string  true  "UUID de la asignación"
// @Success      204           {string}  string  ""
// @Failure      401           {object}  httpx.ErrorResponse
// @Failure      404           {object}  httpx.ErrorResponse
// @Failure      409           {object}  httpx.ErrorResponse
// @Failure      422           {object}  httpx.ErrorResponse
// @Failure      500           {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/assignments/{assignmentId}/revocations [post]
func (h *Handler) RevokeAssignment(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	aid, err := uuidParam(c, "assignmentId")
	if err != nil {
		return err
	}
	if err := h.service.RevokeAssignment(c.Context(), actor, id, aid); err != nil {
		return mapError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListNotes lista las notas de un expediente filtradas por departamento y opcionalmente por episodio.
//
// @Summary      Listar notas por departamento
// @Description  Obtiene notas paginadas de un expediente para un departamento dado.
// @Tags         case-files-notes
// @Produce      json
// @Param        recordId      path      string  true   "UUID del expediente"
// @Param        departmentId  path      string  true   "UUID del departamento"
// @Param        limit         query     int     false  "Tamaño de página (1-100, default 20)"
// @Param        cursor        query     string  false  "Cursor de paginación"
// @Param        episodeId     query     string  false  "UUID del episodio (opcional)"
// @Success      200           {object}  object
// @Failure      401           {object}  httpx.ErrorResponse
// @Failure      404           {object}  httpx.ErrorResponse
// @Failure      422           {object}  httpx.ErrorResponse
// @Failure      500           {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/departments/{departmentId}/notes [get]
func (h *Handler) ListNotes(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	dep, err := uuidParam(c, "departmentId")
	if err != nil {
		return err
	}
	page, err := parsePage(c)
	if err != nil {
		return err
	}
	var ep *uuid.UUID
	if raw := c.Query("episodeId"); raw != "" {
		parsed, perr := uuid.Parse(raw)
		if perr != nil {
			return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "episodeId inv\u00e1lido")
		}
		ep = &parsed
	}
	res, err := h.service.ListNotes(c.Context(), actor, id, dep, ep, page)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(res)
}

// CreateNote crea una nueva nota de forma idempotente en un expediente y departamento.
//
// @Summary      Crear nota
// @Description  Registra una nota en un expediente y departamento usando X-Operation-Id.
// @Tags         case-files-notes
// @Accept       json
// @Produce      json
// @Param        recordId      path      string                     true  "UUID del expediente"
// @Param        departmentId  path      string                     true  "UUID del departamento"
// @Param        X-Operation-Id  header    string                     true  "UUID de operación idempotente"
// @Param        request       body      casemodels.NoteCreateRequest  true  "Contenido de la nota"
// @Success      201           {object}  object
// @Failure      400           {object}  httpx.ErrorResponse
// @Failure      401           {object}  httpx.ErrorResponse
// @Failure      404           {object}  httpx.ErrorResponse
// @Failure      409           {object}  httpx.ErrorResponse
// @Failure      422           {object}  httpx.ErrorResponse
// @Failure      500           {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/departments/{departmentId}/notes [post]
func (h *Handler) CreateNote(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	dep, err := uuidParam(c, "departmentId")
	if err != nil {
		return err
	}
	op, err := operationID(c, true)
	if err != nil {
		return err
	}
	var req casemodels.NoteCreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	res, err := h.service.CreateNote(c.Context(), actor, id, dep, *op, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// ListConsents lista los consentimientos registrados para un expediente.
//
// @Summary      Listar consentimientos
// @Description  Devuelve los consentimientos asociados a un expediente bajo la clave items.
// @Tags         case-files-consents
// @Produce      json
// @Param        recordId  path      string          true  "UUID del expediente"
// @Success      200       {object}  map[string]any
// @Failure      401       {object}  httpx.ErrorResponse
// @Failure      404       {object}  httpx.ErrorResponse
// @Failure      422       {object}  httpx.ErrorResponse
// @Failure      500       {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/consents [get]
func (h *Handler) ListConsents(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	res, err := h.service.ListConsents(c.Context(), actor, id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"items": res})
}

// RecordConsent registra un nuevo consentimiento en el expediente.
//
// @Summary      Registrar consentimiento
// @Description  Guarda un consentimiento con cabecera X-Operation-Id opcional.
// @Tags         case-files-consents
// @Accept       json
// @Produce      json
// @Param        recordId      path      string                  true   "UUID del expediente"
// @Param        X-Operation-Id  header    string                  false  "UUID de operación idempotente (opcional)"
// @Param        request       body      casemodels.ConsentInput true   "Datos del consentimiento"
// @Success      201           {object}  map[string]any
// @Failure      400           {object}  httpx.ErrorResponse
// @Failure      401           {object}  httpx.ErrorResponse
// @Failure      404           {object}  httpx.ErrorResponse
// @Failure      409           {object}  httpx.ErrorResponse
// @Failure      422           {object}  httpx.ErrorResponse
// @Failure      500           {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/consents [post]
func (h *Handler) RecordConsent(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	op, err := operationID(c, false)
	if err != nil {
		return err
	}
	var req casemodels.ConsentInput
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	res, err := h.service.RecordConsent(c.Context(), actor, id, op, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"items": res})
}

// RevokeConsent revoca un consentimiento específico de un expediente.
//
// @Summary      Revocar consentimiento
// @Description  Registra la revocación de un consentimiento identificado por consentId.
// @Tags         case-files-consents
// @Accept       json
// @Produce      json
// @Param        recordId   path      string                            true  "UUID del expediente"
// @Param        consentId  path      string                            true  "UUID del consentimiento"
// @Param        request    body      casemodels.RevokeConsentRequest   true  "Motivo o datos de revocación"
// @Success      201        {object}  map[string]any
// @Failure      400        {object}  httpx.ErrorResponse
// @Failure      401        {object}  httpx.ErrorResponse
// @Failure      404        {object}  httpx.ErrorResponse
// @Failure      409        {object}  httpx.ErrorResponse
// @Failure      422        {object}  httpx.ErrorResponse
// @Failure      500        {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/consents/{consentId}/revocations [post]
func (h *Handler) RevokeConsent(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	cid, err := uuidParam(c, "consentId")
	if err != nil {
		return err
	}
	var req casemodels.RevokeConsentRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	res, err := h.service.RevokeConsent(c.Context(), actor, id, cid, req)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"items": res})
}

// CreateReferral crea una canalización de forma idempotente sobre el expediente.
//
// @Summary      Crear canalización
// @Description  Registra una canalización para el expediente requiriendo X-Operation-Id y devuelve ETag.
// @Tags         case-files-referrals
// @Accept       json
// @Produce      json
// @Param        recordId      path      string                        true  "UUID del expediente"
// @Param        X-Operation-Id  header    string                        true  "UUID de operación idempotente"
// @Param        request       body      casemodels.ReferralCreateRequest  true  "Datos de la canalización"
// @Success      201           {object}  object
// @Header       201           {string}  ETag                          "Versión actual del recurso"
// @Failure      400           {object}  httpx.ErrorResponse
// @Failure      401           {object}  httpx.ErrorResponse
// @Failure      404           {object}  httpx.ErrorResponse
// @Failure      409           {object}  httpx.ErrorResponse
// @Failure      422           {object}  httpx.ErrorResponse
// @Failure      500           {object}  httpx.ErrorResponse
// @Router       /case-files/{recordId}/referrals [post]
func (h *Handler) CreateReferral(c *fiber.Ctx) error {
	actor, err := actorOf(c)
	if err != nil {
		return err
	}
	id, err := uuidParam(c, "recordId")
	if err != nil {
		return err
	}
	op, err := operationID(c, true)
	if err != nil {
		return err
	}
	var req casemodels.ReferralCreateRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return mapError(err)
	}
	res, err := h.service.CreateReferral(c.Context(), actor, id, *op, req)
	if err != nil {
		return mapError(err)
	}
	c.Set(fiber.HeaderETag, httpx.FormatETag(res.VersionTag))
	return c.Status(fiber.StatusCreated).JSON(res)
}
