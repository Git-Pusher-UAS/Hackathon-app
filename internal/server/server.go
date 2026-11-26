package server

import (
	"context"
	"log"
	"time"

	"backend-hackathon/internal/config"
	"backend-hackathon/internal/handlers"

	"github.com/gofiber/fiber/v3"
)

// Server envuelve la aplicación Fiber con apagado ordenado.
type Server struct {
	app  *fiber.App
	addr string
	log  *log.Logger
}

// New construye el servidor HTTP con timeouts y el router de la API.
func New(cfg *config.Config, db handlers.Pinger, version string) *Server {
	s := &Server{
		app:  router(cfg, db, version),
		addr: ":" + cfg.Server.Port,
		log:  log.Default(),
	}
	s.app.Hooks().OnPreShutdown(func() error {
		s.log.Println("apagando servidor...")
		return nil
	})
	return s
}

// Addr devuelve la dirección de escucha (por ejemplo ":3001").
func (s *Server) Addr() string {
	return s.addr
}

// Run escucha hasta que ctx se cancele y espera hasta 10 s a las peticiones
// en curso. Fiber sincroniza el arranque y el apagado, incluso si ctx ya terminó.
func (s *Server) Run(ctx context.Context) error {
	var shutdownErr error
	s.app.Hooks().OnPostShutdown(func(err error) error {
		shutdownErr = err
		return nil
	})
	err := s.app.Listen(s.addr, fiber.ListenConfig{
		ListenerNetwork:       fiber.NetworkTCP,
		GracefulContext:       ctx,
		ShutdownTimeout:       10 * time.Second,
		DisableStartupMessage: true,
	})
	// Listen espera a OnPostShutdown antes de retornar; conserva también los
	// errores de apagado, que Fiber entrega al hook y no al retorno de Listen.
	if err != nil {
		return err
	}
	return shutdownErr
}
