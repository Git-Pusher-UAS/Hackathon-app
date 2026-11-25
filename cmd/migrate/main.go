package main

import (
	"log"

	"backend-hackathon/internal/config"
	gormrepo "backend-hackathon/internal/repository/gorm"
)

// Comando de migración:
//
//	go run ./cmd/migrate
//
// Crea los esquemas y aplica AutoMigrate sobre las tablas del modelo.
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

	if err := gormrepo.Migrate(db); err != nil {
		log.Fatalf("migración: %v", err)
	}

	log.Println("migración completada (esquemas + tablas)")
}
