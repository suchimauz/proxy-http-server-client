package config

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "golang_template"
)

type (
	Config struct {
		timezone string `envconfig:"app_timezone" default:"UTC"` // String timezone format
		Timezone *time.Location
	}
)

func NewConfig() (*Config, error) {
	var cfg Config

	godotenv.Load()

	// Parse variables from environment or return err
	err := envconfig.Process(appName, &cfg)
	if err != nil {
		return nil, err
	}

	// Parse timezone from cfg.tz or return err
	cfg.Timezone, err = time.LoadLocation(cfg.timezone)
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (cfg *Config) validate() error {
	// pass some validations here
	return nil
}
