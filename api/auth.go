package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "session"
	sessionDuration   = 30 * 24 * time.Hour // 30 days

	demoEmail    = "demo@playground.app"
	demoPassword = "demo1234"
)

// ensureDemoUser creates the shared read-only demo account on first boot.
// Idempotent — does nothing if a demo user already exists.
func ensureDemoUser(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE is_demo = 1").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec("INSERT INTO users (email, password_hash, created, is_demo) VALUES (?, ?, ?, 1)",
		demoEmail, string(hash), now)
	return err
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	// Registration is open but gated: only allowed if no users exist yet,
	// OR if INVITE_CODE env var is set and provided in the request.
	// This means the first user can always register, subsequent users need an invite.
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)

	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	if userCount > 0 {
		inviteCode := os.Getenv("INVITE_CODE")
		if inviteCode == "" || req.InviteCode != inviteCode {
			http.Error(w, "registration requires an invite code", 403)
			return
		}
	}

	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "email required, password must be 8+ characters", 400)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}

	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec("INSERT INTO users (email, password_hash, created) VALUES (?, ?, ?)", req.Email, string(hash), now)
	if err != nil {
		http.Error(w, "email already registered", 409)
		return
	}

	userID, _ := result.LastInsertId()
	setSession(w, userID)
	writeJSON(w, map[string]string{"status": "registered"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	var userID int64
	var passwordHash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE email = ?", req.Email).Scan(&userID, &passwordHash)
	if err != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}

	setSession(w, userID)
	writeJSON(w, map[string]string{"status": "logged in"})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("FLY_APP_NAME") != "",
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, map[string]string{"status": "logged out"})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey)
	if userID == nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	var email string
	var isDemo bool
	db.QueryRow("SELECT email, is_demo FROM users WHERE id = ?", userID).Scan(&email, &isDemo)
	writeJSON(w, map[string]any{"email": email, "is_demo": isDemo})
}

func setSession(w http.ResponseWriter, userID int64) {
	sessionID := generateSessionID()
	now := time.Now()
	expires := now.Add(sessionDuration)

	db.Exec("INSERT INTO sessions (id, user_id, created, expires) VALUES (?, ?, ?, ?)",
		sessionID, userID, now.Format(time.RFC3339), expires.Format(time.RFC3339))

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   os.Getenv("FLY_APP_NAME") != "",
		SameSite: http.SameSiteLaxMode,
	})
}

type contextKey string

const userIDKey contextKey = "userID"

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Error(w, "unauthorized", 401)
			return
		}

		var userID int64
		var expires string
		err = db.QueryRow("SELECT user_id, expires FROM sessions WHERE id = ?", cookie.Value).Scan(&userID, &expires)
		if err != nil {
			http.Error(w, "unauthorized", 401)
			return
		}

		expiresTime, _ := time.Parse(time.RFC3339, expires)
		if time.Now().After(expiresTime) {
			db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
			http.Error(w, "session expired", 401)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// demoReadOnlyMiddleware blocks the shared demo account from any write
// request, so a public demo link can never mutate or spam real data.
// Must run after authMiddleware so userIDKey is already in context.
func demoReadOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		userID := r.Context().Value(userIDKey)
		var isDemo bool
		db.QueryRow("SELECT is_demo FROM users WHERE id = ?", userID).Scan(&isDemo)
		if isDemo {
			http.Error(w, "demo account is read-only", 403)
			return
		}

		next.ServeHTTP(w, r)
	})
}
