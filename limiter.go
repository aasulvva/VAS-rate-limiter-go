package limiter

import (
	error_handling "github.com/aasulvva/VAS-error-handling-go"
	util "github.com/aasulvva/VAS-util-go"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	Mutex    sync.Mutex
	Limiters map[string]Visitor
	// Threshold at which old visitors are deleted from the map (in seconds since last visit)
	VisitorAgeThreshold int
	// Frequency of cleanup (in seconds)
	CleanupFrequency int
	Configuration    LimiterConfig
}

func NewLimiter(config LimiterConfig) *RateLimiter {
	return &RateLimiter{
		Mutex:               sync.Mutex{},
		Limiters:            make(map[string]Visitor),
		VisitorAgeThreshold: VISITOR_AGE_THRESHOLD,
		CleanupFrequency:    CLEANUP_FREQUENCY,
		Configuration:       config,
	}
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.Mutex.Lock()
	defer rl.Mutex.Unlock()

	// See if limiter is in map
	limiter, ok := rl.Limiters[ip]
	if !ok {
		log.Printf("[LIMITER] New visitor %s\n", ip)
		rl.Limiters[ip] = Visitor{
			Limiter:  rate.NewLimiter(rate.Limit(rl.Configuration.MaxRequests), int(rl.Configuration.BurstLimit)),
			LastSeen: time.Now(),
		}
		return rl.Limiters[ip].Limiter
	} else {
		log.Printf("[LIMITER] Visitor %s already seen! Last seen: %s\n", ip, limiter.LastSeen)
		rl.Limiters[ip] = Visitor{
			Limiter:  rl.Limiters[ip].Limiter,
			LastSeen: time.Now(),
		}
		return rl.Limiters[ip].Limiter
	}
}

// Function that runs periodically in a goroutine to cleanup old visitors (threshold set with const)
func (rl *RateLimiter) CleanupLimiters() {
	for {
		log.Println("[LIMITER] Cleaning up old visitors...")

		rl.Mutex.Lock()
		oldCount := len(rl.Limiters)

		for ip, visitor := range rl.Limiters {
			if time.Since(visitor.LastSeen) > (time.Duration(rl.VisitorAgeThreshold) * time.Second) {
				delete(rl.Limiters, ip)
			}
		}

		newCount := len(rl.Limiters)
		rl.Mutex.Unlock()
		log.Printf("[LIMITER] Removed %d old visitor(s)\n", oldCount-newCount)

		time.Sleep(time.Duration(rl.CleanupFrequency) * time.Second)
	}

}

func (rl *RateLimiter) LimiterHandler(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[LIMITER] New %s request on %s from %s\n", r.Method, r.RequestURI, r.RemoteAddr)

		// Get limiter for visitor
		limiter := rl.GetLimiter(util.ExtractIP(r.RemoteAddr))

		// Check limiter
		if !limiter.Allow() {
			error_handling.LogError(w, error_handling.RateLimitError(util.ExtractIP(r.RemoteAddr)))
			return
		}

		// Run handler function
		next.ServeHTTP(w, r)
	})
}
