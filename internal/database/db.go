package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os/exec"
	"time"

	"inventory/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := dsn(cfg)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func EnsureDatabaseContainer(ctx context.Context, cfg *config.Config) error {
	if checkTCP(cfg.Host, cfg.DBPort) && checkSQL(cfg) {
		return nil
	}

	if err := runComposeUp(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(90 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("database startup timeout: %w", ctx.Err())
		case <-timeout.C:
			return fmt.Errorf("database did not become reachable in time")
		case <-ticker.C:
			if checkTCP(cfg.Host, cfg.DBPort) && checkSQL(cfg) {
				return nil
			}
		}
	}
}

func runComposeUp(ctx context.Context) error {
	commands := [][]string{
		{"docker", "compose", "up", "-d", "db"},
		{"docker-compose", "up", "-d", "db"},
	}

	for _, command := range commands {
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("unable to start database container with docker compose")
}

func checkTCP(host, port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func checkSQL(cfg *config.Config) bool {
	db, err := sql.Open("pgx", dsn(cfg))
	if err != nil {
		return false
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return false
	}

	return true
}

func dsn(cfg *config.Config) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)
}
