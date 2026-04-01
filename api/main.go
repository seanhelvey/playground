package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
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

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/items", handleGetItems)
	mux.HandleFunc("GET /api/items/{name}", handleGetItem)
	mux.HandleFunc("POST /api/items/{name}/log", handleAddLog)
	mux.HandleFunc("PATCH /api/items/{name}", handleUpdateItem)
	mux.HandleFunc("GET /api/checkins", handleGetCheckins)
	mux.HandleFunc("POST /api/checkins", handleAddCheckin)
	mux.HandleFunc("GET /api/wins", handleGetWins)
	mux.HandleFunc("POST /api/wins", handleAddWin)
	mux.HandleFunc("GET /api/tasks", handleGetTasks)
	mux.HandleFunc("POST /api/tasks", handleAddTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", handleDeleteTask)
	mux.HandleFunc("GET /api/engagement", handleEngagement)

	// Serve static files (PWA)
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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
	writeJSON(w, map[string]string{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}
