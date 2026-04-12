package main

import (
	"database/sql"
	"strings"
)

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			last_updated TEXT NOT NULL,
			input_type TEXT NOT NULL DEFAULT 'boolean',
			step_size INTEGER NOT NULL DEFAULT 0,
			step_unit TEXT NOT NULL DEFAULT '',
			display_order INTEGER NOT NULL DEFAULT 99,
			active INTEGER NOT NULL DEFAULT 1,
			target_value INTEGER,
			target_period TEXT,
			range_min INTEGER NOT NULL DEFAULT 1,
			range_max INTEGER NOT NULL DEFAULT 10,
			completed_date TEXT
		);

		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL REFERENCES items(id),
			date TEXT NOT NULL,
			type TEXT,
			note TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS milestones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL REFERENCES items(id),
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
		"ALTER TABLE items ADD COLUMN step_size INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE items ADD COLUMN step_unit TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE items ADD COLUMN display_order INTEGER NOT NULL DEFAULT 99",
		"ALTER TABLE items ADD COLUMN active INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE items ADD COLUMN target_value INTEGER",
		"ALTER TABLE items ADD COLUMN target_period TEXT",
		"ALTER TABLE items ADD COLUMN range_min INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE items ADD COLUMN range_max INTEGER NOT NULL DEFAULT 10",
		"ALTER TABLE items ADD COLUMN completed_date TEXT",
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}
	return nil
}
