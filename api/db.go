package main

import (
	"database/sql"
	"strings"
)

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			name TEXT PRIMARY KEY,
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

// migrateAlter adds columns and tables to existing DBs idempotently.
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

	// Deactivate removed items
	db.Exec("UPDATE items SET active = 0 WHERE name IN ('Coloft', 'Nature', 'Screen time', 'Live closer to nature', 'Stick with the process')")
	// Mark completed items
	db.Exec("UPDATE items SET completed_date = '2026-04-07' WHERE name = 'Deploy a full-stack project' AND completed_date IS NULL")

	// Insert new items (idempotent)
	type newItem struct {
		name, inputType, stepUnit string
		stepSize, order, rangeMin, rangeMax int
		targetValue                         int
		targetPeriod                        string
	}
	today := "2026-04-07"
	inserts := []newItem{
		{"No work after dinner", "boolean", "", 0, 5, 1, 10, 1, "daily"},
		{"App time under target", "boolean", "", 0, 6, 1, 10, 1, "daily"},
		{"Plant ID", "counter", "species", 1, 8, 1, 10, 1, "daily"},
		{"Gardening", "counter", "min", 30, 9, 1, 10, 120, "monthly"},
		{"Fishing", "counter", "min", 30, 10, 1, 10, 120, "monthly"},
		{"Body", "slider", "", 1, 17, 1, 10, 0, ""},
		{"Mind", "slider", "", 1, 18, 1, 10, 0, ""},
		{"Social", "slider", "", 1, 19, 1, 10, 0, ""},
	}
	for _, it := range inserts {
		db.Exec(
			`INSERT OR IGNORE INTO items (name, last_updated, input_type, step_size, step_unit, display_order, range_min, range_max, target_value, target_period, active)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
			it.name, today, it.inputType, it.stepSize, it.stepUnit, it.order, it.rangeMin, it.rangeMax, it.targetValue, it.targetPeriod,
		)
	}

	// Update config for existing items
	type itemConfig struct {
		name, inputType, stepUnit, targetPeriod string
		stepSize, order, targetValue            int
	}
	updates := []itemConfig{
		{"Wake to alarm", "boolean", "", "daily", 0, 1, 1},
		{"Meditation", "counter", "min", "weekly", 5, 2, 35},
		{"DM a friend", "boolean", "", "daily", 0, 3, 1},
		{"Fast after dinner", "boolean", "", "daily", 0, 4, 1},
		{"Dancing", "counter", "min", "weekly", 15, 11, 120},
		{"Music", "counter", "min", "weekly", 15, 12, 120},
		{"Fishing", "counter", "min", "monthly", 30, 10, 120},
		{"Gardening", "counter", "min", "monthly", 30, 9, 120},
		{"Plant ID", "counter", "species", "daily", 1, 8, 1},
		{"Own a home", "counter", "hr", "monthly", 1, 13, 2},
		{"Build fallback income", "counter", "hr", "monthly", 1, 14, 2},
		{"Contribute to non-Django OSS", "counter", "hr", "monthly", 1, 16, 2},
	}
	for _, u := range updates {
		db.Exec(
			"UPDATE items SET input_type=?, step_size=?, step_unit=?, display_order=?, target_value=?, target_period=? WHERE name=?",
			u.inputType, u.stepSize, u.stepUnit, u.order, u.targetValue, u.targetPeriod, u.name,
		)
	}

	return nil
}
