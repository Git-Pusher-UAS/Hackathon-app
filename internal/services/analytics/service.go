package analytics

import (
	"context"
	"fmt"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	anmodels "backend-hackathon/internal/models/analytics"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const metodologia = "Expedientes abiertos en el periodo. Cifras menores a 5 se suprimen."

type SummaryQuery struct {
	Period string
	From   *time.Time
	To     *time.Time
	Type   string
	Status string
}

type HeatmapQuery struct {
	From *time.Time
	To   *time.Time
	Type string
}

type Service interface {
	Summary(ctx context.Context, q SummaryQuery) (*anmodels.Summary, error)
	Heatmap(ctx context.Context, q HeatmapQuery) ([]anmodels.HeatmapCell, error)
}

type service struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func New(uow repository.UnitOfWork) Service { return &service{uow: uow, now: time.Now} }

var zona = func() *time.Location {
	if loc, err := time.LoadLocation("America/Mazatlan"); err == nil {
		return loc
	}
	return time.FixedZone("UTC-7", -7*60*60)
}()

func Corte(now time.Time) time.Time {
	l := now.In(zona)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, zona).UTC()
}

func Rango(period string, corte time.Time) (repository.TimeRange, repository.TimeRange, error) {
	var desde time.Time
	switch period {
	case "", "30d":
		desde = corte.AddDate(0, 0, -30)
	case "6m":
		desde = corte.AddDate(0, -6, 0)
	case "12m":
		desde = corte.AddDate(0, -12, 0)
	default:
		return repository.TimeRange{}, repository.TimeRange{}, httpx.Invalid("period invalido", map[string]string{"period": "valor_invalido"})
	}
	actual := repository.TimeRange{Desde: desde, Hasta: corte}
	anterior := repository.TimeRange{Desde: desde.Add(-corte.Sub(desde)), Hasta: desde}
	return actual, anterior, nil
}

func suprimir(n int) (*int, bool) {
	if n < anmodels.SupresionK {
		return nil, true
	}
	v := n
	return &v, false
}

func serie(cs []repository.Conteo) []anmodels.Serie {
	out := make([]anmodels.Serie, 0, len(cs))
	for _, c := range cs {
		v, sup := suprimir(c.Valor)
		out = append(out, anmodels.Serie{Etiqueta: c.Etiqueta, Valor: v, Suprimido: sup})
	}
	return out
}

func indicador(clave, titulo string, actual, anterior int) anmodels.Indicador {
	v, sup := suprimir(actual)
	ind := anmodels.Indicador{Clave: clave, Titulo: titulo, Valor: v, Suprimido: sup}
	switch {
	case sup:
		ind.Detalle = "Suprimido"
	default:
		if _, supAnt := suprimir(anterior); supAnt {
			ind.Detalle = "Sin comparacion con el periodo anterior"
		} else {
			ind.Detalle = fmt.Sprintf("%+d vs. periodo anterior", actual-anterior)
		}
	}
	return ind
}

func (s *service) filter(q SummaryQuery) (repository.IndicadoresFiltro, repository.IndicadoresFiltro, time.Time, error) {
	corte := Corte(s.now())
	actual, anterior, err := Rango(q.Period, corte)
	if err != nil {
		return repository.IndicadoresFiltro{}, repository.IndicadoresFiltro{}, time.Time{}, err
	}
	if q.From != nil {
		actual.Desde = *q.From
		anterior.Desde = q.From.Add(-actual.Hasta.Sub(*q.From))
		anterior.Hasta = *q.From
	}
	if q.To != nil {
		actual.Hasta = *q.To
		corte = *q.To
	}
	f := repository.IndicadoresFiltro{Rango: actual}
	fa := repository.IndicadoresFiltro{Rango: anterior}
	if q.Type != "" {
		if id, err := uuid.Parse(q.Type); err == nil {
			f.TipoCasoID, fa.TipoCasoID = &id, &id
		}
	}
	if q.Status != "" {
		st := models.EstadoExpediente(q.Status)
		f.Estado, fa.Estado = &st, &st
	}
	return f, fa, corte, nil
}

func (s *service) Summary(ctx context.Context, q SummaryQuery) (*anmodels.Summary, error) {
	f, fa, corte, err := s.filter(q)
	if err != nil {
		return nil, err
	}
	r := s.uow.Repositories().Indicadores()
	abiertos, err := r.Expedientes(ctx, f)
	if err != nil {
		return nil, err
	}
	abiertosAnt, err := r.Expedientes(ctx, fa)
	if err != nil {
		return nil, err
	}
	canal, err := r.Canalizaciones(ctx, f)
	if err != nil {
		return nil, err
	}
	canalAnt, err := r.Canalizaciones(ctx, fa)
	if err != nil {
		return nil, err
	}
	mes, err := r.PorMes(ctx, f)
	if err != nil {
		return nil, err
	}
	origen, err := r.PorOrigen(ctx, f)
	if err != nil {
		return nil, err
	}
	area, err := r.PorArea(ctx, f)
	if err != nil {
		return nil, err
	}
	return &anmodels.Summary{
		Corte:       corte,
		Metodologia: metodologia,
		Indicadores: []anmodels.Indicador{
			indicador("abiertos", "Expedientes abiertos", abiertos, abiertosAnt),
			indicador("canalizaciones", "Canalizaciones", canal, canalAnt),
		},
		PorMes:    serie(mes),
		PorOrigen: serie(origen),
		PorArea:   serie(area),
	}, nil
}

func (s *service) Heatmap(ctx context.Context, q HeatmapQuery) ([]anmodels.HeatmapCell, error) {
	corte := Corte(s.now())
	actual, _, err := Rango("30d", corte)
	if err != nil {
		return nil, err
	}
	if q.From != nil {
		actual.Desde = *q.From
	}
	if q.To != nil {
		actual.Hasta = *q.To
	}
	f := repository.IndicadoresFiltro{Rango: actual}
	if q.Type != "" {
		if id, err := uuid.Parse(q.Type); err == nil {
			f.TipoCasoID = &id
		}
	}
	cs, err := s.uow.Repositories().Indicadores().PorSector(ctx, f)
	if err != nil {
		return nil, err
	}
	var publicables []int
	for _, c := range cs {
		if c.Valor >= anmodels.SupresionK {
			publicables = append(publicables, c.Valor)
		}
	}
	out := make([]anmodels.HeatmapCell, 0, len(cs))
	for _, c := range cs {
		cell := anmodels.HeatmapCell{SectorID: c.Etiqueta}
		if c.Valor < anmodels.SupresionK {
			cell.Suppressed = true
			cell.Count = nil
		} else {
			v := c.Valor
			cell.Count = &v
			menores := 0
			for _, p := range publicables {
				if p < c.Valor {
					menores++
				}
			}
			nivel := 1
			if len(publicables) > 0 {
				nivel = 1 + menores*5/len(publicables)
			}
			if nivel < 1 {
				nivel = 1
			}
			if nivel > 5 {
				nivel = 5
			}
			cell.Nivel = nivel
		}
		out = append(out, cell)
	}
	return out, nil
}
