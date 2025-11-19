package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Middleware is a function that wraps an HTTP handler
type Middleware func(http.Handler) http.Handler

// ChainMiddleware chains multiple middleware functions together
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// MVP: Allow all origins
			// In production, validate origin
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			logger.Info("HTTP request",
				logger.String("method", r.Method),
				logger.String("path", r.URL.Path),
				logger.String("remote_addr", r.RemoteAddr),
				logger.Int("status", wrapped.statusCode),
				logger.Duration("duration", duration),
			)
		})
	}
}

// ErrorHandlingMiddleware handles errors and returns JSON responses
func ErrorHandlingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic in handler",
						logger.String("path", r.URL.Path),
						logger.String("error", err.(string)),
					)
					respondWithError(w, http.StatusInternalServerError, "Internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware implements simple rate limiting
func RateLimitMiddleware(requestsPerSecond int) Middleware {
	type clientInfo struct {
		count     int
		lastReset time.Time
	}

	clients := make(map[string]*clientInfo)
	var mu sync.RWMutex

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for key, info := range clients {
				if now.Sub(info.lastReset) > 1*time.Minute {
					delete(clients, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			now := time.Now()

			mu.Lock()
			info, exists := clients[clientIP]
			if !exists || now.Sub(info.lastReset) >= 1*time.Second {
				info = &clientInfo{
					count:     1,
					lastReset: now,
				}
				clients[clientIP] = info
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			info.count++
			if info.count > requestsPerSecond {
				mu.Unlock()
				respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates JWT tokens and injects user context
func AuthMiddleware(jwtSecret string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health endpoints
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/live" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// MVP: Allow requests without auth (use default user)
				// In production, this should be required
				ctx := context.WithValue(r.Context(), "user_id", "default")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Validate token (reuse auth logic from wsgateway)
			// For now, MVP allows default user
			ctx := context.WithValue(r.Context(), "user_id", "default")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Helper functions

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  code,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// Check X-Real-IP header
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

