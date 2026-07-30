package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/Basith-08/tracklume-api/internal/security"
	"github.com/google/uuid"
)

type contextKey string

type AccountStatusChecker interface {
	IsActive(context.Context, uuid.UUID) (bool, error)
}

const userIDKey contextKey = "authenticated_user_id"
const requestIDKey contextKey = "request_id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.NewString()
		}
		r.Header.Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func Authenticate(tokens *security.TokenManager, checker AccountStatusChecker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := r.Header.Get("Authorization")
		parts := strings.Fields(value)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required", nil)
			return
		}
		userID, err := tokens.Parse(parts[1])
		if err != nil {
			response.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required", nil)
			return
		}
		if checker != nil {
			active, checkErr := checker.IsActive(r.Context(), userID)
			if checkErr != nil || !active {
				response.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required", nil)
				return
			}
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func BodyLimit(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "request_id", RequestIDFromContext(r.Context()), "panic", recovered)
					response.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)
			attrs := []any{"request_id", RequestIDFromContext(r.Context()), "method", r.Method, "path", r.URL.Path, "status", ww.status, "duration_ms", time.Since(started).Milliseconds(), "client_ip", clientIP(r)}
			if userID, ok := UserIDFromContext(r.Context()); ok {
				attrs = append(attrs, "user_id", userID.String())
			}
			logger.Info("http request", attrs...)
		})
	}
}

func Timeout(duration time.Duration, next http.Handler) http.Handler {
	return http.TimeoutHandler(next, duration, `{"error":{"code":"TIMEOUT","message":"Request timed out"}}`)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusWriter) Write(body []byte) (int, error) { return w.ResponseWriter.Write(body) }

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

type limiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	max     int
	window  time.Duration
}

func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	state := &limiter{entries: make(map[string][]time.Time), max: max, window: window}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			now := time.Now()
			state.mu.Lock()
			recent := state.entries[key][:0]
			for _, at := range state.entries[key] {
				if now.Sub(at) < state.window {
					recent = append(recent, at)
				}
			}
			if len(recent) >= state.max {
				state.entries[key] = recent
				state.mu.Unlock()
				response.WriteError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests", nil)
				return
			}
			state.entries[key] = append(recent, now)
			state.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
