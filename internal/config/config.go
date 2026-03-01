package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	Host            string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	SecretKey       string
	UserSeederName  string
	UserSeederEmail string
	UserSeederPass  string
	UserSeederRole  string
	RoleSeederOne   string
	RoleSeederTwo   string
	RoleSeederThree string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:            getEnv("PORT", "8008"),
		Host:            getEnv("HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "inventorydb"),
		SecretKey:       getEnv("SECRET_KEY", "secret"),
		UserSeederName:  getEnv("USER_SEEDER_NAME", "Heron"),
		UserSeederEmail: getEnv("USER_SEEDER_EMAIL", "heronhurpia@gmail.com"),
		UserSeederPass:  getEnv("USER_SEEDER_PASSWORD", "mudar123"),
		UserSeederRole:  getEnv("USER_SEEDER_ROLE", "su"),
		RoleSeederOne:   getEnv("ROLE_SEEDER_1", "su"),
		RoleSeederTwo:   getEnv("ROLE_SEEDER_2", "admin"),
		RoleSeederThree: getEnv("ROLE_SEEDER_3", "user"),
	}

	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("SECRET_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
