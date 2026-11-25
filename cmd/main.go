package main

import (
	"backend-hackathon/internal/config"
	"backend-hackathon/internal/repositories"
	"log"
	// "os"
	// "github.com/jackc/pgx/v5"
	// "github.com/joho/godotenv"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db, err := repositories.NewDB(&cfg.DB) // cargo la configuracion de la bd
	if err != nil {
		panic(err)
	}
	defer db.Pool.Close()
	// defer conn.Close(context.Background())
}
