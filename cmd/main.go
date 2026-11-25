package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"backend-hackathon/internal/config"
	gormrepo "backend-hackathon/internal/repository/gorm"
	"backend-hackathon/internal/server"
)

// version se puede sobrescribir en el build:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd
var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := gormrepo.Open(&cfg.DB)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// El UnitOfWork es el punto de entrada para la capa de servicios.
	_ = gormrepo.NewUnitOfWork(db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg, sqlDB, version)
	log.Printf("API escuchando en %s (version %s)", srv.Addr(), version)

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server: %v", err)
	}

	log.Println("servidor detenido")
}
