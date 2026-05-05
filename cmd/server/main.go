package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/orisegev/redis-lite/internal/config"
	"github.com/orisegev/redis-lite/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
