package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Accrual   string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:5557"`
	Address   string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	RateLimit int    `env:"RATE_LIMIT" envDefault:"1"`
	Key       string `env:"HASH_KEY" envDefault:""`
	Database  string `env:"DATABASE_URI" envDefault:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"`
	Storage   bool
}

type FlagConfig struct {
	Accrual   *string
	Address   *string
	RateLimit *int
	Key       *string
	Database  *string
	Storage   *bool
}

type UserID string

var (
	Cfg   Config
	Flags FlagConfig
)

func InitFlags() {
	Flags.Address = flag.String("a", "localhost:8080", "server address in format host:port")
	Flags.Database = flag.String("d", "", "postgres database uri")
	Flags.Accrual = flag.String("r", "localhost:8081", "accrual system address in format host:port")
	Flags.Key = flag.String("k", "abcd", "key to hash")
	Flags.RateLimit = flag.Int("l", 1, "max amount of goroutines working simultaneously")
	Flags.Storage = flag.Bool("s", false, "inmemory storage for lazy debugging")
	flag.Parse()
}

func SetConfig() {
	env.Parse(&Cfg)
	if _, check := os.LookupEnv("RUN_ADDRESS"); !check {
		Cfg.Address = *Flags.Address
	}
	if _, check := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); !check {
		Cfg.Accrual = *Flags.Accrual
	}
	if _, check := os.LookupEnv("DATABASE_URI"); !check {
		Cfg.Database = *Flags.Database
	}
	if _, check := os.LookupEnv("HASH_KEY"); !check {
		Cfg.Key = *Flags.Key
	}
	if _, check := os.LookupEnv("RATE_LIMIT"); !check {
		Cfg.RateLimit = *Flags.RateLimit
	}
	if _, check := os.LookupEnv("MEMSTORAGEUSE"); !check {
		Cfg.Storage = *Flags.Storage
	}
}
