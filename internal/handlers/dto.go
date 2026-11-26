package handlers

// ErrorResponse es el formato uniforme de error devuelto por la API.
type ErrorResponse struct {
	Error string `json:"error" example:"error interno"`
}
