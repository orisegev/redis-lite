package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	Password         string
	SnapshotPath     string
	SnapshotInterval time.Duration
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

	snapshotPath := os.Getenv("SNAPSHOT_PATH")
	if snapshotPath == "" {
		snapshotPath = "dump.json"
	}

	snapshotInterval := 60 * time.Second
	if s := os.Getenv("SNAPSHOT_INTERVAL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			snapshotInterval = d
		}
	}

	return Config{
		Port:             port,
		Password:         password,
		SnapshotPath:     snapshotPath,
		SnapshotInterval: snapshotInterval,
	}
}
