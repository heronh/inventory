package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	DisableDBBootstrap bool
	SecretKey          string
	UserSeederName     string
	UserSeederEmail    string
	UserSeederPass     string
	UserSeederRole     string
	RoleSeederOne      string
	RoleSeederTwo      string
	RoleSeederThree    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "inventorydb"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		DisableDBBootstrap: getEnv("DISABLE_DB_BOOTSTRAP", "false") == "true",
		UserSeederName:     getEnv("USER_SEEDER_NAME", "Heron"),
		UserSeederEmail:    getEnv("USER_SEEDER_EMAIL", "heronhurpia@gmail.com"),
		UserSeederPass:     getEnv("USER_SEEDER_PASSWORD", "mudar123"),
		UserSeederRole:     getEnv("USER_SEEDER_ROLE", "su"),
		RoleSeederOne:      getEnv("ROLE_SEEDER_1", "su"),
		RoleSeederTwo:      getEnv("ROLE_SEEDER_2", "admin"),
		RoleSeederThree:    getEnv("ROLE_SEEDER_3", "user"),
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
