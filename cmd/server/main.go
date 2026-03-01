package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"inventory/internal/config"
	"inventory/internal/database"
	"inventory/internal/handlers"
	"inventory/internal/models"
	"inventory/internal/seeder"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err = database.EnsureDatabaseContainer(ctx, cfg); err != nil {
		log.Fatalf("database bootstrap failed: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err = db.AutoMigrate(
		&models.Privilege{},
		&models.User{},
		&models.Tarefas{},
		&models.Client{},
		&models.Supplier{},
		&models.Image{},
		&models.Product{},
		&models.Entry{},
		&models.Sale{},
		&models.Log{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	if err = seeder.Seed(db, cfg); err != nil {
		log.Fatalf("seeding failed: %v", err)
	}

	app, err := handlers.NewApp(db, cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      app.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("inventory server running on port %s", cfg.Port)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
