package outreach

import (
	"context"
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

const fechaFmt = "2006-01-02"

type Service interface {
	List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoVisita, page repository.PageRequest) ([]omodels.Visita, error)
	Create(ctx context.Context, actor repository.ActorContext, in omodels.CreateRequest) (*omodels.Visita, error)
}

type service struct{ uow repository.UnitOfWork }

func New(uow repository.UnitOfWork) Service { return &service{uow: uow} }

func view(v *models.Visita, sector string) omodels.Visita {
	return omodels.Visita{ID: v.ID, Sector: sector, Fecha: v.Fecha.Format(fechaFmt), Tema: v.Tema, Responsable: v.Responsable, Estado: v.Estado}
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoVisita, page repository.PageRequest) ([]omodels.Visita, error) {
	out := []omodels.Visita{}
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.Visitas().List(ctx, estado, page)
		if err != nil {
			return err
		}
		sectors := map[uuid.UUID]string{}
		out = make([]omodels.Visita, 0, len(rp.Items))
		for i := range rp.Items {
			name, ok := sectors[rp.Items[i].SectorID]
			if !ok {
				sec, err := r.Sectores().GetByID(ctx, rp.Items[i].SectorID)
				if err != nil {
					return err
				}
				name = sec.Nombre
				sectors[rp.Items[i].SectorID] = name
			}
			out = append(out, view(&rp.Items[i], name))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) Create(ctx context.Context, actor repository.ActorContext, in omodels.CreateRequest) (*omodels.Visita, error) {
	fields := map[string]string{}
	fecha, err := time.Parse(fechaFmt, strings.TrimSpace(in.Fecha))
	if err != nil {
		fields["fecha"] = "formato_invalido"
	}
	tema := strings.TrimSpace(in.Tema)
	if n := utf8.RuneCountInString(tema); n < 1 || n > 250 {
		fields["tema"] = "longitud_invalida"
	}
	sectorName := strings.TrimSpace(in.Sector)
	if sectorName == "" {
		fields["sector"] = "requerido"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("visita invalida", fields)
	}
	var out omodels.Visita
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		sectors, err := r.Sectores().List(ctx, true)
		if err != nil {
			return err
		}
		var sectorID uuid.UUID
		found := false
		for _, sec := range sectors {
			if strings.EqualFold(sec.Nombre, sectorName) || sec.Codigo == sectorName || sec.ID.String() == sectorName {
				sectorID, sectorName, found = sec.ID, sec.Nombre, true
				break
			}
		}
		if !found {
			return httpx.Invalid("sector inexistente", map[string]string{"sector": "no_existe"})
		}
		responsable := "Capacitacion"
		if p, err := r.Personal().GetByID(ctx, actor.ActorID); err == nil {
			if d, err := r.Departamentos().GetByID(ctx, p.DepartamentoID); err == nil {
				responsable = d.Nombre
			}
		}
		v := &models.Visita{ID: uuid.New(), SectorID: sectorID, Fecha: fecha, Tema: tema, Responsable: responsable, Estado: models.VisitaPlaneada, CreadoPor: actor.ActorID}
		if err := r.Visitas().Create(ctx, v); err != nil {
			return err
		}
		out = view(v, sectorName)
		return audit.RecordResource(ctx, r, &actor.ActorID, "outreach.create", "visita", models.ResultadoExito, &v.ID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
