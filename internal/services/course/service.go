// Package course implementa cursos, charlas y conferencias de Capacitación (§11).
package course

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	cmodels "backend-hackathon/internal/models/course"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const (
	cursorScope = "courses.list"
	fechaFmt    = "2006-01-02"
)

var (
	ErrNotFound = errors.New("course not found")
	horaRe      = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	// zona es la hora local de Los Mochis, para decidir qué fecha ya pasó.
	zona = func() *time.Location {
		if loc, err := time.LoadLocation("America/Mazatlan"); err == nil {
			return loc
		}
		return time.FixedZone("UTC-7", -7*60*60)
	}()
)

// ListQuery son los filtros de GET /courses.
type ListQuery struct {
	Tipo         *models.TipoCurso
	Desde, Hasta *time.Time
}

type Service interface {
	List(ctx context.Context, actor repository.ActorContext, q ListQuery, page repository.PageRequest) (*cmodels.Page, error)
	Get(ctx context.Context, id uuid.UUID) (*cmodels.Curso, error)
	Create(ctx context.Context, actor repository.ActorContext, in cmodels.CreateRequest) (*cmodels.Curso, error)
	Attendance(ctx context.Context, id uuid.UUID) ([]cmodels.Asistencia, error)
	SetAttendance(ctx context.Context, actor repository.ActorContext, id uuid.UUID, marcas []cmodels.MarcaAsistencia) error
}

type service struct {
	uow    repository.UnitOfWork
	secret string
	now    func() time.Time
}

func New(uow repository.UnitOfWork, secret string) Service {
	return &service{uow: uow, secret: secret, now: time.Now}
}

// Hoy es la fecha local de Los Mochis.
func Hoy(now time.Time) time.Time {
	l := now.In(zona)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

func view(c *models.Curso, inscritas int) cmodels.Curso {
	return cmodels.Curso{ID: c.ID, Tipo: c.Tipo, Titulo: c.Titulo, Descripcion: c.Descripcion,
		Fecha: c.Fecha.Format(fechaFmt), Hora: c.Hora, Lugar: c.Lugar, Cupo: c.Cupo, Inscritas: inscritas}
}

// ValidTipo indica si el tipo de curso existe.
func ValidTipo(t models.TipoCurso) bool {
	return t == models.CursoCurso || t == models.CursoCharla || t == models.CursoConferencia
}

func fmtDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(fechaFmt)
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, q ListQuery, page repository.PageRequest) (*cmodels.Page, error) {
	tipo := ""
	if q.Tipo != nil {
		tipo = string(*q.Tipo)
	}
	cur := httpx.Cursor{Secret: s.secret, Scope: cursorScope}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, tipo, fmtDate(q.Desde), fmtDate(q.Hasta))
	if err != nil {
		return nil, err
	}
	r := s.uow.Repositories()
	rp, err := r.Cursos().List(ctx, repository.CursoFilter{Tipo: q.Tipo, Desde: q.Desde, Hasta: q.Hasta},
		repository.PageRequest{Limit: page.Limit, Cursor: plain})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rp.Items))
	for _, c := range rp.Items {
		ids = append(ids, c.ID)
	}
	counts, err := r.Cursos().Inscritas(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &cmodels.Page{Items: make([]cmodels.Curso, 0, len(rp.Items))}
	for i := range rp.Items {
		out.Items = append(out.Items, view(&rp.Items[i], counts[rp.Items[i].ID]))
	}
	out.NextCursor = cur.Encode(actor.ActorID, hash, rp.NextCursor)
	return out, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*cmodels.Curso, error) {
	r := s.uow.Repositories()
	c, err := r.Cursos().GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	counts, err := r.Cursos().Inscritas(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	v := view(c, counts[id])
	return &v, nil
}

func longitud(s string, min, max int) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	return n >= min && n <= max
}

func (s *service) Create(ctx context.Context, actor repository.ActorContext, in cmodels.CreateRequest) (*cmodels.Curso, error) {
	fields := map[string]string{}
	if !ValidTipo(in.Tipo) {
		fields["tipo"] = "valor_invalido"
	}
	if !longitud(in.Titulo, 1, 150) {
		fields["titulo"] = "longitud_invalida"
	}
	if !longitud(in.Descripcion, 1, 2000) {
		fields["descripcion"] = "longitud_invalida"
	}
	if !longitud(in.Lugar, 1, 250) {
		fields["lugar"] = "longitud_invalida"
	}
	fecha, err := time.Parse(fechaFmt, in.Fecha)
	switch {
	case err != nil:
		fields["fecha"] = "formato_invalido"
	case fecha.Before(Hoy(s.now())):
		fields["fecha"] = "fecha_pasada"
	}
	if !horaRe.MatchString(in.Hora) {
		fields["hora"] = "formato_invalido"
	}
	if in.Cupo < 1 {
		fields["cupo"] = "minimo_1"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("curso inválido", fields)
	}
	c := &models.Curso{ID: uuid.New(), Tipo: in.Tipo, Titulo: strings.TrimSpace(in.Titulo),
		Descripcion: strings.TrimSpace(in.Descripcion), Fecha: fecha, Hora: in.Hora,
		Lugar: strings.TrimSpace(in.Lugar), Cupo: in.Cupo, CreadoPor: actor.ActorID}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		if err := r.Cursos().Create(ctx, c); err != nil {
			return err
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "course.create", "curso",
			models.ResultadoExito, &c.ID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	v := view(c, 0)
	return &v, nil
}

func (s *service) Attendance(ctx context.Context, id uuid.UUID) ([]cmodels.Asistencia, error) {
	r := s.uow.Repositories()
	if _, err := r.Cursos().GetByID(ctx, id); errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	ins, err := r.Inscripciones().ListByCurso(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]cmodels.Asistencia, 0, len(ins))
	for _, x := range ins {
		out = append(out, cmodels.Asistencia{EnrollmentID: x.ID, Alias: x.Alias, Presente: x.Presente})
	}
	return out, nil
}

// SetAttendance marca la asistencia en una sola transacción: si alguna
// inscripción no es del curso, no se cambia nada.
func (s *service) SetAttendance(ctx context.Context, actor repository.ActorContext, id uuid.UUID, marcas []cmodels.MarcaAsistencia) error {
	if len(marcas) == 0 {
		return httpx.Invalid("la lista está vacía", map[string]string{"body": "requerido"})
	}
	return s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		if _, err := r.Cursos().GetByID(ctx, id); errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		for i, m := range marcas {
			if err := r.Inscripciones().SetPresente(ctx, id, m.EnrollmentID, m.Presente); err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return httpx.Invalid("inscripción ajena al curso", map[string]string{"[" + strconv.Itoa(i) + "].enrollmentId": "no_pertenece"})
				}
				return err
			}
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "course.attendance", "curso",
			models.ResultadoExito, &id, actor.RequestID, actor.IPOrigen)
	})
}
