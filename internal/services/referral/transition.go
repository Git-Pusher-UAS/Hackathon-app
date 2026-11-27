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

	"github.com/google/uuid"
)

const (
	modulo = "expedientes"
	// Motivo que queda cifrado en la asignación creada al aceptar.
	motivoAsignacion = "Asignación por canalización interna aceptada"
)

var (
	ErrNotFound             = errors.New("referral not found")
	ErrInvalidTransition    = errors.New("invalid referral transition")
	ErrCaseClosed           = errors.New("case is closed")
	ErrPreconditionRequired = errors.New("precondition required")
	ErrVersionMismatch      = errors.New("version mismatch")
)

// Transitioner aplica las transiciones de §8.3.
type Transitioner interface {
	Transition(ctx context.Context, actor repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error)
}

// regla dice desde qué estado se aplica cada acción y qué lado puede hacerlo.
// En una externa no hay departamento destino: el origen la completa.
type regla struct {
	from, to    models.EstadoCanalizacion
	porDestino  bool
	needsMotivo bool
}

var reglas = map[string]regla{
	refmodels.AccionAceptar:   {models.CanalizacionPendiente, models.CanalizacionAceptada, true, false},
	refmodels.AccionRechazar:  {models.CanalizacionPendiente, models.CanalizacionRechazada, true, true},
	refmodels.AccionCompletar: {models.CanalizacionAceptada, models.CanalizacionCompletada, true, false},
	refmodels.AccionCancelar:  {models.CanalizacionPendiente, models.CanalizacionCancelada, false, true},
}

func (s *service) Transition(ctx context.Context, actor repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error) {
	if s.keyring == nil {
		return nil, ErrCryptoUnavailable
	}
	t, ok := reglas[in.Accion]
	if !ok {
		return nil, httpx.Invalid("accion inválida", map[string]string{"accion": "valor_invalido"})
	}
	motivo := ""
	if in.Motivo != nil {
		motivo = strings.TrimSpace(*in.Motivo)
	}
	if n := utf8.RuneCountInString(motivo); (t.needsMotivo && n < 10) || n > 500 {
		return nil, httpx.Invalid("motivo inválido", map[string]string{"motivo": "motivo_required"})
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
		esDestino := c.DestinoID != nil && *c.DestinoID == p.DepartamentoID
		esOrigen := c.OrigenID == p.DepartamentoID
		if !esDestino && !esOrigen { // ajeno = inexistente
			return ErrNotFound
		}
		if c.Version != expected {
			return ErrVersionMismatch
		}
		from, lado := t.from, esOrigen
		if t.porDestino {
			lado = esDestino
		}
		// Externa: no hay departamento destino. El origen la completa desde
		// pendiente (se completa a mano) o la cancela; no se acepta ni rechaza.
		if c.Tipo == models.CanalizacionExternaTipo {
			switch in.Accion {
			case refmodels.AccionCompletar:
				from, lado = models.CanalizacionPendiente, esOrigen
			case refmodels.AccionCancelar:
			default:
				return ErrInvalidTransition
			}
		}
		if !lado || c.Estado != from {
			return ErrInvalidTransition
		}

		now := s.now()
		if in.Accion == refmodels.AccionAceptar {
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

		hist := &models.HistorialCanalizacion{CanalizacionID: c.ID, ActorID: actor.ActorID, EstadoAnterior: &anterior,
			EstadoNuevo: t.to, VersionResultante: c.Version, ClaveID: c.ClaveID}
		if motivo != "" {
			hid := uuid.New()
			hist.ID = hid
			clave, payload, err := s.keyring.Seal(ctx, r.ClavesRegistro(), modulo, "historial_canalizacion", hid,
				[]byte(motivo), AAD("historial_canalizacion", "motivo", hid))
			if err != nil {
				return err
			}
			hist.ClaveID, hist.MotivoCifrado = clave, payload
		}
		if err := r.HistorialCanalizaciones().Append(ctx, hist); err != nil {
			return err
		}
		if out, err = s.view(ctx, newNombres(r), c); err != nil {
			return err
		}
		return audit.RecordResource(ctx, r, &actor.ActorID, "referral."+in.Accion, "canalizacion",
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
		[]byte(motivoAsignacion), AAD("asignacion", "motivo", id))
	if err != nil {
		return uuid.Nil, err
	}
	if err := r.Asignaciones().Create(ctx, &models.Asignacion{ID: id, ExpedienteID: expID, PersonalID: personalID,
		DepartamentoID: depID, AsignadaPor: personalID, Inicio: at, MotivoCifrado: payload, ClaveID: clave}); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
