package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"audio_compression/config"
	v1 "audio_compression/internal/controller/http/v1"
	"audio_compression/internal/controller/rmq"
	"audio_compression/pkg/httpserver"
	"audio_compression/pkg/logger"

	// tmetric "audio_compression/internal/telemetry/metric"
	ttrace "audio_compression/internal/telemetry/trace"
)

var name = "server"

// NewServer ...
func NewServer(cfg *config.Config) *Server {
	srv := &Server{}

	srv.InitGlobalProvider(name, cfg.JaegerEndpoint)

	return srv
}

type Server struct {
	// metricProviderCloseFn []tmetric.CloseFunc
	traceProviderCloseFn []ttrace.CloseFunc
}

// Run ...
func (s *Server) Run(ctx context.Context, cfg *config.Config) error {
	l := logger.New("Info")
	l.Info("Starting server...")

	AMQPClient, err := rmq.NewAMQPClient(cfg, l)
	if err != nil {
		l.Fatal(err)
	}
	go AMQPClient.DecompressionConsumer()

	handler := gin.New()
	v1.NewRouter(handler, l, AMQPClient)
	httpServer := httpserver.New(handler, httpserver.Port(cfg.Server.Port))

	l.Info("server serving on port %d ", cfg.Server.Port)

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
	case err := <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	log.Printf("server stopped")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()

	// Shutdown
	if err := httpServer.Shutdown(); err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}

	log.Printf("server exited properly")

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

func (s *Server) cors() *cors.Cors {
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
