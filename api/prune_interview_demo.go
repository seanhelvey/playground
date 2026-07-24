package main

import "database/sql"

// pruneInterviewDemoContent hides items whose names read as personal/health
// content before the account is shown publicly as an interview demo.
// One-off: idempotent (safe to run on every boot), meant to be deleted from
// the codebase once confirmed live — see CLAUDE.md, data changes are one-off
// and don't belong in permanent code.
func pruneInterviewDemoContent(db *sql.DB) error {
	patterns := []string{
		"Commit to liberation from a%",
		"Read TMS reminders%",
		"Two hours between 3 meals%",
	}
	for _, p := range patterns {
		if _, err := db.Exec("UPDATE items SET active = 0 WHERE name LIKE ?", p); err != nil {
			return err
		}
	}
	return nil
}
