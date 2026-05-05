package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	Password string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "6379"
	}

	password := os.Getenv("AUTH_PASSWORD")
	if password == "" {
		password = "defaultpassword"
		log.Println("Warning: AUTH_PASSWORD not set, using default")
	}

	return Config{
		Port:     port,
		Password: password,
	}
}
