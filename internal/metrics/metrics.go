package metrics

import "sync/atomic"

var (
	RateLimitBlocked    atomic.Int64
	OTPChallengeCreated atomic.Int64
	OTPVerifyFailed     atomic.Int64
	OTPVerifySuccess    atomic.Int64
	ContactReleaseTotal atomic.Int64
	AuditAppendErrors   atomic.Int64
)

type Snapshot struct {
	RateLimitBlocked    int64 `json:"rateLimitBlocked"`
	OTPChallengeCreated int64 `json:"otpChallengeCreated"`
	OTPVerifyFailed     int64 `json:"otpVerifyFailed"`
	OTPVerifySuccess    int64 `json:"otpVerifySuccess"`
	ContactReleaseTotal int64 `json:"contactReleaseTotal"`
	AuditAppendErrors   int64 `json:"auditAppendErrors"`
}

func Get() Snapshot {
	return Snapshot{
		RateLimitBlocked:    RateLimitBlocked.Load(),
		OTPChallengeCreated: OTPChallengeCreated.Load(),
		OTPVerifyFailed:     OTPVerifyFailed.Load(),
		OTPVerifySuccess:    OTPVerifySuccess.Load(),
		ContactReleaseTotal: ContactReleaseTotal.Load(),
		AuditAppendErrors:   AuditAppendErrors.Load(),
	}
}
