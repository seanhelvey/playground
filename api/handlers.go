package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Items

type Item struct {
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
	Log           []LogEntry `json:"log"`
}

type LogEntry struct {
	ID       int     `json:"id"`
	ItemName string  `json:"item_name,omitempty"`
	Date     string  `json:"date"`
	Type     *string `json:"type,omitempty"`
	Note     string  `json:"note"`
}

func handleGetItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT name, last_updated, input_type, step_size, step_unit, range_min, range_max, target_value, target_period, display_order, completed_date FROM items WHERE active = 1 ORDER BY display_order, name")
	if err != nil {
		log.Printf("error getting items: %v", err)
		http.Error(w, "internal error", 500)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		rows.Scan(&it.Name, &it.LastUpdated, &it.InputType, &it.StepSize, &it.StepUnit, &it.RangeMin, &it.RangeMax, &it.TargetValue, &it.TargetPeriod, &it.DisplayOrder, &it.CompletedDate)

		it.Log = []LogEntry{}
		if logRows, err := db.Query("SELECT id, date, type, note FROM logs WHERE item_name = ? ORDER BY id DESC", it.Name); err == nil {
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
	today := time.Now().Format("2006-01-02")
	_, err := db.Exec(
		`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, range_min, range_max, target_value, target_period, active, display_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 99)`,
		req.Name, today, req.InputType, req.StepSize, req.StepUnit,
		req.RangeMin, req.RangeMax, req.TargetValue, req.TargetPeriod,
	)
	if err != nil {
		http.Error(w, "item already exists or db error", 400)
		return
	}
	writeJSON(w, map[string]string{"status": "created"})
}

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var it Item
	err := db.QueryRow("SELECT name, last_updated, input_type, step_size, step_unit, range_min, range_max, target_value, target_period, display_order, completed_date FROM items WHERE name = ? AND active = 1", name).
		Scan(&it.Name, &it.LastUpdated, &it.InputType, &it.StepSize, &it.StepUnit, &it.RangeMin, &it.RangeMax, &it.TargetValue, &it.TargetPeriod, &it.DisplayOrder, &it.CompletedDate)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	it.Log = []LogEntry{}
	if logRows, err := db.Query("SELECT id, date, type, note FROM logs WHERE item_name = ? ORDER BY id DESC", name); err == nil {
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
	name := r.PathValue("name")
	var update struct {
		CompletedDate *string `json:"completed_date"`
		Active        *int    `json:"active"`
		TargetValue   *int    `json:"target_value"`
		TargetPeriod  *string `json:"target_period"`
		DisplayOrder  *int    `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	today := time.Now().Format("2006-01-02")
	if update.CompletedDate != nil {
		db.Exec("UPDATE items SET completed_date = ?, last_updated = ? WHERE name = ?", *update.CompletedDate, today, name)
	}
	if update.Active != nil {
		db.Exec("UPDATE items SET active = ?, last_updated = ? WHERE name = ?", *update.Active, today, name)
	}
	if update.TargetValue != nil {
		db.Exec("UPDATE items SET target_value = ?, last_updated = ? WHERE name = ?", *update.TargetValue, today, name)
	}
	if update.TargetPeriod != nil {
		db.Exec("UPDATE items SET target_period = ?, last_updated = ? WHERE name = ?", *update.TargetPeriod, today, name)
	}
	if update.DisplayOrder != nil {
		db.Exec("UPDATE items SET display_order = ? WHERE name = ?", *update.DisplayOrder, name)
	}
	writeJSON(w, map[string]string{"status": "updated"})
}

func handleAddLog(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
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
	db.Exec("INSERT INTO logs (item_name, date, type, note) VALUES (?, ?, ?, ?)", name, date, entry.Type, entry.Note)
	db.Exec("UPDATE items SET last_updated = ? WHERE name = ?", date, name)
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

// Wins

type Win struct {
	ID   int    `json:"id"`
	Date string `json:"date"`
	Note string `json:"note"`
}

func handleGetWins(w http.ResponseWriter, r *http.Request) {
	rows, _ := db.Query("SELECT id, date, note FROM wins ORDER BY date DESC")
	defer rows.Close()
	wins := []Win{}
	for rows.Next() {
		var w Win
		rows.Scan(&w.ID, &w.Date, &w.Note)
		wins = append(wins, w)
	}
	writeJSON(w, wins)
}

func handleAddWin(w http.ResponseWriter, r *http.Request) {
	var win Win
	if err := json.NewDecoder(r.Body).Decode(&win); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	win.Date = time.Now().Format("2006-01-02")
	db.Exec("INSERT INTO wins (date, note) VALUES (?, ?)", win.Date, win.Note)
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

