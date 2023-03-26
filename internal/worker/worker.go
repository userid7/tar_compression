package worker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"audio_compression/config"
	"audio_compression/internal/compression"
	"audio_compression/internal/controller/rmq"
	"audio_compression/internal/db/gorm/mysql"
	"audio_compression/pkg/logger"

	// tmetric "audio_compression/internal/telemetry/metric"
	ttrace "audio_compression/internal/telemetry/trace"
)

var name = "audio-compression-worker"

// NewServer ...
func NewWorker(cfg *config.Config) *Worker {
	worker := &Worker{}

	worker.InitGlobalProvider(name, cfg.JaegerEndpoint)

	return worker
}

type Worker struct {
	// metricProviderCloseFn []tmetric.CloseFunc
	traceProviderCloseFn []ttrace.CloseFunc
}

// Run ...
func (s *Worker) Run(ctx context.Context, cfg *config.Config) error {
	l := logger.New("Info")
	db := mysql.NewDB(cfg.MYSQL)

	compUsecase := compression.NewCompressionUsecase(cfg, db, l)

	amqpWoker, err := rmq.NewAMQPWorker(cfg, l, compUsecase)
	if err != nil {
		l.Fatal(err)
	}

	if err := amqpWoker.StartConsumer(); err != nil {
		l.Error(err)
	}

	l.Info("compression worker started")

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
		// case err := <-amqpWoker.Notify():
		// 	l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	log.Printf("server stopped")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()

	// Shutdown
	if err := amqpWoker.CloseChan(); err != nil {
		l.Error(fmt.Errorf("app - Run - compressionConsumer.Shutdown: %w", err))
	}

	log.Printf("server exited properly")

	sql, err := db.DB()
	if err != nil {
		log.Fatal().Msgf("unable to get db driver")
	}

	if err = sql.Close(); err != nil {
		log.Fatal().Msgf("unable close db connection")
	}

	// for _, closeFn := range s.metricProviderCloseFn {
	// 	go func() {
	// 		err = closeFn(ctxShutDown)
	// 		if err != nil {
	// 			log.Error().Err(err).Msgf("Unable to close metric provider")
	// 		}
	// 	}()
	// }
	for _, closeFn := range s.traceProviderCloseFn {
		go func() {
			err = closeFn(ctxShutDown)
			if err != nil {
				log.Error().Err(err).Msgf("Unable to close trace provider")
			}
		}()
	}

	return err
}

func (s *Worker) cors() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"POST", "GET", "PUT", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:     []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		MaxAge:             60, // 1 minutes
		AllowCredentials:   true,
		OptionsPassthrough: false,
		Debug:              false,
	})
}
