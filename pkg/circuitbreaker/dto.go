package circuitbreaker

import (
	"time"

	"github.com/dnsx2k/gofuse/pkg/settings"
)

type Host struct {
	ID            string
	State         State
	OpenExpiresAt time.Time
	FailuresCount int
	Settings      *settings.ClientConfiguration
}

type State string

const (
	StateClosed   State = "closed"
	StateOpen           = "open"
	StateHalfOpen       = "half-open"
)
