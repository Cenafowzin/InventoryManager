package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/rubendubeux/inventory-manager/internal/auth"
	"github.com/rubendubeux/inventory-manager/internal/db"
	"github.com/rubendubeux/inventory-manager/pkg/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading environment variables directly")
	}

	databaseURL := mustEnv("DATABASE_URL")
	port := getEnv("PORT", "8080")
	jwtSecret := mustEnv("JWT_SECRET")
	accessTTL := parseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	refreshTTL := parseDuration(getEnv("JWT_REFRESH_TTL", "168h"))

	pool, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(databaseURL); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	// Auth
	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, auth.Config{
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	})
	authHandler := auth.NewHandler(authSvc)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/health", healthHandler(pool))

	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/refresh", authHandler.Refresh)

	// Grupo protegido — próximas fases adicionam rotas aqui
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(jwtSecret))
		// rotas protegidas
	})

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "db": "unreachable"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "connected"})
	}
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return val
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("invalid duration %q: %v", s, err)
	}
	return d
}
