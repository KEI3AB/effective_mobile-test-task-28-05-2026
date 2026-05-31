package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTP HTTPConfig
	PG   PGConfig
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT" env-default:"8080"`
}

type PGConfig struct {
	Host     string `env:"POSTGRES_HOST" env-required:"true"`
	Port     int    `env:"POSTGRES_PORT" env-default:"5432"`
	User     string `env:"POSTGRES_USER" env-required:"true"`
	Password string `env:"POSTGRES_PASSWORD" env-required:"true"`
	DBName   string `env:"POSTGRES_DB" env-required:"true"`
	PoolMax  int    `env:"POSTGRES_POOL_MAX" env-default:"10"`
}

func MustLoad() *Config {
	_ = godotenv.Load()

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
