package main

import "database/sql"

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
			last_updated TEXT NOT NULL
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
	return err
}
