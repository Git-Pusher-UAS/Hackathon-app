package httpx

// ErrorDetail es el detalle del sobre de error de /api.
type ErrorDetail struct {
	Code      string `json:"code" example:"VALIDATION_ERROR"`
	Message   string `json:"message" example:"alias debe tener entre 1 y 100 caracteres"`
	RequestID string `json:"requestId" example:"6f1c2f7e-6f3f-4f0e-9f3a-2b1d4c5e6f70"`
}

// ErrorResponse es el sobre de error uniforme de /api.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}
