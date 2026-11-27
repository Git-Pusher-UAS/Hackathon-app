// Package caseaccessrequest implementa las solicitudes de acceso a un
// expediente que lleva otra área (§7): crear y listar.
package caseaccessrequest

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	carmodels "backend-hackathon/internal/models/caseaccessrequest"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const (
	modulo      = "expedientes"
	tipoReg     = "solicitud_vinculacion"
	cursorScope = "case_access_requests.list"
	maxPending  = 100 // solicitudes pendientes propias que se revisan para detectar duplicados
)

var (
	ErrLinkInvalid       = errors.New("link token not found")
	ErrLinkExpired       = errors.New("link token expired or used")
	ErrRequestExists     = errors.New("pending request exists")
	ErrCryptoUnavailable = errors.New("cryptographic service unavailable")
)

type Service interface {
	Create(ctx context.Context, actor repository.ActorContext, in carmodels.CreateRequest) (*carmodels.Created, error)
	List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoSolicitudVinculacion, page repository.PageRequest) (*carmodels.Page, error)
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

// MotivoAAD liga el motivo cifrado a su solicitud.
func MotivoAAD(id uuid.UUID) []byte { return []byte(tipoReg + ":motivo:" + id.String()) }

func (s *service) Create(ctx context.Context, actor repository.ActorContext, in carmodels.CreateRequest) (*carmodels.Created, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	token := strings.TrimPrefix(strings.TrimSpace(in.LinkToken), "lt_")
	motivo := strings.TrimSpace(in.Motivo)
	fields := map[string]string{}
	if token == "" {
		fields["linkToken"] = "requerido"
	}
	if n := utf8.RuneCountInString(motivo); n < 10 || n > 500 {
		fields["motivo"] = "longitud_invalida"
	}
	if len(fields) > 0 {
		return nil, httpx.Invalid("solicitud inválida", fields)
	}

	var out carmodels.Created
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		t, err := r.TokensVinculacion().GetByTokenHash(ctx, security.HashOpaqueToken(token))
		if errors.Is(err, repository.ErrNotFound) {
			return ErrLinkInvalid
		}
		if err != nil {
			return err
		}
		if t.SolicitanteID != actor.ActorID { // un token solo sirve a quien lo obtuvo
			return ErrLinkInvalid
		}
		pendiente := models.SolicitudPendiente
		previas, err := r.SolicitudesVinculacion().List(ctx, repository.SolicitudVinculacionFilter{Estado: &pendiente,
			SolicitanteID: &actor.ActorID}, repository.PageRequest{Limit: maxPending})
		if err != nil {
			return err
		}
		for _, p := range previas.Items {
			if p.ExpedienteID == t.ExpedienteID {
				return ErrRequestExists
			}
		}
		now := s.now()
		ok, err := r.TokensVinculacion().Consumir(ctx, t.ID, now)
		if err != nil {
			return err
		}
		if !ok {
			return ErrLinkExpired
		}
		id := uuid.New()
		clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, tipoReg, id, []byte(motivo), MotivoAAD(id))
		if err != nil {
			return err
		}
		if err := r.SolicitudesVinculacion().Create(ctx, &models.SolicitudVinculacion{ID: id, TokenID: t.ID,
			SolicitanteID: actor.ActorID, ExpedienteID: t.ExpedienteID, MotivoCifrado: payload,
			Estado: models.SolicitudPendiente, ClaveID: clave, Version: 1, CreadoEn: now, ActualizadoEn: now}); err != nil {
			return err
		}
		out = carmodels.Created{RequestID: id, Estado: models.SolicitudPendiente}
		return audit.RecordResource(ctx, r, &actor.ActorID, "case_access_request.create", "expediente",
			models.ResultadoExito, &t.ExpedienteID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoSolicitudVinculacion, page repository.PageRequest) (*carmodels.Page, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	st := ""
	if estado != nil {
		st = string(*estado)
	}
	cur := httpx.Cursor{Secret: s.secret, Scope: cursorScope}
	plain, hash, err := cur.Decode(page.Cursor, actor.ActorID, st)
	if err != nil {
		return nil, err
	}
	out := &carmodels.Page{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.SolicitudesVinculacion().List(ctx, repository.SolicitudVinculacionFilter{Estado: estado},
			repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		people := map[uuid.UUID]*models.Personal{}
		deps := map[uuid.UUID]string{}
		out.Items = make([]carmodels.Item, 0, len(rp.Items))
		for _, x := range rp.Items {
			p, ok := people[x.SolicitanteID]
			if !ok {
				if p, err = r.Personal().GetByID(ctx, x.SolicitanteID); err != nil {
					return err
				}
				people[x.SolicitanteID] = p
			}
			depName, ok := deps[p.DepartamentoID]
			if !ok {
				d, err := r.Departamentos().GetByID(ctx, p.DepartamentoID)
				if err != nil {
					return err
				}
				depName = d.Nombre
				deps[p.DepartamentoID] = depName
			}
			e, err := r.Expedientes().GetByID(ctx, x.ExpedienteID)
			if err != nil {
				return err
			}
			motivo, err := s.keyring.Open(ctx, r.ClavesRegistro(), x.ClaveID, x.MotivoCifrado, MotivoAAD(x.ID))
			if err != nil {
				return err
			}
			out.Items = append(out.Items, carmodels.Item{ID: x.ID, ExpedienteID: x.ExpedienteID, ExpedienteAlias: e.Alias,
				Solicitante:  carmodels.Solicitante{ID: p.ID, NombreLaboral: p.NombreLaboral},
				Departamento: depName, DepartamentoID: p.DepartamentoID,
				Motivo: string(motivo), Fecha: x.CreadoEn.UTC(), Estado: x.Estado})
		}
		out.NextCursor = cur.Encode(actor.ActorID, hash, rp.NextCursor)
		return audit.RecordResource(ctx, r, &actor.ActorID, "case_access_request.list", "solicitud_vinculacion",
			models.ResultadoExito, nil, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
