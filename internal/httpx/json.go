// Package httpx agrupa helpers de respuesta HTTP.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorBody es el envelope estándar de error de la API.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// JSON escribe v como JSON con el status indicado.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: no se pudo escribir la respuesta: %v", err)
	}
}

// Error escribe un envelope de error.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorBody{Error: code, Message: message})
}
