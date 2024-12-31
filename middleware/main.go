package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Predixus/dyna-rag/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var jwtSecret string

// setup set up the environment. Required environment variables:
// - JWT_SECRET - the private JWT secret for verifying tokens
func setup() string {
	godotenv.Load()
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("`JWT_SECRET` not provided")
	}

	return jwtSecret
}

func init() {
	jwtSecret = setup()
}

type ResponseData struct {
	http.ResponseWriter
	statusCode int
	wrapped    bool
}

type RateLimiter struct {
	client        *redis.Client
	windowSeconds int
	maxRequests   int
}

func NewRateLimiter(redisURL string, windowSeconds, maxRequests int) (*RateLimiter, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Failed to connect to Redis: %w", err)
	}

	return &RateLimiter{
		client:        client,
		windowSeconds: windowSeconds,
		maxRequests:   maxRequests,
	}, nil
}

// isAllowed checks if the request is allowed using a sliding window in Redis
func (rl *RateLimiter) isAllowed(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s", ip)
	now := time.Now().Unix()
	windowStart := now - int64(rl.windowSeconds)

	pipe := rl.client.Pipeline()

	// remove old entries
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))
	// add current timestamp
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})

	// get count of requests in window
	countCmd := pipe.ZCard(ctx, key)

	// set key expiration
	pipe.Expire(ctx, key, time.Duration(rl.windowSeconds)*time.Second)

	// execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("redis pipeline failed: %w", err)
	}

	count := countCmd.Val()
	return count <= int64(rl.maxRequests), nil
}

func WrapWriter(w http.ResponseWriter) *ResponseData {
	// if already wrapped, don't wrap again
	if rd, ok := w.(*ResponseData); ok {
		return rd
	}
	return &ResponseData{ResponseWriter: w, statusCode: http.StatusOK}
}

func (w *ResponseData) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := WrapWriter(w)
		next.ServeHTTP(wrapped, r)
		log.Println(wrapped.statusCode, r.Method, r.URL.Path, time.Since(start))
	})
}

func IncrementRequestCount(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ensure called to propagate status code throughout the stack
		wrapped := WrapWriter(w)
		next.ServeHTTP(wrapped, r)
		statusCode := wrapped.statusCode
		ctx := context.Background()
		userId, ok := r.Context().Value("userId").(string)
		if !ok {
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		if statusCode == 200 {
			_, err := store.IncrementAPIUsage(ctx, userId)
			if err != nil {
				log.Println("API usage increment error: ", err)
			}
		}
	})
}

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"sub"`
}

func validateClaims(claims *CustomClaims) error {
	now := time.Now()

	// confirm token has not expired
	if !claims.ExpiresAt.Time.After(now) {
		return fmt.Errorf("token has expired")
	}

	if !claims.NotBefore.Time.Before(now) {
		return fmt.Errorf("token is not yet valid")
	}

	// confirm token was issued in the past
	if !claims.IssuedAt.Time.Before(now) {
		return fmt.Errorf("token issue time is in the future")
	}

	// extract a valid userId
	if claims.UserID == "" {
		return fmt.Errorf("user ID (sub) claim is required")
	}

	return nil
}

func BearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorisation header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			http.Error(w, "Bearer token cannot be empty", http.StatusUnauthorized)
			return
		}

		// parse out the token with the custom claim (userid)
		token, err := jwt.ParseWithClaims(
			tokenString,
			&CustomClaims{},
			func(token *jwt.Token) (interface{}, error) {
				// Verify the signing algorithm
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			},
		)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				http.Error(w, "Token has expired", http.StatusUnauthorized)
				return
			}
			log.Printf("Error parsing token: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*CustomClaims)
		if !ok || !token.Valid {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Validate required claims
		if err := validateClaims(claims); err != nil {
			log.Printf("Claims validation failed: %v", err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Set user ID in context
		ctx := context.WithValue(r.Context(), "userId", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimit implements distributed rate limiting using Redis
func RateLimit(rl *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get IP from X-Forwarded-For header, or fallback to RemoteAddr
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.RemoteAddr
			}

			// for X-Forwarded-For, use the first IP in the list
			if strings.Contains(ip, ",") {
				ip = strings.Split(ip, ",")[0]
			}

			allowed, err := rl.isAllowed(r.Context(), ip)
			if err != nil {
				// on error, allow the request but log the issue
				log.Printf("Rate limiting error: %v", err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().
					Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Unix()+int64(rl.windowSeconds)))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type Middleware func(http.Handler) http.Handler

func CreateStack(xs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for ii := len(xs) - 1; ii >= 0; ii-- {
			x := xs[ii]
			next = x(next)
		}
		return next
	}
}
