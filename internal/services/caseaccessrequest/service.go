// Package caseaccessrequest implementa las solicitudes de acceso a un
// expediente existente: POST (API-018) y la bandeja GET (API-019).
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
	campoMotivo = "motivo"
	cursorScope = "case_access_requests.list"
)

var (
	ErrNotFound          = errors.New("link token not found")
	ErrGone              = errors.New("link token expired or used")
	ErrCryptoUnavailable = errors.New("cryptographic service unavailable")
)

// ValidationError es un cuerpo o filtro inválido.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

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

func aad(id uuid.UUID) []byte { return []byte(tipoReg + ":" + campoMotivo + ":" + id.String()) }

func (s *service) Create(ctx context.Context, actor repository.ActorContext, in carmodels.CreateRequest) (*carmodels.Created, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	token := strings.TrimSpace(in.LinkToken)
	if token == "" {
		return nil, &ValidationError{Msg: "linkToken es obligatorio"}
	}
	motivo := strings.TrimSpace(in.Motivo)
	if n := utf8.RuneCountInString(motivo); n < 10 || n > 500 {
		return nil, &ValidationError{Msg: "motivo debe tener entre 10 y 500 caracteres"}
	}
	var out carmodels.Created
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		t, err := r.TokensVinculacion().GetByTokenHash(ctx, security.HashOpaqueToken(token))
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// Un token solo sirve a quien lo obtuvo.
		if t.SolicitanteID != actor.ActorID {
			return ErrNotFound
		}
		now := s.now()
		ok, err := r.TokensVinculacion().Consumir(ctx, t.ID, now)
		if err != nil {
			return err
		}
		if !ok {
			return ErrGone
		}
		id := uuid.New()
		clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, tipoReg, id, []byte(motivo), aad(id))
		if err != nil {
			return err
		}
		if err := r.SolicitudesVinculacion().Create(ctx, &models.SolicitudVinculacion{ID: id, TokenID: t.ID,
			SolicitanteID: actor.ActorID, ExpedienteID: t.ExpedienteID, MotivoCifrado: payload,
			Estado: models.SolicitudPendiente, ClaveID: clave, Version: 1, CreadoEn: now, ActualizadoEn: now}); err != nil {
			return err
		}
		out = carmodels.Created{ID: id, Estado: models.SolicitudPendiente}
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
			motivo, err := s.keyring.Open(ctx, r.ClavesRegistro(), x.ClaveID, x.MotivoCifrado, aad(x.ID))
			if err != nil {
				return err
			}
			out.Items = append(out.Items, carmodels.Item{ID: x.ID, ExpedienteID: x.ExpedienteID,
				Solicitante:  carmodels.Ref{ID: p.ID, Nombre: p.NombreLaboral},
				Departamento: carmodels.Ref{ID: p.DepartamentoID, Nombre: depName},
				Motivo:       string(motivo), Estado: x.Estado, CreatedAt: x.CreadoEn.UTC(), ResueltaEn: x.ResueltaEn})
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
