package referral

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"backend-hackathon/internal/audit"
	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	refmodels "backend-hackathon/internal/models/referral"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/google/uuid"
)

const (
	modulo           = "expedientes"
	motivoAsignacion = "Asignacion por canalizacion interna aceptada"
)

var (
	ErrNotFound             = errors.New("referral not found")
	ErrInvalidTransition    = errors.New("invalid referral transition")
	ErrCaseClosed           = errors.New("case is closed")
	ErrPreconditionRequired = errors.New("precondition required")
	ErrVersionMismatch      = errors.New("version mismatch")
)

type Transitioner interface {
	Transition(ctx context.Context, actor repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error)
}

type regla struct {
	from, to    models.EstadoCanalizacion
	porDestino  bool
	needsMotivo bool
}

var reglas = map[string]regla{
	"accept":   {models.CanalizacionPendiente, models.CanalizacionAceptada, true, false},
	"reject":   {models.CanalizacionPendiente, models.CanalizacionRechazada, true, true},
	"complete": {models.CanalizacionAceptada, models.CanalizacionCompletada, true, false},
	"cancel":   {models.CanalizacionPendiente, models.CanalizacionCancelada, false, true},
	// compat
	"aceptar":   {models.CanalizacionPendiente, models.CanalizacionAceptada, true, false},
	"rechazar":  {models.CanalizacionPendiente, models.CanalizacionRechazada, true, true},
	"completar": {models.CanalizacionAceptada, models.CanalizacionCompletada, true, false},
	"cancelar":  {models.CanalizacionPendiente, models.CanalizacionCancelada, false, true},
}

func normalizeAction(a string) string { return strings.ToLower(strings.TrimSpace(a)) }

func (s *service) Transition(ctx context.Context, actor repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	action := normalizeAction(in.Action)
	t, ok := reglas[action]
	if !ok {
		return nil, httpx.Invalid("accion invalida", map[string]string{"action": "valor_invalido"})
	}
	motivo := ""
	if in.Motivo != nil {
		motivo = strings.TrimSpace(*in.Motivo)
	}
	if n := utf8.RuneCountInString(motivo); (t.needsMotivo && n < 10) || n > 500 {
		return nil, httpx.Invalid("motivo invalido", map[string]string{"motivo": "longitud_invalida"})
	}
	raw := strings.TrimSpace(ifMatch)
	if raw == "" {
		return nil, ErrPreconditionRequired
	}
	var expected int64
	if tag, ok := httpx.ParseIfMatch(raw); ok {
		v, err := httpx.DecodeVersionTag(s.secret, tag)
		if err != nil {
			return nil, ErrVersionMismatch
		}
		expected = v
	} else if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
		expected = v
	} else if v, err := httpx.DecodeVersionTag(s.secret, raw); err == nil {
		expected = v
	} else {
		return nil, ErrVersionMismatch
	}

	var out refmodels.View
	err := s.uow.Do(ctx, actor, func(ctx context.Context, r repository.Repositories) error {
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
		esDestino := c.DestinoID != nil && *c.DestinoID == p.DepartamentoID
		esOrigen := c.OrigenID == p.DepartamentoID
		if !esDestino && !esOrigen {
			return ErrNotFound
		}
		if c.Version != expected {
			return ErrVersionMismatch
		}
		from, lado := t.from, esOrigen
		if t.porDestino {
			lado = esDestino
		}
		if c.Tipo == models.CanalizacionExternaTipo {
			switch action {
			case "complete", "completar":
				from, lado = models.CanalizacionPendiente, esOrigen
			case "cancel", "cancelar":
			default:
				return ErrInvalidTransition
			}
		}
		if !lado || c.Estado != from {
			return ErrInvalidTransition
		}

		now := s.now()
		if action == "accept" || action == "aceptar" {
			e, err := r.Expedientes().GetByID(ctx, c.ExpedienteID)
			if err != nil {
				return err
			}
			if e.Estado == models.ExpedienteCerrado {
				return ErrCaseClosed
			}
			asig, err := s.assign(ctx, r, c.ExpedienteID, actor.ActorID, *c.DestinoID, now)
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

		hist := &models.HistorialCanalizacion{CanalizacionID: c.ID, ActorID: actor.ActorID, EstadoAnterior: &anterior, EstadoNuevo: t.to, VersionResultante: c.Version, ClaveID: c.ClaveID}
		if motivo != "" {
			hid := uuid.New()
			hist.ID = hid
			clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, "historial_canalizacion", hid, []byte(motivo), AAD("historial_canalizacion", "motivo", hid))
			if err != nil {
				return err
			}
			hist.ClaveID, hist.MotivoCifrado = clave, payload
		}
		if err := r.HistorialCanalizaciones().Append(ctx, hist); err != nil {
			return err
		}
		out, err = s.view(ctx, newNombres(r), c)
		if err != nil {
			return err
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "referral."+action, "canalizacion", models.ResultadoExito, &c.ID, actor.RequestID, actor.IPOrigen)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) assign(ctx context.Context, r repository.Repositories, expID, personalID, depID uuid.UUID, at time.Time) (uuid.UUID, error) {
	if a, err := r.Asignaciones().FindVigente(ctx, expID, personalID, depID); err == nil {
		return a.ID, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return uuid.Nil, err
	}
	id := uuid.New()
	clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, "asignacion", id, []byte(motivoAsignacion), AAD("asignacion", "motivo", id))
	if err != nil {
		return uuid.Nil, err
	}
	if err := r.Asignaciones().Create(ctx, &models.Asignacion{ID: id, ExpedienteID: expID, PersonalID: personalID, DepartamentoID: depID, AsignadaPor: personalID, Inicio: at, MotivoCifrado: payload, ClaveID: clave}); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
