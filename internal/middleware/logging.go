package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseRecorder captura el status y el tamaño de la respuesta para el log de acceso.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Logging escribe una línea por petición con método, ruta, status y duración.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf("%s %s %d %dB %s request_id=%s",
			r.Method, r.URL.RequestURI(), status, rec.bytes,
			time.Since(start).Round(time.Microsecond), RequestIDFrom(r.Context()))
	})
}
