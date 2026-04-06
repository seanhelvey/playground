package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./playground.db"
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		log.Fatal("migration failed:", err)
	}

	// Seed from data.json if DB is empty
	seedPath := os.Getenv("SEED_PATH")
	if seedPath == "" {
		seedPath = "../data.json"
	}
	if err := seedFromJSON(db, seedPath); err != nil {
		log.Printf("seed: %v (may be fine if already seeded)", err)
	}

	// Rate limit: 10 attempts per minute on auth endpoints
	authLimiter := newRateLimiter(10, time.Minute)

	mux := http.NewServeMux()

	// Public routes (no auth)
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("POST /api/register", authLimiter.middleware(handleRegister))
	mux.HandleFunc("POST /api/login", authLimiter.middleware(handleLogin))
	mux.HandleFunc("POST /api/logout", handleLogout)

	// Protected API routes (auth required)
	api := http.NewServeMux()
	api.HandleFunc("GET /api/me", handleMe)
	api.HandleFunc("GET /api/items", handleGetItems)
	api.HandleFunc("GET /api/items/{name}", handleGetItem)
	api.HandleFunc("POST /api/items/{name}/log", handleAddLog)
	api.HandleFunc("PATCH /api/items/{name}", handleUpdateItem)
	api.HandleFunc("GET /api/checkins", handleGetCheckins)
	api.HandleFunc("POST /api/checkins", handleAddCheckin)
	api.HandleFunc("GET /api/wins", handleGetWins)
	api.HandleFunc("POST /api/wins", handleAddWin)
	api.HandleFunc("GET /api/tasks", handleGetTasks)
	api.HandleFunc("POST /api/tasks", handleAddTask)
	api.HandleFunc("DELETE /api/tasks/{id}", handleDeleteTask)
	api.HandleFunc("GET /api/engagement", handleEngagement)
	mux.Handle("/api/", authMiddleware(api))

	// Serve static files (PWA) — no auth, the app shell loads for everyone
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../static"
	}
	staticDir, _ = filepath.Abs(staticDir)
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sw.js" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fs.ServeHTTP(w, r)
	}))

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigin != "" && origin != "" && origin != allowedOrigin {
			http.Error(w, "forbidden", 403)
			return
		}
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		// If ALLOWED_ORIGIN is not set, allow all — auth middleware handles security
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	sha := os.Getenv("GIT_SHA")
	if sha == "" {
		sha = "dev"
	}
	writeJSON(w, map[string]string{"status": "ok", "sha": sha, "time": time.Now().Format(time.RFC3339)})
}
