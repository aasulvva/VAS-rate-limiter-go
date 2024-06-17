package limiter

import (
	"golang.org/x/time/rate"
	"log"
	"sync"
	"time"
)

type LoginLimiter struct {
	Mutex    sync.Mutex
	Limiters map[LoginIdentifier]Visitor
	// Threshold at which old visitors are deleted from the map (in seconds since last visit)
	VisitorAgeThreshold int
	// Frequency of cleanup (in seconds)
	CleanupFrequency int
	Configuration    CooldownConfig
}

func NewLoginLimiter(config CooldownConfig) *LoginLimiter {
	return &LoginLimiter{
		Mutex:               sync.Mutex{},
		Limiters:            make(map[LoginIdentifier]Visitor),
		VisitorAgeThreshold: int(config.Cooldown * 2),
		CleanupFrequency:    int(config.Cooldown * 3 / 2),
		Configuration:       config,
	}
}

func (ll *LoginLimiter) GetLimiter(userId uint, ip string) *rate.Limiter {
	ll.Mutex.Lock()
	defer ll.Mutex.Unlock()

	identifier := LoginIdentifier{
		UserId:    userId,
		IpAddress: ip,
	}

	// See if limiter is in map
	limiter, ok := ll.Limiters[identifier]
	if !ok {
		log.Printf("[LIMITER] Cooldown limiter registered for user with ID %d connecting from %s\n", userId, ip)
		ll.Limiters[identifier] = Visitor{
			Limiter:  rate.NewLimiter(rate.Every(time.Duration(ll.Configuration.Cooldown)*time.Second), int(ll.Configuration.Requests)),
			LastSeen: time.Now(),
		}
		return ll.Limiters[identifier].Limiter
	} else {
		log.Printf("[LIMITER] Cooldown limiter for user with ID %d connecting from %s updated! Last updated: %s\n", userId, ip, limiter.LastSeen)
		ll.Limiters[identifier] = Visitor{
			Limiter:  ll.Limiters[identifier].Limiter,
			LastSeen: time.Now(),
		}
		return ll.Limiters[identifier].Limiter
	}
}

// Function that runs periodically in a goroutine to cleanup old visitors (threshold set with const)
func (ll *LoginLimiter) CleanupLimiters() {
	for {
		log.Println("[LIMITER] Cleaning up old login limiters...")

		ll.Mutex.Lock()
		oldCount := len(ll.Limiters)

		for id, visitor := range ll.Limiters {
			if time.Since(visitor.LastSeen) > (time.Duration(ll.VisitorAgeThreshold) * time.Second) {
				delete(ll.Limiters, id)
			}
		}

		newCount := len(ll.Limiters)
		ll.Mutex.Unlock()
		log.Printf("[LIMITER] Removed %d old login limiter(s)\n", oldCount-newCount)

		time.Sleep(time.Duration(ll.CleanupFrequency) * time.Second)
	}

}
