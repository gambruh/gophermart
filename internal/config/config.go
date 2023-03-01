package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Accrual   string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:""`
	Address   string `env:"RUN_ADDRESS" envDefault:"127.0.0.1:8080"`
	RateLimit int    `env:"RATE_LIMIT" envDefault:"1"`
	Key       string `env:"HASH_KEY" envDefault:""`
	Database  string `env:"DATABASE_URI" envDefault:""`
}

type FlagConfig struct {
	Accrual   *string
	Address   *string
	RateLimit *int
	Key       *string
	Database  *string
}

var (
	Cfg   Config
	Flags FlagConfig
)

func InitFlags() {
	//flags. Default value of time.Duration is in ns, so to get 2s and 10s I had to multiply it

	Flags.Address = flag.String("a", "127.0.0.1:8080", "server address in format host:port")
	Flags.Database = flag.String("d", "", "postgres database uri")
	Flags.Accrual = flag.String("r", "", "ACCRUAL_SYSTEM_ADDRESS")
	Flags.Key = flag.String("k", "abcd", "key to hash")
	Flags.RateLimit = flag.Int("l", 1, "max amount of goroutines working simultaneously")
	flag.Parse()
}

func SetConfig() {
	// getting environment variables to set server address and poll and report intervals
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
}