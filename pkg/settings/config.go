package settings

import (
	"time"
)

type ClientConfiguration struct {
	LongPooling    bool          `env:"GOFUSE_LONG_POOLING,default=true"`
	MaxFailedTries int           `env:"GOFUSE_MAX_FAILED_TRIES,default=5"`
	OpenTTL        time.Duration `env:"GOFUSE_OPEN_STATE_TTL,default=35s"`
	Timeout        time.Duration `env:"GOFUSE_OPEN_STATE_TTL,default=35s"`
}
