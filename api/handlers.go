package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// Items

type Item struct {
	Name            string      `json:"name"`
	Type            string      `json:"type"`
	Momentum        string      `json:"momentum"`
	Focus           *string     `json:"focus"`
	Next            *string     `json:"next"`
	URL             *string     `json:"url,omitempty"`
	TargetDate      *string     `json:"target_date,omitempty"`
	SuccessCriteria *string     `json:"success_criteria,omitempty"`
	LastUpdated     string      `json:"last_updated"`
	Milestones      []Milestone `json:"milestones"`
	Log             []LogEntry  `json:"log"`
}

type LogEntry struct {
	ID       int     `json:"id"`
	ItemName string  `json:"item_name,omitempty"`
	Date     string  `json:"date"`
	Type     *string `json:"type,omitempty"`
	Note     string  `json:"note"`
}

type Milestone struct {
	ID       int    `json:"id"`
	ItemName string `json:"item_name,omitempty"`
	Date     string `json:"date"`
	Label    string `json:"label"`
}

func handleGetItems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT name, type, momentum, focus, next, url, target_date, success_criteria, last_updated FROM items ORDER BY CASE momentum WHEN 'rising' THEN 0 WHEN 'steady' THEN 1 WHEN 'stalling' THEN 2 WHEN 'dormant' THEN 3 END")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		rows.Scan(&it.Name, &it.Type, &it.Momentum, &it.Focus, &it.Next, &it.URL, &it.TargetDate, &it.SuccessCriteria, &it.LastUpdated)

		logRows, _ := db.Query("SELECT id, date, type, note FROM logs WHERE item_name = ? ORDER BY date DESC", it.Name)
		it.Log = []LogEntry{}
		for logRows.Next() {
			var l LogEntry
			logRows.Scan(&l.ID, &l.Date, &l.Type, &l.Note)
			it.Log = append(it.Log, l)
		}
		logRows.Close()

		msRows, _ := db.Query("SELECT id, date, label FROM milestones WHERE item_name = ? ORDER BY date DESC", it.Name)
		it.Milestones = []Milestone{}
		for msRows.Next() {
			var m Milestone
			msRows.Scan(&m.ID, &m.Date, &m.Label)
			it.Milestones = append(it.Milestones, m)
		}
		msRows.Close()

		items = append(items, it)
	}
	writeJSON(w, items)
}

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var it Item
	err := db.QueryRow("SELECT name, type, momentum, focus, next, url, target_date, success_criteria, last_updated FROM items WHERE name = ?", name).
		Scan(&it.Name, &it.Type, &it.Momentum, &it.Focus, &it.Next, &it.URL, &it.TargetDate, &it.SuccessCriteria, &it.LastUpdated)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	logRows, _ := db.Query("SELECT id, date, type, note FROM logs WHERE item_name = ? ORDER BY date DESC", name)
	it.Log = []LogEntry{}
	for logRows.Next() {
		var l LogEntry
		logRows.Scan(&l.ID, &l.Date, &l.Type, &l.Note)
		it.Log = append(it.Log, l)
	}
	logRows.Close()

	msRows, _ := db.Query("SELECT id, date, label FROM milestones WHERE item_name = ? ORDER BY date DESC", name)
	it.Milestones = []Milestone{}
	for msRows.Next() {
		var m Milestone
		msRows.Scan(&m.ID, &m.Date, &m.Label)
		it.Milestones = append(it.Milestones, m)
	}
	msRows.Close()

	writeJSON(w, it)
}

func handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var update struct {
		Momentum *string `json:"momentum"`
		Focus    *string `json:"focus"`
		Next     *string `json:"next"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	today := time.Now().Format("2006-01-02")
	if update.Momentum != nil {
		valid := map[string]bool{"rising": true, "steady": true, "stalling": true, "dormant": true}
		if !valid[*update.Momentum] {
			http.Error(w, "momentum must be rising, steady, stalling, or dormant", 400)
			return
		}
		db.Exec("UPDATE items SET momentum = ?, last_updated = ? WHERE name = ?", *update.Momentum, today, name)
	}
	if update.Focus != nil {
		db.Exec("UPDATE items SET focus = ?, last_updated = ? WHERE name = ?", *update.Focus, today, name)
	}
	if update.Next != nil {
		db.Exec("UPDATE items SET next = ?, last_updated = ? WHERE name = ?", *update.Next, today, name)
	}
	writeJSON(w, map[string]string{"status": "updated"})
}

func handleAddLog(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var entry struct {
		Note string  `json:"note"`
		Type *string `json:"type,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	today := time.Now().Format("2006-01-02")
	db.Exec("INSERT INTO logs (item_name, date, type, note) VALUES (?, ?, ?, ?)", name, today, entry.Type, entry.Note)
	db.Exec("UPDATE items SET last_updated = ? WHERE name = ?", today, name)
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
	rows, _ := db.Query("SELECT id, date, body, mind, social, feeling, more_of, less_of FROM check_ins ORDER BY date DESC LIMIT 12")
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
	c.Date = time.Now().Format("2006-01-02")
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

// Engagement

func handleEngagement(w http.ResponseWriter, r *http.Request) {
	var totalItems, recentItems int
	db.QueryRow("SELECT COUNT(*) FROM items").Scan(&totalItems)
	db.QueryRow("SELECT COUNT(*) FROM items WHERE last_updated >= date('now', '-7 days')").Scan(&recentItems)

	var totalGoals, onPaceGoals int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE type = 'Goal' AND target_date IS NOT NULL").Scan(&totalGoals)
	db.QueryRow("SELECT COUNT(*) FROM items WHERE type = 'Goal' AND target_date IS NOT NULL AND momentum IN ('rising', 'steady')").Scan(&onPaceGoals)

	var recentLogs int
	db.QueryRow("SELECT COUNT(DISTINCT date) FROM logs WHERE date >= date('now', '-7 days')").Scan(&recentLogs)

	updatePct := 0.0
	if totalItems > 0 {
		updatePct = float64(recentItems) / float64(totalItems)
	}
	goalPct := 1.0
	if totalGoals > 0 {
		goalPct = float64(onPaceGoals) / float64(totalGoals)
	}

	score := int((updatePct*0.6 + goalPct*0.4) * 100)
	label := "slipping"
	if score >= 70 {
		label = "rising"
	} else if score >= 40 {
		label = "steady"
	}

	writeJSON(w, map[string]any{
		"score":            score,
		"label":            label,
		"items_updated_7d": recentItems,
		"items_total":      totalItems,
		"goals_on_pace":    onPaceGoals,
		"goals_total":      totalGoals,
		"active_days_7d":   recentLogs,
	})
}
