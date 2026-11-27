// Package referral implementa la bandeja de canalizaciones internas (API-032)
// y sus transiciones con control de versión (API-034). La creación vive en
// casefile (API-033).
package referral

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	refmodels "backend-hackathon/internal/models/referral"
	repository "backend-hackathon/internal/repository/repositories"
	"backend-hackathon/internal/security"

	"github.com/google/uuid"
)

const (
	modulo      = "expedientes"
	cursorScope = "referrals.list"
	// Motivo que queda cifrado en la asignación creada al aceptar.
	motivoAsignacion = "Asignación por canalización interna aceptada"
)

var (
	ErrNotFound             = errors.New("referral not found")
	ErrInvalidTransition    = errors.New("invalid referral transition")
	ErrCaseClosed           = errors.New("case is closed")
	ErrPreconditionRequired = errors.New("precondition required")
	ErrVersionMismatch      = errors.New("version mismatch")
	ErrCryptoUnavailable    = errors.New("cryptographic service unavailable")
)

// ValidationError es un cuerpo o filtro inválido.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

type Service interface {
	List(ctx context.Context, actor repository.ActorContext, q refmodels.ListQuery, page repository.PageRequest) (*refmodels.Page, error)
	Transition(ctx context.Context, actor repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error)
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

func aad(tipo, campo string, id uuid.UUID) []byte {
	return []byte(tipo + ":" + campo + ":" + id.String())
}

func estadosKey(xs []models.EstadoCanalizacion) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = string(x)
	}
	return strings.Join(parts, ",")
}

// view arma la vista con nombres de departamento y el resumen descifrado.
func (s *service) view(ctx context.Context, r repository.Repositories, c *models.CanalizacionInterna, deps map[uuid.UUID]string) (refmodels.View, error) {
	name := func(id uuid.UUID) (string, error) {
		if n, ok := deps[id]; ok {
			return n, nil
		}
		d, err := r.Departamentos().GetByID(ctx, id)
		if err != nil {
			return "", err
		}
		deps[id] = d.Nombre
		return d.Nombre, nil
	}
	origen, err := name(c.OrigenID)
	if err != nil {
		return refmodels.View{}, err
	}
	destino, err := name(c.DestinoID)
	if err != nil {
		return refmodels.View{}, err
	}
	e, err := r.Expedientes().GetByID(ctx, c.ExpedienteID)
	if err != nil {
		return refmodels.View{}, err
	}
	resumen, err := s.keyring.Open(ctx, r.ClavesRegistro(), c.ClaveID, c.ResumenCifrado, aad("canalizacion", "resumen", c.ID))
	if err != nil {
		return refmodels.View{}, err
	}
	return refmodels.View{ID: c.ID, ExpedienteID: c.ExpedienteID, Alias: e.Alias,
		Origen: refmodels.Ref{ID: c.OrigenID, Nombre: origen}, Destino: refmodels.Ref{ID: c.DestinoID, Nombre: destino},
		Resumen: string(resumen), Estado: c.Estado, VersionTag: httpx.EncodeVersionTag(s.secret, c.Version),
		CreatedAt: c.CreadoEn.UTC(), UpdatedAt: c.ActualizadoEn.UTC()}, nil
}

func (s *service) List(ctx context.Context, actor repository.ActorContext, q refmodels.ListQuery, page repository.PageRequest) (*refmodels.Page, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	if q.Direction == "" {
		q.Direction = refmodels.DirectionIncoming
	}
	if q.Direction != refmodels.DirectionIncoming && q.Direction != refmodels.DirectionOutgoing {
		return nil, &ValidationError{Msg: "direction debe ser incoming u outgoing"}
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
		deps := map[uuid.UUID]string{}
		out.Items = make([]refmodels.View, 0, len(rp.Items))
		for i := range rp.Items {
			v, err := s.view(ctx, r, &rp.Items[i], deps)
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

// transition describe qué lado puede aplicar cada acción y desde qué estado.
type transition struct {
	from        models.EstadoCanalizacion
	to          models.EstadoCanalizacion
	byDestino   bool
	needsMotivo bool
}

var transitions = map[string]transition{
	refmodels.ActionAccept:   {models.CanalizacionPendiente, models.CanalizacionAceptada, true, false},
	refmodels.ActionReject:   {models.CanalizacionPendiente, models.CanalizacionRechazada, true, true},
	refmodels.ActionComplete: {models.CanalizacionAceptada, models.CanalizacionCompletada, true, false},
	refmodels.ActionCancel:   {models.CanalizacionPendiente, models.CanalizacionCancelada, false, true},
}

func (s *service) Transition(ctx context.Context, actor repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	t, ok := transitions[in.Action]
	if !ok {
		return nil, &ValidationError{Msg: "action debe ser accept, reject, complete o cancel"}
	}
	motivo := ""
	if in.Motivo != nil {
		motivo = strings.TrimSpace(*in.Motivo)
	}
	if n := utf8.RuneCountInString(motivo); (t.needsMotivo && n < 10) || n > 500 {
		return nil, &ValidationError{Msg: "motivo debe tener entre 10 y 500 caracteres"}
	}
	tag, ok := httpx.ParseIfMatch(ifMatch)
	if !ok {
		return nil, ErrPreconditionRequired
	}
	expected, err := httpx.DecodeVersionTag(s.secret, tag)
	if err != nil {
		return nil, ErrVersionMismatch
	}

	var out refmodels.View
	err = s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
		c, err := r.Canalizaciones().GetByIDForUpdate(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		p, err := r.Personal().GetByID(ctx, actor.ActorID)
		if err != nil {
			return err
		}
		// Quien no está en ninguno de los dos lados no ve la canalización.
		if p.DepartamentoID != c.DestinoID && p.DepartamentoID != c.OrigenID {
			return ErrNotFound
		}
		if c.Version != expected {
			return ErrVersionMismatch
		}
		side := c.OrigenID
		if t.byDestino {
			side = c.DestinoID
		}
		if p.DepartamentoID != side || c.Estado != t.from {
			return ErrInvalidTransition
		}

		now := s.now()
		if in.Action == refmodels.ActionAccept {
			e, err := r.Expedientes().GetByID(ctx, c.ExpedienteID)
			if err != nil {
				return err
			}
			if e.Estado == models.ExpedienteCerrado {
				return ErrCaseClosed
			}
			asig, err := s.assign(ctx, r, c.ExpedienteID, actor.ActorID, c.DestinoID, now)
			if err != nil {
				return err
			}
			c.AsignacionDestinoID = &asig
		}

		anterior := c.Estado
		c.Estado = t.to
		if err := r.Canalizaciones().Update(ctx, c, expected); err != nil {
			if errors.Is(err, repository.ErrVersionMismatch) {
				return ErrVersionMismatch
			}
			return err
		}
		c.Version = expected + 1
		c.ActualizadoEn = now

		hist := &models.HistorialCanalizacion{CanalizacionID: c.ID, ActorID: actor.ActorID, EstadoAnterior: &anterior,
			EstadoNuevo: t.to, VersionResultante: c.Version, ClaveID: c.ClaveID}
		if motivo != "" {
			hid := uuid.New()
			hist.ID = hid
			clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, "historial_canalizacion", hid,
				[]byte(motivo), aad("historial_canalizacion", "motivo", hid))
			if err != nil {
				return err
			}
			hist.ClaveID, hist.MotivoCifrado = clave, payload
		}
		if err := r.HistorialCanalizaciones().Append(ctx, hist); err != nil {
			return err
		}
		if out, err = s.view(ctx, r, c, map[uuid.UUID]string{}); err != nil {
			return err
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "referral."+in.Action, "canalizacion",
			models.ResultadoExito, &c.ID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// assign deja al actor asignado al expediente en el departamento destino.
// Si ya tenía una asignación vigente la reutiliza.
func (s *service) assign(ctx context.Context, r repository.Repositories, expID, personalID, depID uuid.UUID, at time.Time) (uuid.UUID, error) {
	if a, err := r.Asignaciones().FindVigente(ctx, expID, personalID, depID); err == nil {
		return a.ID, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return uuid.Nil, err
	}
	id := uuid.New()
	clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, "asignacion", id,
		[]byte(motivoAsignacion), aad("asignacion", "motivo", id))
	if err != nil {
		return uuid.Nil, err
	}
	if err := r.Asignaciones().Create(ctx, &models.Asignacion{ID: id, ExpedienteID: expID, PersonalID: personalID,
		DepartamentoID: depID, AsignadaPor: personalID, Inicio: at, MotivoCifrado: payload, ClaveID: clave}); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
