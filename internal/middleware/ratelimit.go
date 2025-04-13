package middleware

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter stores rate limiters for individual IP addresses.
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit // Requests per second
	b   int        // Burst size
}

// NewIPRateLimiter creates a new rate limiter for IP addresses.
// r is the allowed requests per second, b is the burst size.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
	// TODO: Consider adding a background goroutine to clean up old entries
	// from the ips map to prevent memory leaks if there are many unique IPs.
	// For now, it keeps limiters for all IPs encountered.
	return limiter
}

// AddIP creates a new rate limiter for the given IP address if it doesn't exist.
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	// Check if the IP already exists
	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
		// log.Printf("DEBUG: Added rate limiter for IP: %s", ip) // Optional debug log
	}
	return limiter
}

// GetLimiter returns the rate limiter for the given IP address.
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock() // Release read lock before potentially acquiring write lock in AddIP

	if !exists {
		// Use AddIP which handles locking internally
		return i.AddIP(ip)
	}
	return limiter
}

// RateLimiterMiddleware creates a Gin middleware for rate limiting based on IP address.
func RateLimiterMiddleware(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(r, b)
	log.Printf("INFO: Initializing IP Rate Limiter (Rate: %.2f req/s, Burst: %d)", r, b)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP == "" {
			// Should not happen with default Gin settings, but handle defensively
			log.Printf("WARN: RateLimiterMiddleware could not get ClientIP for request: %s %s", c.Request.Method, c.Request.URL.Path)
			// Fail open or closed? Let's fail open for now, but log it.
			// Alternatively, could return an error:
			// c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not determine client IP"})
			// return
			c.Next()
			return
		}

		ipLimiter := limiter.GetLimiter(clientIP)

		if !ipLimiter.Allow() {
			// log.Printf("DEBUG: Rate limit exceeded for IP: %s (Path: %s)", clientIP, c.Request.URL.Path) // Optional debug log
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
			return
		}

		c.Next()
	}
}
