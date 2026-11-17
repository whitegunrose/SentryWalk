package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// scanAt posts one app scan for guard at a local time on 2026-10-20.
func scanAt(t *testing.T, h http.Handler, id, guard, tag, clock string) {
	t.Helper()
	code, body := post(t, h, scanBody(map[string]any{
		"scan_id": id, "guard_id": guard, "tag_id": tag, "scanned_at": "2026-10-20T" + clock,
	}))
	if code != http.StatusOK {
		t.Fatalf("scan %s: %d %v", id, code, body)
	}
}

func setMinTime(t *testing.T, h http.Handler, from, to string, minutes int) int {
	t.Helper()
	b, _ := json.Marshal(MinTime{From: from, To: to, Minutes: minutes})
	req := httptest.NewRequest("POST", "/dashboard/min-times", strings.NewReader(string(b)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func dashboardData(t *testing.T, h http.Handler) DashboardData {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/dashboard/data", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /dashboard/data: %d %s", rec.Code, rec.Body)
	}
	var d DashboardData
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// The example from the spec: A-B is 5 minutes, guard scans A at 12:00 and
// B at 12:02, so the walk is too fast.
func TestTooFastWalkFlagged(t *testing.T) {
	_, h := newTestServer(t)
	if code := setMinTime(t, h, "A", "B", 5); code != http.StatusOK {
		t.Fatalf("set min time: %d", code)
	}
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	scanAt(t, h, "2", "123", "04B27C10", "12:02:00")

	d := dashboardData(t, h)
	if len(d.Walks) != 1 || !d.Walks[0].TooFast {
		t.Fatalf("want one too-fast walk, got %+v", d.Walks)
	}
	if got := d.Walks[0].Flags; len(got) != 1 || got[0] != "TOO FAST A-B 2m00s (min 5 min)" {
		t.Errorf("flags = %v", got)
	}
	// Newest first: the B scan is the too-fast one.
	if !d.Scans[0].TooFast || d.Scans[1].TooFast {
		t.Errorf("want only the B scan marked, got %+v", d.Scans)
	}
}

func TestMinTimeWorksBothDirections(t *testing.T) {
	_, h := newTestServer(t)
	setMinTime(t, h, "B", "A", 5) // stored as A-B
	scanAt(t, h, "1", "123", "04B27C10", "12:00:00")
	scanAt(t, h, "2", "123", "04A19F22", "12:01:00")

	d := dashboardData(t, h)
	if len(d.MinTimes) != 1 || d.MinTimes[0] != (MinTime{"A", "B", 5}) {
		t.Errorf("min times = %+v", d.MinTimes)
	}
	if !d.Walks[0].TooFast {
		t.Error("B then A in 1 minute should be too fast")
	}
}

func TestExactlyMinimumIsNotTooFast(t *testing.T) {
	_, h := newTestServer(t)
	setMinTime(t, h, "A", "B", 5)
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	scanAt(t, h, "2", "123", "04B27C10", "12:05:00")
	if d := dashboardData(t, h); d.Walks[0].TooFast {
		t.Errorf("5 min with a 5 min minimum should be fine: %+v", d.Walks[0])
	}
}

func TestNoRuleMeansNeverTooFast(t *testing.T) {
	_, h := newTestServer(t)
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	scanAt(t, h, "2", "123", "04B27C10", "12:00:10")
	if d := dashboardData(t, h); d.Walks[0].TooFast {
		t.Error("no minimum set, so nothing is too fast")
	}
}

func TestGuardsAndHoursAreSeparateWalks(t *testing.T) {
	_, h := newTestServer(t)
	setMinTime(t, h, "A", "B", 5)
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	scanAt(t, h, "2", "456", "04B27C10", "12:01:00") // other guard: not a pair
	scanAt(t, h, "3", "123", "04A19F22", "12:58:00")
	scanAt(t, h, "4", "123", "04B27C10", "13:01:00") // next hour: new walk

	d := dashboardData(t, h)
	if len(d.Walks) != 3 {
		t.Fatalf("want 3 walks (123@12, 456@12, 123@13), got %d: %+v", len(d.Walks), d.Walks)
	}
	for _, w := range d.Walks {
		if w.TooFast {
			t.Errorf("walk %s %s should not be too fast", w.Guard, w.Start)
		}
	}
	if d.Walks[0].Start != "2026-10-20T13:00:00" {
		t.Errorf("newest walk first, got %s", d.Walks[0].Start)
	}
}

func TestRemoveMinTime(t *testing.T) {
	s, h := newTestServer(t)
	setMinTime(t, h, "A", "B", 5)
	setMinTime(t, h, "A", "B", 0)
	if d := dashboardData(t, h); len(d.MinTimes) != 0 {
		t.Errorf("want no rules, got %+v", d.MinTimes)
	}
	data, _ := os.ReadFile(s.MinTimesFile)
	if string(data) != "from,to,minutes\n" {
		t.Errorf("file = %q", data)
	}
}

func TestBadMinTimeRejected(t *testing.T) {
	_, h := newTestServer(t)
	for _, m := range []MinTime{{"A", "A", 5}, {"", "B", 5}, {"A", "B", -1}, {"A,", "B", 5}} {
		if code := setMinTime(t, h, m.From, m.To, m.Minutes); code != http.StatusUnprocessableEntity {
			t.Errorf("%+v: got %d, want 422", m, code)
		}
	}
}

func TestDashboardPageServed(t *testing.T) {
	_, h := newTestServer(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/dashboard", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "SentryWalk Dashboard") {
		t.Errorf("GET /dashboard: %d", rec.Code)
	}
	if d := dashboardData(t, h); len(d.Checkpoints) != 2 || len(d.Scans) != 0 {
		t.Errorf("empty server: %+v", d)
	}
}
