package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	// Config -.
	Config struct {
		App    `yaml:"app"`
		Server `yaml:"server"`
		Log    `yaml:"logger"`
		MYSQL  `yaml:"mysql"`
		RMQ    `yaml:"rabbitmq"`
		OTEL   `yaml:"otel"`
	}

	// App -.
	App struct {
		Name    string `env-required:"true" yaml:"name"    env:"APP_NAME"`
		Version string `env-required:"true" yaml:"version" env:"APP_VERSION"`
	}

	// Server -.
	Server struct {
		Port string `env-required:"true" yaml:"port" env:"HTTP_PORT"`
	}

	// Log -.
	Log struct {
		Level string `env-required:"true" yaml:"log_level"   env:"LOG_LEVEL"`
	}

	// MYSQL -.
	MYSQL struct {
		// URL string `env-required:"false" yaml:"url"                env:"MYSQL_URL"`
		Host     string `env-required:"true" yaml:"host"   env:"MYSQL_HOST"`
		Port     string `env-required:"true" yaml:"port"   env:"MYSQL_PORT"`
		Username string `env-required:"true" yaml:"username"   env:"MYSQL_USERNAME"`
		Password string `env-required:"true" yaml:"password"   env:"MYSQL_PASSWORD"`
		Dbname   string `env-required:"true" yaml:"dbname"   env:"MYSQL_DBNAME"`
	}

	// RMQ -.
	RMQ struct {
		ServerExchange string `env-required:"true" yaml:"rpc_server_exchange" env:"RMQ_RPC_SERVER"`
		ClientExchange string `env-required:"true" yaml:"rpc_client_exchange" env:"RMQ_RPC_CLIENT"`
		URL            string `env-required:"false" yaml:"url" env:"RMQ_URL"`
	}

	OTEL struct {
		JaegerEndpoint string `env-required:"true" yaml:"jaeger_endpoint" env:"JAEGER_ENDPOINT"`
		PrometheusPort string `env-required:"true" yaml:"prometheus_port" env:"PROMETHEUS_PORT"`
	}
)

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("./config/config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	// err = cleanenv.ReadEnv(cfg)
	// if err != nil {
	// 	return nil, err
	// }

	return cfg, nil
}
