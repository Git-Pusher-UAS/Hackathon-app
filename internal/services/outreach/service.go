// Package outreach implementa las visitas de Capacitación a sectores (§11.6).
package outreach

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	omodels "backend-hackathon/internal/models/outreach"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const (
	cursorScope = "outreach.list"
	fechaFmt    = "2006-01-02"
)

type Service interface {
	List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoVisita, page repository.PageRequest) (*omodels.Page, error)
	Create(ctx context.Context, actor repository.ActorContext, in omodels.CreateRequest) (*omodels.Visita, error)
}

type service struct {
	uow    repository.UnitOfWork
	secret string
}

func New(uow repository.UnitOfWork, secret string) Service { return &service{uow: uow, secret: secret} }

type sectores struct {
	r repository.Repositories
	m map[uuid.UUID]string
}

func (s *sectores) nombre(ctx context.Context, id uuid.UUID) (string, error) {
	if n, ok := s.m[id]; ok {
		return n, nil
	}
	sec, err := s.r.Sectores().GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	s.m[id] = sec.Nombre
	return sec.Nombre, nil
}

func view(v *models.Visita, sector string) omodels.Visita {
	return omodels.Visita{ID: v.ID, SectorID: v.SectorID, Sector: sector, Fecha: v.Fecha.Format(fechaFmt),
		Tema: v.Tema, Responsable: v.Responsable, Estado: v.Estado}
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoVisita, page repository.PageRequest) (*omodels.Page, error) {
	st := ""
	if estado != nil {
		st = string(*estado)
	}
	cur := httpx.Cursor{Secret: s.secret, Scope: cursorScope}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, st)
	if err != nil {
		return nil, err
	}
	r := s.uow.Repositories()
	rp, err := r.Visitas().List(ctx, estado, repository.PageRequest{Limit: page.Limit, Cursor: plain})
	if err != nil {
		return nil, err
	}
	sec := &sectores{r: r, m: map[uuid.UUID]string{}}
	out := &omodels.Page{Items: make([]omodels.Visita, 0, len(rp.Items))}
	for i := range rp.Items {
		n, err := sec.nombre(ctx, rp.Items[i].SectorID)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, view(&rp.Items[i], n))
	}
	out.NextCursor = cur.Encode(actor.ActorID, hash, rp.NextCursor)
	return out, nil
}

func (s *service) Create(ctx context.Context, actor repository.ActorContext, in omodels.CreateRequest) (*omodels.Visita, error) {
	fields := map[string]string{}
	sectorID, err := uuid.Parse(strings.TrimSpace(in.SectorID))
	if err != nil {
		fields["sectorId"] = "formato_invalido"
	}
	fecha, err := time.Parse(fechaFmt, in.Fecha)
	if err != nil {
		fields["fecha"] = "formato_invalido"
	}
	tema := strings.TrimSpace(in.Tema)
	if n := utf8.RuneCountInString(tema); n < 1 || n > 250 {
		fields["tema"] = "longitud_invalida"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("visita inválida", fields)
	}
	var out omodels.Visita
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		sec, err := r.Sectores().GetByID(ctx, sectorID)
		if errors.Is(err, repository.ErrNotFound) {
			return httpx.Invalid("sector inexistente", map[string]string{"sectorId": "no_existe"})
		}
		if err != nil {
			return err
		}
		responsable := "Capacitación"
		if p, err := r.Personal().GetByID(ctx, actor.ActorID); err == nil {
			if d, err := r.Departamentos().GetByID(ctx, p.DepartamentoID); err == nil {
				responsable = d.Nombre
			}
		}
		v := &models.Visita{ID: uuid.New(), SectorID: sectorID, Fecha: fecha, Tema: tema, Responsable: responsable,
			Estado: models.VisitaPlaneada, CreadoPor: actor.ActorID}
		if err := r.Visitas().Create(ctx, v); err != nil {
			return err
		}
		out = view(v, sec.Nombre)
		return audit.RecordResource(ctx, r, &actor.ActorID, "outreach.create", "visita",
			models.ResultadoExito, &v.ID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
