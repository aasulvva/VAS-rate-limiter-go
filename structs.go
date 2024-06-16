package limiter

import (
	"golang.org/x/time/rate"
	"time"
)

type Visitor struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
}

type LoginIdentifier struct {
	UserId    uint
	IpAddress string
}

type LimiterConfig struct {
	MaxRequests uint
	BurstLimit  uint
}

type CooldownConfig struct {
	Requests uint
	Cooldown uint
}
