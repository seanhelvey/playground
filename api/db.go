package main

import (
	"database/sql"
	"strings"
)

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			name TEXT PRIMARY KEY,
			type TEXT NOT NULL DEFAULT 'Habit',
			momentum TEXT NOT NULL DEFAULT 'dormant',
			focus TEXT,
			next TEXT,
			url TEXT,
			target_date TEXT,
			success_criteria TEXT,
			last_updated TEXT NOT NULL,
			input_type TEXT NOT NULL DEFAULT 'boolean',
			cadence TEXT NOT NULL DEFAULT 'daily',
			step_size INTEGER NOT NULL DEFAULT 0,
			step_unit TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_name TEXT NOT NULL REFERENCES items(name),
			date TEXT NOT NULL,
			type TEXT,
			note TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS milestones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_name TEXT NOT NULL REFERENCES items(name),
			date TEXT NOT NULL,
			label TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS check_ins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			body INTEGER,
			mind INTEGER,
			social INTEGER,
			feeling TEXT,
			more_of TEXT,
			less_of TEXT
		);

		CREATE TABLE IF NOT EXISTS wins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			note TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			created TEXT NOT NULL,
			expires TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS push_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			endpoint TEXT NOT NULL UNIQUE,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL,
			created TEXT NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	return migrateAlter(db)
}

// migrateAlter adds columns to existing tables idempotently.
func migrateAlter(db *sql.DB) error {
	alters := []string{
		"ALTER TABLE items ADD COLUMN input_type TEXT NOT NULL DEFAULT 'boolean'",
		"ALTER TABLE items ADD COLUMN cadence TEXT NOT NULL DEFAULT 'daily'",
		"ALTER TABLE items ADD COLUMN step_size INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE items ADD COLUMN step_unit TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE items ADD COLUMN display_order INTEGER NOT NULL DEFAULT 99",
		"ALTER TABLE items ADD COLUMN active INTEGER NOT NULL DEFAULT 1",
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}

	// Set correct input_type, cadence, step_size, step_unit for known items.
	// Safe to re-run — only updates rows that still have the default value.
	type itemConfig struct {
		name, inputType, cadence, stepUnit string
		stepSize, order                    int
	}
	updates := []itemConfig{
		{"Wake to alarm", "boolean", "daily", "", 0, 1},
		{"Meditation", "counter", "daily", "min", 5, 2},
		{"DM a friend", "boolean", "daily", "", 0, 3},
		{"Fast after dinner", "boolean", "daily", "", 0, 4},
		{"Screen time", "boolean", "daily", "", 0, 5},
		{"Dancing", "counter", "weekly", "min", 15, 6},
		{"Music", "counter", "weekly", "min", 15, 7},
		{"Nature", "note", "ongoing", "", 0, 8},
		{"Own a home", "note", "monthly", "", 0, 10},
		{"Build fallback income", "note", "monthly", "", 0, 11},
		{"Deploy a full-stack project", "note", "ongoing", "", 0, 12},
		{"Contribute to non-Django OSS", "note", "ongoing", "", 0, 13},
		{"Stick with the process", "boolean", "daily", "", 0, 99},
	}
	for _, u := range updates {
		db.Exec(
			"UPDATE items SET input_type=?, cadence=?, step_size=?, step_unit=?, display_order=? WHERE name=?",
			u.inputType, u.cadence, u.stepSize, u.stepUnit, u.order, u.name,
		)
	}
	db.Exec("UPDATE items SET active = 0 WHERE name = 'Coloft'")
	return nil
}
