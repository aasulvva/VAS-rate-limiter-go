package limiter

import (
	"golang.org/x/time/rate"
	"time"
)

type Visitor struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
}

type Config struct {
	MaxRequests uint
	BurstLimit  uint
}
