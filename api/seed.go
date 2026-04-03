package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type SeedData struct {
	Items    []SeedItem    `json:"items"`
	Wins     []SeedWin     `json:"wins"`
	CheckIns []SeedCheckin `json:"check_ins"`
	Tasks    []SeedTask    `json:"tasks"`
}

type SeedTask struct {
	Task    string `json:"task"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

type SeedItem struct {
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Momentum        string          `json:"momentum"`
	Focus           string          `json:"focus"`
	Next            *string         `json:"next"`
	URL             *string         `json:"url"`
	TargetDate      *string         `json:"target_date"`
	SuccessCriteria *string         `json:"success_criteria"`
	LastUpdated     string          `json:"last_updated"`
	Milestones      []SeedMilestone `json:"milestones"`
	Log             []SeedLog       `json:"log"`
}

type SeedLog struct {
	Date string  `json:"date"`
	Type *string `json:"type"`
	Note string  `json:"note"`
}

type SeedMilestone struct {
	Date  string `json:"date"`
	Label string `json:"label"`
}

type SeedWin struct {
	Date string `json:"date"`
	Note string `json:"note"`
}

type SeedCheckin struct {
	Date    string  `json:"date"`
	Body    *int    `json:"body"`
	Mind    *int    `json:"mind"`
	Social  *int    `json:"social"`
	Feeling *string `json:"feeling"`
	MoreOf  *string `json:"more_of"`
	LessOf  *string `json:"less_of"`
}

func seedFromJSON(db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var seed SeedData
	if err := json.Unmarshal(data, &seed); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	// Also try to load tasks from tasks.json alongside data.json
	tasksPath := path[:len(path)-len("data.json")] + "tasks.json"
	if tasksData, err := os.ReadFile(tasksPath); err == nil {
		var taskFile struct {
			Tasks []SeedTask `json:"tasks"`
		}
		if json.Unmarshal(tasksData, &taskFile) == nil {
			seed.Tasks = taskFile.Tasks
		}
	}

	// Check if items already seeded
	var itemCount int
	db.QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)

	// Sync tasks from tasks.json every startup.
	// Claude Code adds tasks to tasks.json via git. On deploy, new tasks
	// get inserted here. Matches on task text to avoid duplicates.
	for _, t := range seed.Tasks {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM tasks WHERE task = ?", t.Task).Scan(&exists)
		if exists == 0 {
			db.Exec("INSERT INTO tasks (task, status, created) VALUES (?, ?, ?)", t.Task, t.Status, t.Created)
		}
	}

	if itemCount > 0 {
		return nil // items already seeded
	}

	tx, _ := db.Begin()
	defer tx.Rollback()

	for _, item := range seed.Items {
		tx.Exec("INSERT INTO items (name, type, momentum, focus, next, url, target_date, success_criteria, last_updated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			item.Name, item.Type, item.Momentum, item.Focus, item.Next, item.URL, item.TargetDate, item.SuccessCriteria, item.LastUpdated)

		for _, l := range item.Log {
			tx.Exec("INSERT INTO logs (item_name, date, type, note) VALUES (?, ?, ?, ?)", item.Name, l.Date, l.Type, l.Note)
		}
		for _, m := range item.Milestones {
			tx.Exec("INSERT INTO milestones (item_name, date, label) VALUES (?, ?, ?)", item.Name, m.Date, m.Label)
		}
	}

	for _, w := range seed.Wins {
		tx.Exec("INSERT INTO wins (date, note) VALUES (?, ?)", w.Date, w.Note)
	}

	for _, c := range seed.CheckIns {
		tx.Exec("INSERT INTO check_ins (date, body, mind, social, feeling, more_of, less_of) VALUES (?, ?, ?, ?, ?, ?, ?)",
			c.Date, c.Body, c.Mind, c.Social, c.Feeling, c.MoreOf, c.LessOf)
	}

	for _, t := range seed.Tasks {
		tx.Exec("INSERT INTO tasks (task, status, created) VALUES (?, ?, ?)", t.Task, t.Status, t.Created)
	}

	return tx.Commit()
}
