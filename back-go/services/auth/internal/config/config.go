// Package config — env-конфигурация auth-сервиса. Загружается один раз
// при старте; в случае отсутствия required-полей сервис не стартует.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`
	GRPCAddr string `env:"GRPC_ADDR" envDefault:":9090"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	ServiceName string `env:"SERVICE_NAME" envDefault:"auth"`
}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	return c, nil
}
