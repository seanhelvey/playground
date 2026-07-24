package main

import (
	"database/sql"
	"time"
)

// interviewDemoSeedMarker guards resetInterviewDemoItems so it only runs
// once — the Fly machine stops/starts on traffic, so main() runs on every
// cold start, not just once ever.
const interviewDemoSeedMarker = "__interview_demo_seeded_v2__"

// resetInterviewDemoItems hides every existing item and replaces them with a
// small, clean, obviously-good set for the public interview demo. One-off:
// guarded to run exactly once, meant to be deleted from the codebase once
// confirmed live — see CLAUDE.md, data changes are one-off and don't belong
// in permanent code.
func resetInterviewDemoItems(db *sql.DB) error {
	var done int
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE task = ?", interviewDemoSeedMarker).Scan(&done); err != nil {
		return err
	}
	if done > 0 {
		return nil
	}

	if _, err := db.Exec("UPDATE items SET active = 0 WHERE active = 1"); err != nil {
		return err
	}

	var goalsGroupID, morningGroupID, daytimeGroupID sql.NullInt64
	if err := db.QueryRow("SELECT id FROM groups WHERE name = 'Goals'").Scan(&goalsGroupID); err != nil && err != sql.ErrNoRows {
		return err
	}
	if err := db.QueryRow("SELECT id FROM groups WHERE name = 'Morning'").Scan(&morningGroupID); err != nil && err != sql.ErrNoRows {
		return err
	}
	if err := db.QueryRow("SELECT id FROM groups WHERE name = 'Daytime'").Scan(&daytimeGroupID); err != nil && err != sql.ErrNoRows {
		return err
	}

	today := time.Now().Format("2006-01-02")

	inserts := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, range_min, range_max, group_id)
			VALUES (?, ?, 'boolean', 0, '', 1, 1, 1, 10, ?)`, []any{"Wake to alarm", today, morningGroupID}},
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, target_value, target_period, range_min, range_max, group_id)
			VALUES (?, ?, 'counter', 5, 'min', 2, 1, 35, 'weekly', 1, 10, ?)`, []any{"Meditation", today, morningGroupID}},
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, range_min, range_max, group_id)
			VALUES (?, ?, 'boolean', 0, '', 3, 1, 1, 10, ?)`, []any{"Screen time under target", today, daytimeGroupID}},
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, range_min, range_max, group_id)
			VALUES (?, ?, 'boolean', 0, '', 10, 1, 1, 10, ?)`, []any{"Deploy a full-stack project", today, goalsGroupID}},
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, target_value, target_period, range_min, range_max, group_id)
			VALUES (?, ?, 'counter', 1, 'hr', 11, 1, 2, 'monthly', 1, 10, ?)`, []any{"Contribute to open source", today, goalsGroupID}},
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, target_value, target_period, range_min, range_max, group_id)
			VALUES (?, ?, 'counter', 1, 'hr', 12, 1, 2, 'monthly', 1, 10, ?)`, []any{"Build a side income stream", today, goalsGroupID}},
		{`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, active, target_value, target_period, range_min, range_max, group_id)
			VALUES (?, ?, 'counter', 1, 'hr', 13, 1, 2, 'monthly', 1, 10, ?)`, []any{"Save toward a home", today, goalsGroupID}},
	}

	for _, ins := range inserts {
		if _, err := db.Exec(ins.query, ins.args...); err != nil {
			return err
		}
	}

	_, err := db.Exec("INSERT INTO tasks (task, status, created) VALUES (?, 'done', ?)", interviewDemoSeedMarker, today)
	return err
}

// fixInterviewDemoGroups assigns the daily items to the Morning/Daytime
// groups that already existed before resetInterviewDemoItems ran (which
// didn't know about them yet, so seeded those items ungrouped). Naturally
// idempotent — safe to run on every boot.
func fixInterviewDemoGroups(db *sql.DB) error {
	assignments := []struct {
		itemName  string
		groupName string
	}{
		{"Wake to alarm", "Morning"},
		{"Meditation", "Morning"},
		{"Screen time under target", "Daytime"},
	}
	for _, a := range assignments {
		_, err := db.Exec(`UPDATE items SET group_id = (SELECT id FROM groups WHERE name = ?) WHERE name = ? AND active = 1`,
			a.groupName, a.itemName)
		if err != nil {
			return err
		}
	}
	return nil
}
