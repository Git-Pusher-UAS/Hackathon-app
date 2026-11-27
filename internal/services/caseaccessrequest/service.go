// Package caseaccessrequest implementa las solicitudes de acceso a expedientes.
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
	maxPending  = 100
)

var (
	ErrLinkInvalid       = errors.New("link token not found")
	ErrLinkExpired       = errors.New("link token expired or used")
	ErrRequestExists     = errors.New("pending request exists")
	ErrNotFound          = errors.New("request not found")
	ErrNotPending        = errors.New("request not pending")
	ErrCryptoUnavailable = errors.New("cryptographic service unavailable")
)

type Service interface {
	Create(ctx context.Context, actor repository.ActorContext, in carmodels.CreateRequest) (*carmodels.Created, error)
	List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoSolicitudVinculacion, page repository.PageRequest) ([]carmodels.Item, error)
	Reject(ctx context.Context, actor repository.ActorContext, id uuid.UUID, motivo string) error
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
		return nil, httpx.Invalid("solicitud invalida", fields)
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
		if t.SolicitanteID != actor.ActorID {
			return ErrLinkInvalid
		}
		pendiente := models.SolicitudPendiente
		previas, err := r.SolicitudesVinculacion().List(ctx, repository.SolicitudVinculacionFilter{Estado: &pendiente, SolicitanteID: &actor.ActorID}, repository.PageRequest{Limit: maxPending})
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
		if err := r.SolicitudesVinculacion().Create(ctx, &models.SolicitudVinculacion{ID: id, TokenID: t.ID, SolicitanteID: actor.ActorID, ExpedienteID: t.ExpedienteID, MotivoCifrado: payload, Estado: models.SolicitudPendiente, ClaveID: clave, Version: 1, CreadoEn: now, ActualizadoEn: now}); err != nil {
			return err
		}
		out = carmodels.Created{RequestID: id, Estado: models.SolicitudPendiente}
		return audit.RecordResource(ctx, r, &actor.ActorID, "case_access_request.create", "expediente", models.ResultadoExito, &t.ExpedienteID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) Reject(ctx context.Context, actor repository.ActorContext, id uuid.UUID, motivo string) error {
	motivo = strings.TrimSpace(motivo)
	if n := utf8.RuneCountInString(motivo); n < 10 || n > 500 {
		return httpx.Invalid("motivo invalido", map[string]string{"motivo": "longitud_invalida"})
	}
	return s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		sol, err := r.SolicitudesVinculacion().GetByID(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if sol.Estado != models.SolicitudPendiente {
			return ErrNotPending
		}
		if err := r.SolicitudesVinculacion().Resolver(ctx, id, models.SolicitudRechazada, actor.ActorID, s.now(), nil, sol.Version); err != nil {
			if errors.Is(err, repository.ErrVersionMismatch) {
				return ErrNotPending
			}
			return err
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "case_access_request.reject", "solicitud_vinculacion", models.ResultadoExito, &id, actor.RequestID, actor.IPOrigen)
	})
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, estado *models.EstadoSolicitudVinculacion, page repository.PageRequest) ([]carmodels.Item, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	st := ""
	if estado != nil {
		st = string(*estado)
	}
	filterHash := httpx.FilterHash(st)
	plain, err := httpx.DecodeCursor(s.secret, page.Cursor, actor.ActorID, cursorScope, filterHash)
	if err != nil {
		return nil, httpx.E(422, "VALIDATION_ERROR", "cursor invalido")
	}
	out := []carmodels.Item{}
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		rp, err := r.SolicitudesVinculacion().List(ctx, repository.SolicitudVinculacionFilter{Estado: estado}, repository.PageRequest{Limit: page.Limit, Cursor: plain})
		if err != nil {
			return err
		}
		people := map[uuid.UUID]*models.Personal{}
		deps := map[uuid.UUID]string{}
		out = make([]carmodels.Item, 0, len(rp.Items))
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
			motivo, err := s.keyring.Open(ctx, r.ClavesRegistro(), x.ClaveID, x.MotivoCifrado, MotivoAAD(x.ID))
			if err != nil {
				return err
			}
			out = append(out, carmodels.Item{
				ID:           x.ID,
				ExpedienteID: x.ExpedienteID,
				Solicitante:  carmodels.Solicitante{ID: p.ID, NombreLaboral: p.NombreLaboral},
				Departamento: depName,
				Motivo:       string(motivo),
				Fecha:        x.CreadoEn.UTC(),
				Estado:       x.Estado,
			})
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "case_access_request.list", "solicitud_vinculacion", models.ResultadoExito, nil, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
