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

// migrateAlter adds columns and tables to existing DBs idempotently.
func migrateAlter(db *sql.DB) error {
	alters := []string{
		"ALTER TABLE items ADD COLUMN input_type TEXT NOT NULL DEFAULT 'boolean'",
		"ALTER TABLE items ADD COLUMN cadence TEXT NOT NULL DEFAULT 'daily'",
		"ALTER TABLE items ADD COLUMN step_size INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE items ADD COLUMN step_unit TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE items ADD COLUMN display_order INTEGER NOT NULL DEFAULT 99",
		"ALTER TABLE items ADD COLUMN active INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE items ADD COLUMN target_value INTEGER",
		"ALTER TABLE items ADD COLUMN target_period TEXT",
		"ALTER TABLE items ADD COLUMN range_min INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE items ADD COLUMN range_max INTEGER NOT NULL DEFAULT 10",
		"ALTER TABLE items ADD COLUMN tags TEXT",
	}
	for _, stmt := range alters {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}

	// item_relationships: m2m between items (e.g. habit → goal)
	db.Exec(`CREATE TABLE IF NOT EXISTS item_relationships (
		parent_name TEXT NOT NULL REFERENCES items(name),
		child_name  TEXT NOT NULL REFERENCES items(name),
		PRIMARY KEY (parent_name, child_name)
	)`)

	// Deactivate removed items
	db.Exec("UPDATE items SET active = 0 WHERE name IN ('Coloft', 'Nature', 'Screen time')")

	// Insert new items (idempotent)
	type newItem struct {
		name, inputType, cadence, stepUnit, tags string
		stepSize, order, rangeMin, rangeMax       int
	}
	inserts := []newItem{
		{"No work after dinner", "boolean", "daily", "", "", 0, 5, 1, 10},
		{"App time under target", "boolean", "daily", "", "", 0, 6, 1, 10},
		{"Live closer to nature", "goal", "ongoing", "", `["nature"]`, 0, 7, 1, 10},
		{"Plant ID", "counter", "daily", "species", `["nature"]`, 1, 8, 1, 10},
		{"Gardening", "counter", "weekly", "min", `["nature"]`, 30, 9, 1, 10},
		{"Fishing", "boolean", "weekly", "", `["nature"]`, 0, 10, 1, 10},
		{"Body", "slider", "daily", "", `["health"]`, 1, 17, 1, 10},
		{"Mind", "slider", "daily", "", `["health"]`, 1, 18, 1, 10},
		{"Social", "slider", "daily", "", `["social"]`, 1, 19, 1, 10},
	}
	today := "2026-04-07"
	for _, it := range inserts {
		db.Exec(
			`INSERT OR IGNORE INTO items (name, momentum, last_updated, input_type, cadence, step_size, step_unit, display_order, range_min, range_max, tags, active)
			 VALUES (?, 'dormant', ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
			it.name, today, it.inputType, it.cadence, it.stepSize, it.stepUnit, it.order, it.rangeMin, it.rangeMax, it.tags,
		)
	}

	// Update config for existing items
	type itemConfig struct {
		name, inputType, cadence, stepUnit string
		stepSize, order                    int
	}
	updates := []itemConfig{
		{"Wake to alarm", "boolean", "daily", "", 0, 1},
		{"Meditation", "counter", "daily", "min", 5, 2},
		{"DM a friend", "boolean", "daily", "", 0, 3},
		{"Fast after dinner", "boolean", "daily", "", 0, 4},
		{"Dancing", "counter", "weekly", "min", 15, 11},
		{"Music", "counter", "weekly", "min", 15, 12},
		{"Own a home", "goal", "monthly", "", 0, 13},
		{"Build fallback income", "goal", "monthly", "", 0, 14},
		{"Deploy a full-stack project", "goal", "ongoing", "", 0, 15},
		{"Contribute to non-Django OSS", "goal", "ongoing", "", 0, 16},
		{"Stick with the process", "boolean", "daily", "", 0, 99},
	}
	for _, u := range updates {
		db.Exec(
			"UPDATE items SET input_type=?, cadence=?, step_size=?, step_unit=?, display_order=? WHERE name=?",
			u.inputType, u.cadence, u.stepSize, u.stepUnit, u.order, u.name,
		)
	}

	// Relationships: nature habits → Live closer to nature
	rels := [][2]string{
		{"Live closer to nature", "Plant ID"},
		{"Live closer to nature", "Gardening"},
		{"Live closer to nature", "Fishing"},
	}
	for _, r := range rels {
		db.Exec("INSERT OR IGNORE INTO item_relationships (parent_name, child_name) VALUES (?, ?)", r[0], r[1])
	}

	return nil
}
