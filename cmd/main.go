package main

import (
	"log"

	"backend-hackathon/internal/config"
	gormrepo "backend-hackathon/internal/repository/gorm"
)

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

	log.Println("conexión a la base de datos establecida (GORM)")
}
