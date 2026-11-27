package casefile

import "errors"

var (
	ErrNotFound             = errors.New("record not found")
	ErrCryptoUnavailable    = errors.New("cryptographic service unavailable")
	ErrAlreadyAssigned      = errors.New("already assigned")
	ErrRequestNotPending    = errors.New("access request not pending")
	ErrNoPrivacyNotice      = errors.New("privacy notice required")
	ErrDuplicatePerson      = errors.New("duplicate person")
	ErrPreconditionRequired = errors.New("precondition required")
	ErrVersionMismatch      = errors.New("version mismatch")
	ErrInvalidTransition    = errors.New("invalid state transition")
	ErrCaseClosed           = errors.New("case is closed")
	ErrAlreadyRevoked       = errors.New("already revoked")
	ErrLastAssignment       = errors.New("cannot remove last assignment")
	ErrNotRevocable         = errors.New("cannot revoke record")
	ErrReferralExists       = errors.New("referral already exists")
)
