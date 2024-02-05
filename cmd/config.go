package main

import (
	"time"
)

type Config struct {
	Settings struct {
		LongPooling     bool          `env:"LONG_POOLING,default=true"`
		MaxFailedTries  int           `env:"MAX_FAILED_TRIES,default=5"`
		OpenStateExpiry time.Duration `env:"OPEN_STATE_EXPIRY,default=35s"`
	}
}
