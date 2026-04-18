package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Items

type Item struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	LastUpdated   string     `json:"last_updated"`
	InputType     string     `json:"input_type"`
	StepSize      int        `json:"step_size"`
	StepUnit      string     `json:"step_unit"`
	RangeMin      int        `json:"range_min"`
	RangeMax      int        `json:"range_max"`
	TargetValue   *int       `json:"target_value,omitempty"`
	TargetPeriod  *string    `json:"target_period,omitempty"`
	DisplayOrder  int        `json:"display_order"`
	CompletedDate *string    `json:"completed_date,omitempty"`
	GroupID       *int       `json:"group_id,omitempty"`
	Log           []LogEntry `json:"log"`
}

type LogEntry struct {
	ID   int     `json:"id"`
	Date string  `json:"date"`
	Type *string `json:"type,omitempty"`
	Note string  `json:"note"`
}

func handleGetItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, last_updated, input_type, step_size, step_unit, range_min, range_max, target_value, target_period, display_order, completed_date, group_id FROM items WHERE active = 1 ORDER BY display_order, name")
	if err != nil {
		log.Printf("error getting items: %v", err)
		http.Error(w, "internal error", 500)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		rows.Scan(&it.ID, &it.Name, &it.LastUpdated, &it.InputType, &it.StepSize, &it.StepUnit, &it.RangeMin, &it.RangeMax, &it.TargetValue, &it.TargetPeriod, &it.DisplayOrder, &it.CompletedDate, &it.GroupID)

		it.Log = []LogEntry{}
		if logRows, err := db.Query("SELECT id, date, type, note FROM logs WHERE item_id = ? ORDER BY id DESC", it.ID); err == nil {
			for logRows.Next() {
				var l LogEntry
				logRows.Scan(&l.ID, &l.Date, &l.Type, &l.Note)
				it.Log = append(it.Log, l)
			}
			logRows.Close()
		}

		items = append(items, it)
	}
	writeJSON(w, items)
}

func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string  `json:"name"`
		InputType    string  `json:"input_type"`
		StepSize     int     `json:"step_size"`
		StepUnit     string  `json:"step_unit"`
		RangeMin     int     `json:"range_min"`
		RangeMax     int     `json:"range_max"`
		TargetValue  *int    `json:"target_value"`
		TargetPeriod *string `json:"target_period"`
		GroupID      *int    `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", 400)
		return
	}
	if req.InputType == "" {
		req.InputType = "boolean"
	}
	if req.RangeMin == 0 && req.RangeMax == 0 {
		req.RangeMin = 1
		req.RangeMax = 10
	}
	var groupID interface{}
	if req.GroupID != nil && *req.GroupID > 0 {
		groupID = *req.GroupID
	}
	today := time.Now().Format("2006-01-02")
	_, err := db.Exec(
		`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, range_min, range_max, target_value, target_period, active, display_order, group_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 99, ?)`,
		req.Name, today, req.InputType, req.StepSize, req.StepUnit,
		req.RangeMin, req.RangeMax, req.TargetValue, req.TargetPeriod, groupID,
	)
	if err != nil {
		http.Error(w, "db error", 400)
		return
	}
	writeJSON(w, map[string]string{"status": "created"})
}

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var it Item
	err = db.QueryRow("SELECT id, name, last_updated, input_type, step_size, step_unit, range_min, range_max, target_value, target_period, display_order, completed_date, group_id FROM items WHERE id = ? AND active = 1", id).
		Scan(&it.ID, &it.Name, &it.LastUpdated, &it.InputType, &it.StepSize, &it.StepUnit, &it.RangeMin, &it.RangeMax, &it.TargetValue, &it.TargetPeriod, &it.DisplayOrder, &it.CompletedDate, &it.GroupID)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	it.Log = []LogEntry{}
	if logRows, err := db.Query("SELECT id, date, type, note FROM logs WHERE item_id = ? ORDER BY id DESC", id); err == nil {
		for logRows.Next() {
			var l LogEntry
			logRows.Scan(&l.ID, &l.Date, &l.Type, &l.Note)
			it.Log = append(it.Log, l)
		}
		logRows.Close()
	}
	writeJSON(w, it)
}

func handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var update struct {
		Name          *string `json:"name"`
		InputType     *string `json:"input_type"`
		StepSize      *int    `json:"step_size"`
		StepUnit      *string `json:"step_unit"`
		CompletedDate *string `json:"completed_date"`
		Active        *int    `json:"active"`
		TargetValue   *int    `json:"target_value"`
		TargetPeriod  *string `json:"target_period"`
		DisplayOrder  *int    `json:"display_order"`
		GroupID       *int    `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	// Read current values to detect changes for logging.
	var curName, curInputType string
	var curGroupID *int
	db.QueryRow("SELECT name, input_type, group_id FROM items WHERE id = ?", id).Scan(&curName, &curInputType, &curGroupID)

	today := time.Now().Format("2006-01-02")

	exec := func(query string, args ...any) bool {
		if _, err := db.Exec(query, args...); err != nil {
			log.Printf("handleUpdateItem: %v", err)
			http.Error(w, "db error", 500)
			return false
		}
		return true
	}

	if update.Name != nil {
		if !exec("UPDATE items SET name = ?, last_updated = ? WHERE id = ?", *update.Name, today, id) {
			return
		}
	}
	if update.InputType != nil {
		if !exec("UPDATE items SET input_type = ?, last_updated = ? WHERE id = ?", *update.InputType, today, id) {
			return
		}
	}
	if update.StepSize != nil {
		if !exec("UPDATE items SET step_size = ?, last_updated = ? WHERE id = ?", *update.StepSize, today, id) {
			return
		}
	}
	if update.StepUnit != nil {
		if !exec("UPDATE items SET step_unit = ?, last_updated = ? WHERE id = ?", *update.StepUnit, today, id) {
			return
		}
	}
	if update.CompletedDate != nil {
		if !exec("UPDATE items SET completed_date = ?, last_updated = ? WHERE id = ?", *update.CompletedDate, today, id) {
			return
		}
	}
	if update.Active != nil {
		if !exec("UPDATE items SET active = ?, last_updated = ? WHERE id = ?", *update.Active, today, id) {
			return
		}
	}
	if update.TargetValue != nil {
		if !exec("UPDATE items SET target_value = ?, last_updated = ? WHERE id = ?", *update.TargetValue, today, id) {
			return
		}
	}
	if update.TargetPeriod != nil {
		if !exec("UPDATE items SET target_period = ?, last_updated = ? WHERE id = ?", *update.TargetPeriod, today, id) {
			return
		}
	}
	if update.DisplayOrder != nil {
		if !exec("UPDATE items SET display_order = ? WHERE id = ?", *update.DisplayOrder, id) {
			return
		}
	}
	if update.GroupID != nil {
		if *update.GroupID == 0 {
			if !exec("UPDATE items SET group_id = NULL WHERE id = ?", id) {
				return
			}
		} else {
			if !exec("UPDATE items SET group_id = ? WHERE id = ?", *update.GroupID, id) {
				return
			}
		}
	}

	// Log notable config changes.
	var parts []string
	if update.Name != nil && *update.Name != curName {
		parts = append(parts, fmt.Sprintf("name: %s→%s", curName, *update.Name))
	}
	if update.InputType != nil && *update.InputType != curInputType {
		parts = append(parts, fmt.Sprintf("input_type: %s→%s", curInputType, *update.InputType))
	}
	if update.Active != nil && *update.Active == 0 {
		parts = append(parts, "removed")
	}
	if update.GroupID != nil {
		oldID := 0
		if curGroupID != nil {
			oldID = *curGroupID
		}
		if oldID != *update.GroupID {
			oldName := "(none)"
			if curGroupID != nil {
				db.QueryRow("SELECT name FROM groups WHERE id = ?", *curGroupID).Scan(&oldName)
			}
			newName := "(none)"
			if *update.GroupID > 0 {
				db.QueryRow("SELECT name FROM groups WHERE id = ?", *update.GroupID).Scan(&newName)
			}
			parts = append(parts, fmt.Sprintf("group: %s→%s", oldName, newName))
		}
	}
	if len(parts) > 0 {
		db.Exec("INSERT INTO logs (item_id, date, type, note) VALUES (?, ?, 'config', ?)", id, today, strings.Join(parts, ", "))
	}

	writeJSON(w, map[string]string{"status": "updated"})
}

func handleAddLog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var entry struct {
		Note string  `json:"note"`
		Type *string `json:"type,omitempty"`
		Date string  `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	date := entry.Date
	if len(date) != 10 || date[4] != '-' || date[7] != '-' {
		date = time.Now().Format("2006-01-02")
	}
	db.Exec("INSERT INTO logs (item_id, date, type, note) VALUES (?, ?, ?, ?)", id, date, entry.Type, entry.Note)
	db.Exec("UPDATE items SET last_updated = ? WHERE id = ?", date, id)
	writeJSON(w, map[string]string{"status": "logged"})
}

// Check-ins

type CheckIn struct {
	ID      int     `json:"id"`
	Date    string  `json:"date"`
	Body    *int    `json:"body"`
	Mind    *int    `json:"mind"`
	Social  *int    `json:"social"`
	Feeling *string `json:"feeling"`
	MoreOf  *string `json:"more_of"`
	LessOf  *string `json:"less_of"`
}

func handleGetCheckins(w http.ResponseWriter, r *http.Request) {
	rows, _ := db.Query("SELECT id, date, body, mind, social, feeling, more_of, less_of FROM check_ins ORDER BY date DESC, id DESC LIMIT 12")
	defer rows.Close()
	checkins := []CheckIn{}
	for rows.Next() {
		var c CheckIn
		rows.Scan(&c.ID, &c.Date, &c.Body, &c.Mind, &c.Social, &c.Feeling, &c.MoreOf, &c.LessOf)
		checkins = append(checkins, c)
	}
	writeJSON(w, checkins)
}

func handleAddCheckin(w http.ResponseWriter, r *http.Request) {
	var c CheckIn
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if len(c.Date) != 10 || c.Date[4] != '-' || c.Date[7] != '-' {
		c.Date = time.Now().Format("2006-01-02")
	}
	db.Exec("INSERT INTO check_ins (date, body, mind, social, feeling, more_of, less_of) VALUES (?, ?, ?, ?, ?, ?, ?)",
		c.Date, c.Body, c.Mind, c.Social, c.Feeling, c.MoreOf, c.LessOf)
	writeJSON(w, map[string]string{"status": "saved"})
}

// Tasks

type Task struct {
	ID      int    `json:"id"`
	TaskStr string `json:"task"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

func handleGetTasks(w http.ResponseWriter, r *http.Request) {
	rows, _ := db.Query("SELECT id, task, status, created FROM tasks ORDER BY created DESC")
	defer rows.Close()
	tasks := []Task{}
	for rows.Next() {
		var t Task
		rows.Scan(&t.ID, &t.TaskStr, &t.Status, &t.Created)
		tasks = append(tasks, t)
	}
	writeJSON(w, tasks)
}

func handleAddTask(w http.ResponseWriter, r *http.Request) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	t.Created = time.Now().Format("2006-01-02")
	db.Exec("INSERT INTO tasks (task, status, created) VALUES (?, 'pending', ?)", t.TaskStr, t.Created)
	writeJSON(w, map[string]string{"status": "created"})
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	db.Exec("DELETE FROM tasks WHERE id = ?", id)
	writeJSON(w, map[string]string{"status": "deleted"})
}

// Logs feed

type LogFeedEntry struct {
	ID       int     `json:"id"`
	Date     string  `json:"date"`
	ItemName string  `json:"item_name"`
	Type     *string `json:"type,omitempty"`
	Note     string  `json:"note"`
}

// Groups

type Group struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	DisplayOrder int    `json:"display_order"`
}

func handleGetGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, display_order FROM groups ORDER BY display_order, name")
	if err != nil {
		log.Printf("error getting groups: %v", err)
		http.Error(w, "internal error", 500)
		return
	}
	defer rows.Close()
	groups := []Group{}
	for rows.Next() {
		var g Group
		rows.Scan(&g.ID, &g.Name, &g.DisplayOrder)
		groups = append(groups, g)
	}
	writeJSON(w, groups)
}

func handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", 400)
		return
	}
	result, err := db.Exec("INSERT INTO groups (name, display_order) VALUES (?, 99)", req.Name)
	if err != nil {
		log.Printf("error creating group: %v", err)
		http.Error(w, "db error", 500)
		return
	}
	newID, _ := result.LastInsertId()
	today := time.Now().Format("2006-01-02")
	db.Exec("INSERT INTO logs (item_id, group_id, date, type, note) VALUES (0, ?, ?, 'config', ?)",
		newID, today, "created: "+req.Name)
	writeJSON(w, map[string]string{"status": "created"})
}

func handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var req struct {
		Name         *string `json:"name"`
		DisplayOrder *int    `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if req.Name != nil {
		var curName string
		db.QueryRow("SELECT name FROM groups WHERE id = ?", id).Scan(&curName)
		if _, err := db.Exec("UPDATE groups SET name = ? WHERE id = ?", *req.Name, id); err != nil {
			log.Printf("error updating group: %v", err)
			http.Error(w, "db error", 500)
			return
		}
		if *req.Name != curName {
			today := time.Now().Format("2006-01-02")
			db.Exec("INSERT INTO logs (item_id, group_id, date, type, note) VALUES (0, ?, ?, 'config', ?)",
				id, today, fmt.Sprintf("name: %s→%s", curName, *req.Name))
		}
	}
	if req.DisplayOrder != nil {
		if _, err := db.Exec("UPDATE groups SET display_order = ? WHERE id = ?", *req.DisplayOrder, id); err != nil {
			log.Printf("error updating group order: %v", err)
			http.Error(w, "db error", 500)
			return
		}
	}
	writeJSON(w, map[string]string{"status": "updated"})
}

func handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var groupName string
	db.QueryRow("SELECT name FROM groups WHERE id = ?", id).Scan(&groupName)
	if _, err := db.Exec("UPDATE items SET group_id = NULL WHERE group_id = ?", id); err != nil {
		log.Printf("error ungrouping items: %v", err)
		http.Error(w, "db error", 500)
		return
	}
	if _, err := db.Exec("DELETE FROM groups WHERE id = ?", id); err != nil {
		log.Printf("error deleting group: %v", err)
		http.Error(w, "db error", 500)
		return
	}
	today := time.Now().Format("2006-01-02")
	db.Exec("INSERT INTO logs (item_id, group_id, date, type, note) VALUES (0, 0, ?, 'config', ?)",
		today, "deleted: "+groupName)
	writeJSON(w, map[string]string{"status": "deleted"})
}

func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT l.id, l.date,
			CASE WHEN l.item_id > 0 THEN i.name
			     WHEN l.group_id > 0 THEN g.name
			     ELSE 'Groups'
			END,
			l.type, l.note
		FROM logs l
		LEFT JOIN items i ON l.item_id = i.id AND l.item_id > 0
		LEFT JOIN groups g ON l.group_id = g.id AND l.group_id > 0
		ORDER BY l.id DESC
		LIMIT 200
	`)
	if err != nil {
		log.Printf("error getting logs: %v", err)
		http.Error(w, "internal error", 500)
		return
	}
	defer rows.Close()
	entries := []LogFeedEntry{}
	for rows.Next() {
		var e LogFeedEntry
		rows.Scan(&e.ID, &e.Date, &e.ItemName, &e.Type, &e.Note)
		entries = append(entries, e)
	}
	writeJSON(w, entries)
}
