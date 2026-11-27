package employer

import (
	"errors"
	"log"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	emodels "backend-hackathon/internal/models/employer"
	svc "backend-hackathon/internal/services/employer"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service svc.Service }

func New(service svc.Service) *Handler { return &Handler{service: service} }

func mapErr(err error) error {
	var he *httpx.Error
	if errors.As(err, &he) {
		return he
	}
	switch {
	case errors.Is(err, svc.ErrNotFound):
		return httpx.E(fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "recurso no disponible")
	case errors.Is(err, svc.ErrInvalid):
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "codigo invalido o expirado")
	case errors.Is(err, svc.ErrConflict):
		return httpx.E(fiber.StatusConflict, "CONFLICT", "conflicto")
	case errors.Is(err, svc.ErrForbidden):
		return httpx.E(fiber.StatusForbidden, "FORBIDDEN", "sin acceso")
	default:
		log.Printf("employer: %v", err)
		return httpx.E(fiber.StatusInternalServerError, "INTERNAL_ERROR", "error interno")
	}
}
func setCookies(c *fiber.Ctx, token, csrf string) {
	secure := c.Secure() || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
	c.Cookie(&fiber.Cookie{Name: middleware.EmployerCookieName, Value: token, Path: "/", HTTPOnly: true, Secure: secure, SameSite: "Lax", MaxAge: int(24 * time.Hour / time.Second)})
	c.Cookie(&fiber.Cookie{Name: "employer_csrf_token", Value: csrf, Path: "/", HTTPOnly: false, Secure: secure, SameSite: "Lax", MaxAge: int(24 * time.Hour / time.Second)})
}

// Register godoc
// @Summary Registrar empleador
// @Tags employers
// @Accept json
// @Produce json
// @Param request body emodels.RegistrationRequest true "Registro"
// @Success 201 {object} emodels.RegistrationResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /employers/registrations [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req emodels.RegistrationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Register(c.Context(), req)
	if err != nil {
		return mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// Login godoc
// @Summary Login empleador (OTP)
// @Tags employers
// @Accept json
// @Produce json
// @Param request body emodels.LoginRequest true "Correo"
// @Success 200 {object} emodels.ChallengeResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /employers/auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req emodels.LoginRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Login(c.Context(), req.Correo)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// Verify godoc
// @Summary Verificar OTP empleador
// @Tags employers
// @Accept json
// @Produce json
// @Param request body emodels.VerifyRequest true "Desafío"
// @Success 200 {object} emodels.VerifyResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /employers/auth/verify [post]
func (h *Handler) Verify(c *fiber.Ctx) error {
	var req emodels.VerifyRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Verify(c.Context(), req.ChallengeID, req.Code)
	if err != nil {
		return mapErr(err)
	}
	setCookies(c, res.Token, res.CSRFToken)
	return c.JSON(emodels.VerifyResponse{EmployerID: res.EmployerID})
}

// MeRegistration godoc
// @Summary Ver registro empleador
// @Tags employers
// @Produce json
// @Success 200 {object} emodels.EmployerMe
// @Router /employers/me/registration [get]
func (h *Handler) MeRegistration(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	res, err := h.service.Me(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// UpdateRegistration godoc
// @Summary Actualizar registro empleador
// @Tags employers
// @Accept json
// @Produce json
// @Param request body emodels.UpdateRegistrationRequest true "Datos"
// @Success 200 {object} emodels.EmployerMe
// @Router /employers/me/registration [put]
func (h *Handler) UpdateRegistration(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var req emodels.UpdateRegistrationRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.UpdateRegistration(c.Context(), id, req)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// ListMyJobs godoc
// @Summary Listar vacantes del empleador
// @Tags employer-jobs
// @Produce json
// @Success 200 {array} emodels.JobView
// @Router /employer/jobs [get]
func (h *Handler) ListMyJobs(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	res, err := h.service.ListMyJobs(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// CreateMyJob godoc
// @Summary Crear vacante empleador
// @Tags employer-jobs
// @Accept json
// @Produce json
// @Param request body emodels.JobRequest true "Vacante"
// @Success 201 {object} emodels.JobView
// @Router /employer/jobs [post]
func (h *Handler) CreateMyJob(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	var req emodels.JobRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.CreateJob(c.Context(), id, req)
	if err != nil {
		return mapErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(res)
}

// GetMyJob godoc
// @Summary Obtener vacante empleador
// @Tags employer-jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} emodels.JobView
// @Router /employer/jobs/{id} [get]
func (h *Handler) GetMyJob(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.service.GetMyJob(c.Context(), id, jobID)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// PatchMyJob godoc
// @Summary Actualizar vacante empleador
// @Tags employer-jobs
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body emodels.JobRequest true "Vacante"
// @Success 200 {object} emodels.JobView
// @Router /employer/jobs/{id} [patch]
func (h *Handler) PatchMyJob(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	var req emodels.JobRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.PatchMyJob(c.Context(), id, jobID, req)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// ListJobApplications godoc
// @Summary Listar postulaciones de vacante
// @Tags employer-jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {array} emodels.ApplicationView
// @Router /employer/jobs/{id}/applications [get]
func (h *Handler) ListJobApplications(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.service.ListMyApplications(c.Context(), id, jobID)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// GetApplication godoc
// @Summary Ver postulación (empleador)
// @Tags employer-jobs
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} emodels.ApplicationView
// @Router /employer/applications/{id} [get]
func (h *Handler) GetApplication(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.service.GetApplication(c.Context(), id, appID)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// Interview godoc
// @Summary Agendar/registrar entrevista
// @Tags employer-jobs
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body emodels.InterviewRequest true "Entrevista"
// @Success 200 {object} emodels.ApplicationView
// @Router /employer/applications/{id}/interview [post]
func (h *Handler) Interview(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	var req emodels.InterviewRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Interview(c.Context(), id, appID, req)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}

// Decide godoc
// @Summary Decidir postulación
// @Tags employer-jobs
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body emodels.DecisionRequest true "Decisión"
// @Success 200 {object} emodels.ApplicationView
// @Router /employer/applications/{id}/decision [post]
func (h *Handler) Decide(c *fiber.Ctx) error {
	id, ok := middleware.EmpleadorID(c)
	if !ok {
		return httpx.E(fiber.StatusUnauthorized, "UNAUTHENTICATED", "sesion invalida")
	}
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "id invalido")
	}
	var req emodels.DecisionRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.service.Decide(c.Context(), id, appID, req)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(res)
}
