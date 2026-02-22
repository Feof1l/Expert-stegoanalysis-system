package config

import (
	"main/logger"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	LogLevel  logger.LogLevel `env:"LOG_LEVEL" envDefault:"1"`
	LogDir    string          `env:"LOG_DIR" envDefault:"./logs"`
	Database  DatabaseConfig  `envPrefix:"DATABASE_"`
	Addr      string          `env:"ADDR" envDefault:":8080"`
	JWTSecret string          `env:"JWT_SECRET"`
}

type DatabaseConfig struct {
	URI string `env:"URI"`
}

func Load() (*Config, error) {
	if err := godotenv.Load("../.env"); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
