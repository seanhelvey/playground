package main

import (
	"database/sql"
	"fmt"
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

// migrateAlter adds columns and restructures tables idempotently.
func migrateAlter(db *sql.DB) error {
	// Migrate items from name TEXT PRIMARY KEY → id INTEGER PRIMARY KEY AUTOINCREMENT
	// Detect by checking whether the id column exists.
	var hasID int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('items') WHERE name='id'").Scan(&hasID)
	if hasID == 0 {
		if err := migratePKToID(db); err != nil {
			return fmt.Errorf("pk migration: %w", err)
		}
	}

	// Add any missing columns to existing items tables (for older installs between migrations)
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

	// Remove any duplicates created before name-uniqueness was enforced (keep lowest id per name)
	db.Exec(`DELETE FROM items WHERE id NOT IN (SELECT MIN(id) FROM items GROUP BY name)`)

	// Insert new items — check by name since name is not a unique constraint
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
			`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, range_min, range_max, target_value, target_period, active)
			 SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1 WHERE NOT EXISTS (SELECT 1 FROM items WHERE name = ?)`,
			it.name, today, it.inputType, it.stepSize, it.stepUnit, it.order, it.rangeMin, it.rangeMax, it.targetValue, it.targetPeriod, it.name,
		)
	}

	return nil
}

// migratePKToID rebuilds items/logs/milestones to use surrogate integer PKs.
func migratePKToID(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmts := []string{
		`CREATE TABLE items_new (
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
		)`,
		`INSERT INTO items_new (name, last_updated, input_type, step_size, step_unit, display_order, active, target_value, target_period, range_min, range_max, completed_date)
			SELECT name, last_updated, input_type, step_size, step_unit, display_order, active, target_value, target_period, range_min, range_max, completed_date FROM items`,
		`DROP TABLE items`,
		`ALTER TABLE items_new RENAME TO items`,
		`CREATE TABLE logs_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL REFERENCES items(id),
			date TEXT NOT NULL,
			type TEXT,
			note TEXT NOT NULL
		)`,
		`INSERT INTO logs_new (id, item_id, date, type, note)
			SELECT l.id, i.id, l.date, l.type, l.note FROM logs l JOIN items i ON l.item_name = i.name`,
		`DROP TABLE logs`,
		`ALTER TABLE logs_new RENAME TO logs`,
		`CREATE TABLE milestones_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL REFERENCES items(id),
			date TEXT NOT NULL,
			label TEXT NOT NULL
		)`,
		`INSERT INTO milestones_new (id, item_id, date, label)
			SELECT m.id, i.id, m.date, m.label FROM milestones m JOIN items i ON m.item_name = i.name`,
		`DROP TABLE milestones`,
		`ALTER TABLE milestones_new RENAME TO milestones`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
