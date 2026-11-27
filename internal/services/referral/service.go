// Package referral implementa la bandeja de canalizaciones (§8.2). La
// creación vive en casefile (§8.1).
package referral

import (
	"context"
	"errors"
	"strings"
	"time"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	refmodels "backend-hackathon/internal/models/referral"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const cursorScope = "referrals.list"

var ErrCryptoUnavailable = errors.New("cryptographic service unavailable")

type Service interface {
	List(ctx context.Context, actor repository.ActorContext, q refmodels.ListQuery, page repository.PageRequest) (*refmodels.Page, error)
}

type service struct {
	uow     repository.UnitOfWork
	keyring *security.Keyring
	secret  string
	now     func() time.Time
}

func New(uow repository.UnitOfWork, keyring *security.Keyring, secret string) Service {
	return &service{uow: uow, keyring: keyring, secret: secret, now: time.Now}
}

// AAD liga un campo cifrado a su registro.
func AAD(tipo, campo string, id uuid.UUID) []byte {
	return []byte(tipo + ":" + campo + ":" + id.String())
}

func estadosKey(xs []models.EstadoCanalizacion) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = string(x)
	}
	return strings.Join(parts, ",")
}

// nombres cachea los nombres de departamentos e instituciones de una página.
type nombres struct {
	r     repository.Repositories
	deps  map[uuid.UUID]string
	insts map[uuid.UUID]string
}

func newNombres(r repository.Repositories) *nombres {
	return &nombres{r: r, deps: map[uuid.UUID]string{}, insts: map[uuid.UUID]string{}}
}

func (n *nombres) dep(ctx context.Context, id uuid.UUID) (string, error) {
	if v, ok := n.deps[id]; ok {
		return v, nil
	}
	d, err := n.r.Departamentos().GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	n.deps[id] = d.Nombre
	return d.Nombre, nil
}

func (n *nombres) inst(ctx context.Context, id uuid.UUID) (string, error) {
	if v, ok := n.insts[id]; ok {
		return v, nil
	}
	i, err := n.r.Instituciones().GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	n.insts[id] = i.Nombre
	return i.Nombre, nil
}

// view arma la canalización con nombres, el resumen descifrado y el motivo del
// último rechazo o cancelación.
func (s *service) view(ctx context.Context, n *nombres, c *models.CanalizacionInterna) (refmodels.View, error) {
	r := n.r
	origen, err := n.dep(ctx, c.OrigenID)
	if err != nil {
		return refmodels.View{}, err
	}
	destino := refmodels.Destino{DepartamentoID: c.DestinoID, InstitucionID: c.InstitucionID}
	switch {
	case c.DestinoID != nil:
		destino.Nombre, err = n.dep(ctx, *c.DestinoID)
	case c.InstitucionID != nil:
		destino.Nombre, err = n.inst(ctx, *c.InstitucionID)
	}
	if err != nil {
		return refmodels.View{}, err
	}
	e, err := r.Expedientes().GetByID(ctx, c.ExpedienteID)
	if err != nil {
		return refmodels.View{}, err
	}
	resumen, err := s.keyring.Open(ctx, r.ClavesRegistro(), c.ClaveID, c.ResumenCifrado, AAD("canalizacion", "resumen", c.ID))
	if err != nil {
		return refmodels.View{}, err
	}
	out := refmodels.View{ID: c.ID, ExpedienteID: c.ExpedienteID, Alias: e.Alias, Tipo: c.Tipo,
		Origen:  refmodels.Origen{DepartamentoID: c.OrigenID, Nombre: origen},
		Destino: destino, Resumen: string(resumen), Estado: c.Estado,
		CreatedAt: c.CreadoEn.UTC(), VersionTag: httpx.EncodeVersionTag(s.secret, c.Version)}
	if c.Estado == models.CanalizacionRechazada || c.Estado == models.CanalizacionCancelada {
		hist, err := r.HistorialCanalizaciones().ListByCanalizacion(ctx, c.ID)
		if err != nil {
			return refmodels.View{}, err
		}
		for i := len(hist) - 1; i >= 0; i-- {
			if len(hist[i].MotivoCifrado) == 0 {
				continue
			}
			m, err := s.keyring.Open(ctx, r.ClavesRegistro(), hist[i].ClaveID, hist[i].MotivoCifrado,
				AAD("historial_canalizacion", "motivo", hist[i].ID))
			if err != nil {
				return refmodels.View{}, err
			}
			motivo := string(m)
			out.Motivo = &motivo
			break
		}
	}
	return out, nil
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, q refmodels.ListQuery, page repository.PageRequest) (*refmodels.Page, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	if q.Direction == "" {
		q.Direction = refmodels.DirectionIncoming
	}
	if q.Direction != refmodels.DirectionIncoming && q.Direction != refmodels.DirectionOutgoing {
		return nil, httpx.Invalid("direction inválido", map[string]string{"direction": "valor_invalido"})
	}
	rec := ""
	if q.RecordID != nil {
		rec = q.RecordID.String()
	}
	cur := httpx.Cursor{Secret: s.secret, Scope: cursorScope}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, q.Direction, estadosKey(q.Estados), rec)
	if err != nil {
		return nil, err
	}
	out := &refmodels.Page{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Personal().GetByID(ctx, actor.ActorID)
		if err != nil {
			return err
		}
		f := repository.CanalizacionFilter{Estados: q.Estados, ExpedienteID: q.RecordID}
		if q.Direction == refmodels.DirectionIncoming {
			f.DestinoIDs = []uuid.UUID{p.DepartamentoID}
		} else {
			f.OrigenIDs = []uuid.UUID{p.DepartamentoID}
		}
		rp, err := r.Canalizaciones().List(ctx, f, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		n := newNombres(r)
		out.Items = make([]refmodels.View, 0, len(rp.Items))
		for i := range rp.Items {
			v, err := s.view(ctx, n, &rp.Items[i])
			if err != nil {
				return err
			}
			out.Items = append(out.Items, v)
		}
		out.NextCursor = cur.Encode(actor.ActorID, hash, rp.NextCursor)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
