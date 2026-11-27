package bolsaextra

import (
	"encoding/json"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"
	"backend-hackathon/internal/models"
	repository "backend-hackathon/internal/repository/repositories"
	adminsvc "backend-hackathon/internal/services/adminjobs"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	uow   repository.UnitOfWork
	admin *adminsvc.Service
}

func New(uow repository.UnitOfWork) *Handler { return &Handler{uow: uow, admin: adminsvc.New(uow)} }

// AdminListJobs godoc
// @Summary Listar vacantes (admin)
// @Tags admin-jobs
// @Produce json
// @Success 200 {array} models.Vacante
// @Router /admin/jobs [get]
func (h *Handler) AdminListJobs(c *fiber.Ctx) error {
	res, err := h.admin.ListJobs(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(res)
}

// AdminCreateJob godoc
// @Summary Crear vacante (admin)
// @Tags admin-jobs
// @Accept json
// @Produce json
// @Param request body map[string]any true "Vacante"
// @Success 201 {object} models.Vacante
// @Router /admin/jobs [post]
func (h *Handler) AdminCreateJob(c *fiber.Ctx) error {
	var in map[string]any
	if err := httpx.DecodeJSON(c, &in); err != nil {
		return err
	}
	res, err := h.admin.CreateJob(c.Context(), in)
	if err != nil {
		return err
	}
	return c.Status(201).JSON(res)
}

// AdminGetJob godoc
// @Summary Obtener vacante (admin)
// @Tags admin-jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} models.Vacante
// @Router /admin/jobs/{id} [get]
func (h *Handler) AdminGetJob(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	res, err := h.admin.GetJob(c.Context(), id)
	if err != nil {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrado")
	}
	return c.JSON(res)
}

// AdminReviewJob godoc
// @Summary Revisar vacante (admin)
// @Tags admin-jobs
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body map[string]string true "Decisión"
// @Success 200 {object} models.Vacante
// @Router /admin/jobs/{id}/review [post]
func (h *Handler) AdminReviewJob(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	var in map[string]string
	if err := httpx.DecodeJSON(c, &in); err != nil {
		return err
	}
	res, err := h.admin.ReviewJob(c.Context(), id, in["decision"], in["motivo"])
	if err != nil {
		return err
	}
	return c.JSON(res)
}

// AdminListEmployers godoc
// @Summary Listar empleadores (admin)
// @Tags admin-jobs
// @Produce json
// @Success 200 {array} models.Empleador
// @Router /admin/employers [get]
func (h *Handler) AdminListEmployers(c *fiber.Ctx) error {
	res, err := h.admin.ListEmployers(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(res)
}

// AdminReviewEmployer godoc
// @Summary Revisar empleador (admin)
// @Tags admin-jobs
// @Accept json
// @Produce json
// @Param id path string true "Employer ID"
// @Param request body map[string]string true "Decisión"
// @Success 200 {object} models.Empleador
// @Router /admin/employers/{id}/review [post]
func (h *Handler) AdminReviewEmployer(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	var in map[string]string
	if err := httpx.DecodeJSON(c, &in); err != nil {
		return err
	}
	res, err := h.admin.ReviewEmployer(c.Context(), id, in["decision"], in["motivo"])
	if err != nil {
		return err
	}
	return c.JSON(res)
}

// AdminHires godoc
// @Summary Listar contrataciones
// @Tags admin-jobs
// @Produce json
// @Success 200 {array} models.Postulacion
// @Router /admin/hires [get]
func (h *Handler) AdminHires(c *fiber.Ctx) error {
	res, err := h.admin.ListHires(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(res)
}

// AdminContactRelease godoc
// @Summary Liberar contacto de postulación
// @Tags admin-jobs
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body map[string]any true "Motivo y campos"
// @Success 200 {object} models.Postulacion
// @Router /admin/applications/{id}/contact-release [post]
func (h *Handler) AdminContactRelease(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	var in struct {
		Motivo string   `json:"motivo"`
		Campos []string `json:"campos"`
	}
	if err := httpx.DecodeJSON(c, &in); err != nil {
		return err
	}
	res, err := h.admin.ContactRelease(c.Context(), actor, id, in.Motivo, in.Campos)
	if err != nil {
		return err
	}
	return c.JSON(res)
}

type cvPayload struct {
	Perfil struct {
		Nombre   string `json:"nombre"`
		Titular  string `json:"titular"`
		Resumen  string `json:"resumen"`
		Contacto string `json:"contacto"`
	} `json:"perfil"`
	Experiencia []struct {
		Empresa     string `json:"empresa"`
		Puesto      string `json:"puesto"`
		Inicio      string `json:"inicio"`
		Fin         string `json:"fin"`
		Descripcion string `json:"descripcion"`
	} `json:"experiencia"`
	Estudios []struct {
		Institucion string `json:"institucion"`
		Grado       string `json:"grado"`
		Inicio      string `json:"inicio"`
		Fin         string `json:"fin"`
	} `json:"estudios"`
	Habilidades []string `json:"habilidades"`
}

// GetCV godoc
// @Summary Obtener mi CV
// @Tags portal-cv
// @Produce json
// @Success 200 {object} cvPayload
// @Router /portal/me/cv [get]
func (h *Handler) GetCV(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	r := h.uow.Repositories()
	p, err := r.CV().GetPerfil(c.Context(), cuentaID)
	if err != nil {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "cv no encontrado")
	}
	exps, _ := r.CV().ListExperiencias(c.Context(), cuentaID)
	eds, _ := r.CV().ListEstudios(c.Context(), cuentaID)
	var hab []string
	_ = json.Unmarshal([]byte(p.Habilidades), &hab)
	out := cvPayload{}
	out.Perfil.Nombre = p.Nombre
	out.Perfil.Titular = p.Titular
	out.Perfil.Resumen = p.Resumen
	out.Perfil.Contacto = p.Contacto
	out.Habilidades = hab
	for _, e := range exps {
		out.Experiencia = append(out.Experiencia, struct {
			Empresa     string `json:"empresa"`
			Puesto      string `json:"puesto"`
			Inicio      string `json:"inicio"`
			Fin         string `json:"fin"`
			Descripcion string `json:"descripcion"`
		}{e.Empresa, e.Puesto, e.Inicio, e.Fin, e.Descripcion})
	}
	for _, e := range eds {
		out.Estudios = append(out.Estudios, struct {
			Institucion string `json:"institucion"`
			Grado       string `json:"grado"`
			Inicio      string `json:"inicio"`
			Fin         string `json:"fin"`
		}{e.Institucion, e.Grado, e.Inicio, e.Fin})
	}
	return c.JSON(out)
}

// PutCV godoc
// @Summary Reemplazar mi CV
// @Tags portal-cv
// @Accept json
// @Produce json
// @Param request body cvPayload true "CV"
// @Success 200 {object} cvPayload
// @Router /portal/me/cv [put]
func (h *Handler) PutCV(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	var in cvPayload
	if err := httpx.DecodeJSON(c, &in); err != nil {
		return err
	}
	if strings.TrimSpace(in.Perfil.Nombre) == "" {
		return httpx.Invalid("cv invalido", map[string]string{"perfil.nombre": "requerido"})
	}
	hab, _ := json.Marshal(in.Habilidades)
	r := h.uow.Repositories()
	p := &models.CVPerfil{ID: uuid.New(), CuentaID: cuentaID, Nombre: in.Perfil.Nombre, Titular: in.Perfil.Titular, Resumen: in.Perfil.Resumen, Contacto: in.Perfil.Contacto, Habilidades: string(hab), ActualizadoEn: time.Now(), CreadoEn: time.Now()}
	if err := r.CV().UpsertPerfil(c.Context(), p); err != nil {
		return err
	}
	exps := make([]models.CVExperiencia, 0, len(in.Experiencia))
	for _, e := range in.Experiencia {
		exps = append(exps, models.CVExperiencia{ID: uuid.New(), CuentaID: cuentaID, Empresa: e.Empresa, Puesto: e.Puesto, Inicio: e.Inicio, Fin: e.Fin, Descripcion: e.Descripcion})
	}
	eds := make([]models.CVEstudio, 0, len(in.Estudios))
	for _, e := range in.Estudios {
		eds = append(eds, models.CVEstudio{ID: uuid.New(), CuentaID: cuentaID, Institucion: e.Institucion, Grado: e.Grado, Inicio: e.Inicio, Fin: e.Fin})
	}
	if err := r.CV().ReplaceExperiencias(c.Context(), cuentaID, exps); err != nil {
		return err
	}
	if err := r.CV().ReplaceEstudios(c.Context(), cuentaID, eds); err != nil {
		return err
	}
	return h.GetCV(c)
}

// MyApplications godoc
// @Summary Mis postulaciones
// @Tags portal-applications
// @Produce json
// @Success 200 {array} object
// @Router /portal/me/applications [get]
func (h *Handler) MyApplications(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	rows, err := h.uow.Repositories().Postulaciones().ListByUsuaria(c.Context(), cuentaID)
	if err != nil {
		return err
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, p := range rows {
		v, _ := h.uow.Repositories().Vacantes().GetByID(c.Context(), p.VacanteID)
		puesto, empresa := "", ""
		if v != nil {
			puesto = v.Puesto
			empresa = v.Empresa
		}
		out = append(out, fiber.Map{"id": p.ID, "vacanteId": p.VacanteID, "puesto": puesto, "empresa": empresa, "estado": p.Estado, "creadoEn": p.CreadoEn})
	}
	return c.JSON(out)
}

// MyApplication godoc
// @Summary Detalle de mi postulación
// @Tags portal-applications
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} object
// @Router /portal/me/applications/{id} [get]
func (h *Handler) MyApplication(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	p, err := h.uow.Repositories().Postulaciones().GetByID(c.Context(), id)
	if err != nil {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrada")
	}
	if p.UsuariaID != cuentaID {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrada")
	}
	v, _ := h.uow.Repositories().Vacantes().GetByID(c.Context(), p.VacanteID)
	puesto, empresa := "", ""
	if v != nil {
		puesto = v.Puesto
		empresa = v.Empresa
	}
	return c.JSON(fiber.Map{"id": p.ID, "vacanteId": p.VacanteID, "puesto": puesto, "empresa": empresa, "estado": p.Estado, "creadoEn": p.CreadoEn, "contactoLiberado": p.ContactoLiberado})
}

// InterviewResponse godoc
// @Summary Responder entrevista
// @Tags portal-applications
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body map[string]string true "Respuesta"
// @Success 200 {object} object
// @Router /portal/me/applications/{id}/interview-response [post]
func (h *Handler) InterviewResponse(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	var in struct {
		Respuesta string `json:"respuesta"`
	}
	if err := httpx.DecodeJSON(c, &in); err != nil {
		return err
	}
	r := h.uow.Repositories()
	p, err := r.Postulaciones().GetByID(c.Context(), id)
	if err != nil {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrada")
	}
	if p.UsuariaID != cuentaID {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrada")
	}
	switch strings.ToLower(strings.TrimSpace(in.Respuesta)) {
	case "accept", "accepted":
		p.Estado = models.PostulacionEntrevistaAceptada
	case "decline", "declined", "reject", "rejected":
		p.Estado = models.PostulacionEntrevistaRechazada
	default:
		return httpx.Invalid("respuesta invalida", map[string]string{"respuesta": "valor_invalido"})
	}
	p.ActualizadoEn = time.Now()
	if err := r.Postulaciones().Save(c.Context(), p); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"id": p.ID, "estado": p.Estado})
}

// Withdraw godoc
// @Summary Retirar postulación
// @Tags portal-applications
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body map[string]string true "Motivo"
// @Success 200 {object} object
// @Router /portal/me/applications/{id}/withdrawal [post]
func (h *Handler) Withdraw(c *fiber.Ctx) error {
	cuentaID, ok := middleware.PortalCuentaID(c)
	if !ok {
		return httpx.E(401, "UNAUTHENTICATED", "sesion invalida")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.E(422, "VALIDATION_ERROR", "id invalido")
	}
	r := h.uow.Repositories()
	p, err := r.Postulaciones().GetByID(c.Context(), id)
	if err != nil {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrada")
	}
	if p.UsuariaID != cuentaID {
		return httpx.E(404, "RESOURCE_NOT_FOUND", "no encontrada")
	}
	p.Estado = models.PostulacionRetirada
	p.ActualizadoEn = time.Now()
	if err := r.Postulaciones().Save(c.Context(), p); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"id": p.ID, "estado": p.Estado})
}
