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
	List(ctx context.Context, actor repository.ActorContext, q refmodels.ListQuery, page repository.PageRequest) ([]refmodels.View, error)
	Transitioner
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
	return refmodels.View{
		ID:           c.ID,
		ExpedienteID: c.ExpedienteID,
		Alias:        e.Alias,
		Tipo:         c.Tipo,
		Origen:       refmodels.Origen{DepartamentoID: c.OrigenID, Nombre: origen},
		Destino:      destino,
		Resumen:      string(resumen),
		Estado:       c.Estado,
		Fecha:        c.CreadoEn.UTC(),
		VersionTag:   httpx.EncodeVersionTag(s.secret, c.Version),
	}, nil
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, q refmodels.ListQuery, page repository.PageRequest) ([]refmodels.View, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	if q.Direction == "" {
		q.Direction = refmodels.DirectionIncoming
	}
	if q.Direction != refmodels.DirectionIncoming && q.Direction != refmodels.DirectionOutgoing {
		return nil, httpx.Invalid("direction invalido", map[string]string{"direction": "valor_invalido"})
	}
	rec := ""
	if q.RecordID != nil {
		rec = q.RecordID.String()
	} else if q.RecordRef != "" {
		rec = q.RecordRef
	}
	filterHash := httpx.FilterHash(q.Direction, estadosKey(q.Estados), rec)
	plain, err := httpx.DecodeCursor(s.secret, page.Cursor, actor.ActorID, cursorScope, filterHash)
	if err != nil {
		return nil, httpx.E(422, "VALIDATION_ERROR", "cursor invalido")
	}
	out := []refmodels.View{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		p, err := r.Personal().GetByID(ctx, actor.ActorID)
		if err != nil {
			return err
		}
		f := repository.CanalizacionFilter{Estados: q.Estados, ExpedienteID: q.RecordID}
		if f.ExpedienteID == nil && q.RecordRef != "" {
			rpExp, err := r.Expedientes().List(ctx, repository.ExpedienteFilter{Alias: q.RecordRef}, repository.PageRequest{Limit: 10})
			if err != nil {
				return err
			}
			for i := range rpExp.Items {
				if strings.EqualFold(strings.TrimSpace(rpExp.Items[i].Alias), strings.TrimSpace(q.RecordRef)) {
					id := rpExp.Items[i].ID
					f.ExpedienteID = &id
					break
				}
			}
			if f.ExpedienteID == nil {
				out = []refmodels.View{}
				return nil
			}
		}
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
		out = make([]refmodels.View, 0, len(rp.Items))
		for i := range rp.Items {
			v, err := s.view(ctx, n, &rp.Items[i])
			if err != nil {
				return err
			}
			out = append(out, v)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
