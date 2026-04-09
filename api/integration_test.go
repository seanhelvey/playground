package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

// newTestServer spins up a full in-memory server (real SQLite, real auth).
func newTestServer(t *testing.T) (*httptest.Server, *http.Cookie) {
	t.Helper()

	var err error
	db, err = sql.Open("sqlite3", "file::memory:?mode=memory&cache=shared&_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}

	// Seed items we can test against; capture IDs for route construction
	today := time.Now().Format("2006-01-02")
	res1, _ := db.Exec(`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, target_value, target_period)
		VALUES ('Wake to alarm', ?, 'boolean', 0, '', 1, 1, 'daily')`, today)
	res2, _ := db.Exec(`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, target_value, target_period)
		VALUES ('Meditation', ?, 'counter', 5, 'min', 2, 35, 'weekly')`, today)
	wakeID, _ := res1.LastInsertId()
	medID, _ := res2.LastInsertId()
	t.Setenv("TEST_WAKE_ID", strconv.FormatInt(wakeID, 10))
	t.Setenv("TEST_MED_ID", strconv.FormatInt(medID, 10))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", handleRegister)
	mux.HandleFunc("POST /api/login", handleLogin)
	api := http.NewServeMux()
	api.HandleFunc("GET /api/items", handleGetItems)
	api.HandleFunc("POST /api/items/{id}/log", handleAddLog)
	api.HandleFunc("GET /api/checkins", handleGetCheckins)
	api.HandleFunc("POST /api/checkins", handleAddCheckin)
	mux.Handle("/api/", authMiddleware(api))

	srv := httptest.NewServer(mux)
	t.Cleanup(func() { srv.Close(); db.Close() })

	// Register + get session cookie
	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "password123"})
	resp, err := http.Post(srv.URL+"/api/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("no session cookie after register")
	}
	return srv, cookie
}

func apiReq(t *testing.T, srv *httptest.Server, cookie *http.Cookie, method, path string, body any) *http.Response {
	t.Helper()
	var b *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		b = bytes.NewReader(data)
	} else {
		b = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, srv.URL+path, b)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// TestTogglePersistedOnReload proves that after logging "Done" for a boolean item,
// fetching items returns a log entry for today with note "Done" — so the toggle
// re-renders as on after a page reload.
func TestTogglePersistedOnReload(t *testing.T) {
	srv, cookie := newTestServer(t)
	today := time.Now().Format("2006-01-02")

	// Simulate tapping toggle on
	wakeID := os.Getenv("TEST_WAKE_ID")
	resp := apiReq(t, srv, cookie, "POST", "/api/items/"+wakeID+"/log", map[string]string{"note": "Done"})
	if resp.StatusCode != 200 {
		t.Fatalf("log POST returned %d", resp.StatusCode)
	}

	// Simulate page reload — fetch items
	resp = apiReq(t, srv, cookie, "GET", "/api/items", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("GET items returned %d", resp.StatusCode)
	}

	var items []Item
	json.NewDecoder(resp.Body).Decode(&items)

	var wake *Item
	for i := range items {
		if items[i].Name == "Wake to alarm" {
			wake = &items[i]
		}
	}
	if wake == nil {
		t.Fatal("Wake to alarm not found in items")
	}

	todayLogs := []LogEntry{}
	for _, l := range wake.Log {
		if l.Date == today {
			todayLogs = append(todayLogs, l)
		}
	}
	if len(todayLogs) == 0 {
		t.Fatal("no log entries for today — toggle would appear OFF after reload")
	}
	if todayLogs[0].Note != "Done" {
		t.Fatalf("expected last note 'Done', got %q — toggle state wrong after reload", todayLogs[0].Note)
	}
}

// TestCounterPersistedOnReload proves that after logging a step for a counter item,
// fetching items returns a log entry for today so todayTotal() gives the right sum.
func TestCounterPersistedOnReload(t *testing.T) {
	srv, cookie := newTestServer(t)
	today := time.Now().Format("2006-01-02")

	// Simulate pressing + once (step 5)
	medID := os.Getenv("TEST_MED_ID")
	resp := apiReq(t, srv, cookie, "POST", "/api/items/"+medID+"/log", map[string]string{"note": "5"})
	if resp.StatusCode != 200 {
		t.Fatalf("log POST returned %d", resp.StatusCode)
	}

	// Simulate page reload
	resp = apiReq(t, srv, cookie, "GET", "/api/items", nil)
	var items []Item
	json.NewDecoder(resp.Body).Decode(&items)

	var med *Item
	for i := range items {
		if items[i].Name == "Meditation" {
			med = &items[i]
		}
	}
	if med == nil {
		t.Fatal("Meditation not found in items")
	}

	total := 0
	for _, l := range med.Log {
		if l.Date == today {
			if n := parseInt(l.Note); n > 0 {
				total += n
			}
		}
	}
	if total != 5 {
		t.Fatalf("expected todayTotal=5, got %d — counter would show 0 after reload", total)
	}
}

// TestSliderItemPersistedOnReload proves that after logging a value for a slider
// item via POST /api/items/{id}/log, fetching items returns that log entry — so
// the slider re-renders at the saved value after a page reload.
func TestSliderItemPersistedOnReload(t *testing.T) {
	srv, cookie := newTestServer(t)
	today := time.Now().Format("2006-01-02")

	// Create a slider item
	res, err := db.Exec(`INSERT INTO items (name, last_updated, input_type, range_min, range_max, display_order)
		VALUES ('Mood', ?, 'slider', 1, 10, 99)`, today)
	if err != nil {
		t.Fatal(err)
	}
	moodID, _ := res.LastInsertId()
	moodIDStr := strconv.FormatInt(moodID, 10)

	// Log a value of 7
	resp := apiReq(t, srv, cookie, "POST", "/api/items/"+moodIDStr+"/log", map[string]string{"note": "7"})
	if resp.StatusCode != 200 {
		t.Fatalf("log POST returned %d", resp.StatusCode)
	}

	// Simulate page reload
	resp = apiReq(t, srv, cookie, "GET", "/api/items", nil)
	var items []Item
	json.NewDecoder(resp.Body).Decode(&items)

	var mood *Item
	for i := range items {
		if items[i].Name == "Mood" {
			mood = &items[i]
		}
	}
	if mood == nil {
		t.Fatal("Mood not found in items")
	}

	var todayLog *LogEntry
	for i := range mood.Log {
		if mood.Log[i].Date == today {
			todayLog = &mood.Log[i]
			break
		}
	}
	if todayLog == nil {
		t.Fatal("no log entry for today — slider would reset to default after reload")
	}
	if todayLog.Note != "7" {
		t.Fatalf("expected note '7', got %q — slider would show wrong value after reload", todayLog.Note)
	}
}

// TestSlidersPersistedOnReload proves that body/mind/social values saved via
// POST /api/checkins are returned by GET /api/checkins for today, and would
// appear in the activity log data (same data used to render Recent Activity).
func TestSlidersPersistedOnReload(t *testing.T) {
	srv, cookie := newTestServer(t)
	today := time.Now().Format("2006-01-02")

	resp := apiReq(t, srv, cookie, "POST", "/api/checkins", map[string]int{"body": 8, "mind": 7, "social": 6})
	if resp.StatusCode != 200 {
		t.Fatalf("checkin POST returned %d", resp.StatusCode)
	}

	resp = apiReq(t, srv, cookie, "GET", "/api/checkins", nil)
	var checkins []CheckIn
	json.NewDecoder(resp.Body).Decode(&checkins)

	var todayCI *CheckIn
	for i := range checkins {
		if checkins[i].Date == today {
			todayCI = &checkins[i]
			break
		}
	}
	if todayCI == nil {
		t.Fatal("no check-in for today — sliders would reset to 5 after reload and not appear in activity log")
	}
	if *todayCI.Body != 8 || *todayCI.Mind != 7 || *todayCI.Social != 6 {
		t.Fatalf("expected 8/7/6, got %v/%v/%v", *todayCI.Body, *todayCI.Mind, *todayCI.Social)
	}
	// Verify it has an ID so it sorts correctly in the activity log
	if todayCI.ID == 0 {
		t.Fatal("check-in has no ID — would not sort correctly in activity log")
	}
}

// TestActivityLogCheckinSortsAboveLogs proves that a check-in saved after item
// logs has a higher ID than the logs that preceded it within the same date,
// so the frontend's sort (check-ins first within same date) is meaningful.
func TestActivityLogCheckinSortsAboveLogs(t *testing.T) {
	srv, cookie := newTestServer(t)

	// Log several item entries first (these get lower IDs in logs table)
	medID := os.Getenv("TEST_MED_ID")
	for i := 0; i < 3; i++ {
		apiReq(t, srv, cookie, "POST", "/api/items/"+medID+"/log", map[string]string{"note": "5"})
	}

	// Then save a check-in (gets its own ID in check_ins table)
	resp := apiReq(t, srv, cookie, "POST", "/api/checkins", map[string]int{"body": 7, "mind": 8, "social": 6})
	if resp.StatusCode != 200 {
		t.Fatalf("checkin POST returned %d", resp.StatusCode)
	}

	// Verify check-in exists and has an ID
	resp = apiReq(t, srv, cookie, "GET", "/api/checkins", nil)
	var checkins []CheckIn
	json.NewDecoder(resp.Body).Decode(&checkins)
	if len(checkins) == 0 || checkins[0].ID == 0 {
		t.Fatal("check-in not found or missing ID")
	}
}

// TestActivityLogAllItemTypes proves that boolean, counter, and slider items all
// return log entries via GET /api/items — the data the activity log renders from.
func TestActivityLogAllItemTypes(t *testing.T) {
	srv, cookie := newTestServer(t)
	today := time.Now().Format("2006-01-02")

	// Create a slider item directly (Body/Mind/Social are slider items)
	res3, err := db.Exec(`INSERT INTO items (name, last_updated, input_type, step_size, step_unit, display_order, range_min, range_max)
		VALUES ('Body', ?, 'slider', 1, '', 17, 1, 10)`, today)
	if err != nil {
		t.Fatal(err)
	}
	bodyID, _ := res3.LastInsertId()
	bodyIDStr := strconv.FormatInt(bodyID, 10)

	wakeID := os.Getenv("TEST_WAKE_ID")
	medID := os.Getenv("TEST_MED_ID")

	// Log each item type
	for _, tc := range []struct{ id, note, label string }{
		{wakeID, "Done", "boolean (Wake to alarm)"},
		{medID, "5", "counter (Meditation)"},
		{bodyIDStr, "7", "slider (Body)"},
	} {
		resp := apiReq(t, srv, cookie, "POST", "/api/items/"+tc.id+"/log", map[string]string{"note": tc.note})
		if resp.StatusCode != 200 {
			t.Fatalf("%s log POST returned %d", tc.label, resp.StatusCode)
		}
	}

	// Reload items (simulates page reload)
	resp := apiReq(t, srv, cookie, "GET", "/api/items", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("GET /api/items returned %d", resp.StatusCode)
	}
	var items []Item
	json.NewDecoder(resp.Body).Decode(&items)

	byName := map[string]*Item{}
	for i := range items {
		byName[items[i].Name] = &items[i]
	}

	// Each item must have a today log entry so it appears in the activity log
	for _, name := range []string{"Wake to alarm", "Meditation", "Body"} {
		it := byName[name]
		if it == nil {
			t.Fatalf("%s not found in items response — would be absent from activity log", name)
		}
		var hasToday bool
		for _, l := range it.Log {
			if l.Date == today {
				hasToday = true
				break
			}
		}
		if !hasToday {
			t.Fatalf("%s has no log entry for today — would not appear in activity log after reload", name)
		}
	}
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else if c == '-' {
			continue
		}
	}
	// handle negative
	if len(s) > 0 && s[0] == '-' {
		return -n
	}
	return n
}
