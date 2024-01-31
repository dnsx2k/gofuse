package main

import (
	"time"

	"github.com/ardanlabs/conf"
)

type Config struct {
	conf.Version
	Settings struct {
		LongPooling     bool          `conf:"default:false,help:"`
		MaxFailedTries  int           `conf:"default:3,help:Counter for failed calls, circuit-breaker will change it's state to open after reaching maxFailedTries"`
		OpenStateExpiry time.Duration `conf:"default:35s,help:Duration that circuit-breaker's open state will remain for"`
	}
}
