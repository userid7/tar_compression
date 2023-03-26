package main

import (
	"context"
	"log"

	"audio_compression/config"
	"audio_compression/internal/worker"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	ctx := context.Background()
	w := worker.NewWorker(cfg)
	w.Run(ctx, cfg)
}
