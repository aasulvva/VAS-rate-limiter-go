package limiter

import (
	error_handling "github.com/aasulvva/VAS-error-handling-go"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"sync"
	"time"
)

type CooldownLimiter struct {
	Name     string
	Mutex    sync.Mutex
	Limiters map[string]Visitor
	// Threshold at which old visitors are deleted from the map (in seconds since last visit)
	VisitorAgeThreshold int
	// Frequency of cleanup (in seconds)
	CleanupFrequency int
	Configuration    CooldownConfig
}

func NewCooldownLimiter(name string, config CooldownConfig) *CooldownLimiter {
	return &CooldownLimiter{
		Name:                name,
		Mutex:               sync.Mutex{},
		Limiters:            make(map[string]Visitor),
		VisitorAgeThreshold: 180,
		CleanupFrequency:    120,
		Configuration:       config,
	}
}

func (cl *CooldownLimiter) GetLimiter(ip string) *rate.Limiter {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	// See if limiter is in map
	limiter, ok := cl.Limiters[ip]
	if !ok {
		log.Printf("[LIMITER][%s] Cooldown limiter registered for visitor %s\n", cl.Name, ip)
		cl.Limiters[ip] = Visitor{
			Limiter:  rate.NewLimiter(rate.Every(time.Duration(cl.Configuration.Cooldown)*time.Second), int(cl.Configuration.Requests)),
			LastSeen: time.Now(),
		}
		return cl.Limiters[ip].Limiter
	} else {
		log.Printf("[LIMITER][%s] Cooldown limiter for visitor %s updated! Last updated: %s\n", cl.Name, ip, limiter.LastSeen)
		cl.Limiters[ip] = Visitor{
			Limiter:  cl.Limiters[ip].Limiter,
			LastSeen: time.Now(),
		}
		return cl.Limiters[ip].Limiter
	}
}

// Function that runs periodically in a goroutine to cleanup old visitors (threshold set with const)
func (cl *CooldownLimiter) CleanupLimiters() {
	for {
		log.Printf("[LIMITER][%s] Cleaning up old visitors...\n", cl.Name)

		cl.Mutex.Lock()
		oldCount := len(cl.Limiters)

		for ip, visitor := range cl.Limiters {
			if time.Since(visitor.LastSeen) > (time.Duration(cl.VisitorAgeThreshold) * time.Second) {
				delete(cl.Limiters, ip)
			}
		}

		newCount := len(cl.Limiters)
		cl.Mutex.Unlock()
		log.Printf("[LIMITER][%s] Removed %d old visitor(s)\n", cl.Name, oldCount-newCount)

		time.Sleep(time.Duration(cl.CleanupFrequency) * time.Second)
	}

}

func (cl *CooldownLimiter) LimiterHandler(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[LIMITER] New %s request on %s from %s\n", r.Method, r.RequestURI, r.RemoteAddr)

		// Get limiter for visitor
		limiter := cl.GetLimiter(extractIP(r.RemoteAddr))

		// Check limiter
		if !limiter.Allow() {
			error_handling.LogError(w, error_handling.RateLimitCooldownError(extractIP(r.RemoteAddr), cl.Configuration.Cooldown, cl.Name))
			return
		}

		// Run handler function
		next.ServeHTTP(w, r)
	})
}
