// Package analytics implementa los agregados del tablero (API-037/038) sobre
// las vistas analitica.v_resumen_mvp y analitica.v_mapa_mvp, que ya aplican
// la supresión k = 5.
package analytics

import (
	"context"
	"time"

	anmodels "backend-hackathon/internal/models/analytics"
	repository "backend-hackathon/internal/repository/repositories"
)

type Service interface {
	Summary(ctx context.Context, f repository.ResumenFiltro) (*anmodels.Summary, error)
	Heatmap(ctx context.Context, f repository.MapaFiltro) (*anmodels.Heatmap, error)
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

func (s *service) Summary(ctx context.Context, f repository.ResumenFiltro) (*anmodels.Summary, error) {
	rows, err := s.uow.Repositories().Resumen().Consultar(ctx, f)
	if err != nil {
		return nil, err
	}
	out := &anmodels.Summary{GeneradoEn: s.now().UTC(), SupresionK: anmodels.SupresionK,
		Items: make([]anmodels.SummaryRow, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, anmodels.SummaryRow{SectorID: r.SectorID, TipoCasoID: r.TipoCasoID,
			Periodo: r.Periodo.UTC(), Personas: r.Personas, Episodios: r.Episodios,
			Suprimido: r.Personas == nil || r.Episodios == nil})
	}
	return out, nil
}

func (s *service) Heatmap(ctx context.Context, f repository.MapaFiltro) (*anmodels.Heatmap, error) {
	rows, err := s.uow.Repositories().Mapa().Consultar(ctx, f)
	if err != nil {
		return nil, err
	}
	out := &anmodels.Heatmap{GeneradoEn: s.now().UTC(), SupresionK: anmodels.SupresionK,
		Items: make([]anmodels.HeatmapCell, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, anmodels.HeatmapCell{SectorID: r.SectorID, TipoCasoID: r.TipoCasoID,
			Periodo: r.Periodo.UTC(), Nivel: r.Nivel, Conteo: r.Conteo, Suprimido: r.Conteo == nil})
	}
	return out, nil
}
