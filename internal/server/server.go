package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"backend-hackathon/internal/config"
	"backend-hackathon/internal/handlers"
)

// Server envuelve el http.Server con apagado ordenado.
type Server struct {
	httpServer *http.Server
	log        *log.Logger
}

// New construye el servidor HTTP con timeouts y el router de la API.
func New(cfg *config.Config, db handlers.Pinger, version string) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              ":" + cfg.Server.Port,
			Handler:           router(cfg, db, version),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		log: log.Default(),
	}
}

// Addr devuelve la dirección de escucha (por ejemplo ":3001").
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Run escucha hasta que ctx se cancele y luego apaga el servidor con un
// timeout de 10s para terminar las peticiones en curso.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		s.log.Println("apagando servidor...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}
