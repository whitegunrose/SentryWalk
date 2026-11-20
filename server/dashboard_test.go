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
	if len(walksOf(d)) != 1 || !walksOf(d)[0].TooFast {
		t.Fatalf("want one too-fast walk, got %+v", walksOf(d))
	}
	if got := walksOf(d)[0].Flags; len(got) != 1 || got[0] != "TOO FAST A-B 2m00s (min 5 min)" {
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
	if !walksOf(d)[0].TooFast {
		t.Error("B then A in 1 minute should be too fast")
	}
}

func TestExactlyMinimumIsNotTooFast(t *testing.T) {
	_, h := newTestServer(t)
	setMinTime(t, h, "A", "B", 5)
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	scanAt(t, h, "2", "123", "04B27C10", "12:05:00")
	if d := dashboardData(t, h); walksOf(d)[0].TooFast {
		t.Errorf("5 min with a 5 min minimum should be fine: %+v", walksOf(d)[0])
	}
}

func TestNoRuleMeansNeverTooFast(t *testing.T) {
	_, h := newTestServer(t)
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	scanAt(t, h, "2", "123", "04B27C10", "12:00:10")
	if d := dashboardData(t, h); walksOf(d)[0].TooFast {
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
	if len(walksOf(d)) != 3 {
		t.Fatalf("want 3 walks (123@12, 456@12, 123@13), got %d: %+v", len(walksOf(d)), walksOf(d))
	}
	for _, w := range walksOf(d) {
		if w.TooFast {
			t.Errorf("walk %s %s should not be too fast", w.Guard, w.Start)
		}
	}
	if walksOf(d)[0].Start != "2026-10-20T13:00:00" {
		t.Errorf("newest walk first, got %s", walksOf(d)[0].Start)
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

// walksOf lists every walk on the dashboard, newest hour first, then by guard.
func walksOf(d DashboardData) []Walk {
	all := []Walk{}
	for _, h := range d.WalkHours {
		all = append(all, h.Walks...)
	}
	return all
}

// newABCServer has three checkpoints, A B C, all on the route.
func newABCServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	s, h := newTestServer(t)
	if err := os.WriteFile(s.TagsFile, []byte("tag_id,checkpoint\n04A19F22,A\n04B27C10,B\n04C3D5E8,C\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.RouteFile, []byte("checkpoints=A,B,C\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return s, h
}

// Each scan shows where the walk stood right after it, in any order.
func TestScanShowsWalkProgress(t *testing.T) {
	_, h := newABCServer(t)
	scanAt(t, h, "1", "123", "04C3D5E8", "12:00:00") // C
	scanAt(t, h, "2", "123", "04A19F22", "12:05:00") // A
	scanAt(t, h, "3", "123", "04A19F22", "12:06:00") // A again
	scanAt(t, h, "4", "123", "04B27C10", "12:10:00") // B

	d := dashboardData(t, h)
	if strings.Join(d.Route, ",") != "A,B,C" {
		t.Errorf("route = %v", d.Route)
	}
	// Scans are newest first, so reverse to read them in order.
	want := []struct {
		done      int
		remaining string
		repeat    bool
		complete  bool
	}{{1, "A,B", false, false}, {2, "B", false, false}, {2, "B", true, false}, {3, "", false, true}}
	for i, w := range want {
		p := d.Scans[len(d.Scans)-1-i].Progress
		if p == nil {
			t.Fatalf("scan %d has no progress", i+1)
		}
		if p.Done != w.done || p.Total != 3 || strings.Join(p.Remaining, ",") != w.remaining || p.Repeat != w.repeat || p.Complete != w.complete {
			t.Errorf("scan %d: got %+v, want %+v", i+1, *p, w)
		}
	}
	walk := walksOf(d)[0]
	if walk.Progress == nil || !walk.Progress.Complete || walk.Progress.Done != 3 {
		t.Errorf("walk progress = %+v", walk.Progress)
	}
}

// Progress starts over for each guard and each hour.
func TestProgressIsPerGuardAndHour(t *testing.T) {
	_, h := newABCServer(t)
	scanAt(t, h, "1", "123", "04A19F22", "12:50:00")
	scanAt(t, h, "2", "456", "04B27C10", "12:51:00")
	scanAt(t, h, "3", "123", "04B27C10", "13:01:00")

	d := dashboardData(t, h)
	for _, s := range d.Scans {
		if s.Progress == nil || s.Progress.Done != 1 {
			t.Errorf("scan %s (%s at %s) should be 1 of 3, got %+v", s.ScanID, s.Guard, s.Happened, s.Progress)
		}
	}
}

func TestUnknownTagAndNoRouteHaveNoProgress(t *testing.T) {
	_, h := newTestServer(t) // no route file
	scanAt(t, h, "1", "123", "04A19F22", "12:00:00")
	d := dashboardData(t, h)
	if d.Scans[0].Progress != nil || walksOf(d)[0].Progress != nil {
		t.Errorf("no route should mean no progress: %+v", d.Scans[0].Progress)
	}

	_, h = newABCServer(t)
	scanAt(t, h, "1", "123", "04FFFFFF", "12:00:00") // not in the tag list
	if d := dashboardData(t, h); d.Scans[0].Progress != nil {
		t.Errorf("unknown tag should have no progress: %+v", d.Scans[0].Progress)
	}
}

// Walks are grouped by hour (newest first), then by guard.
func TestWalksGroupedByHourThenGuard(t *testing.T) {
	_, h := newABCServer(t)
	scanAt(t, h, "1", "6767", "04A19F22", "13:10:00")
	scanAt(t, h, "2", "1111", "04A19F22", "13:12:00")
	scanAt(t, h, "3", "1111", "04B27C10", "14:05:00")

	d := dashboardData(t, h)
	if len(d.WalkHours) != 2 {
		t.Fatalf("want 2 hours, got %+v", d.WalkHours)
	}
	if d.WalkHours[0].Start != "2026-10-20T14:00:00" || len(d.WalkHours[0].Walks) != 1 {
		t.Errorf("first hour = %+v", d.WalkHours[0])
	}
	second := d.WalkHours[1]
	if second.Start != "2026-10-20T13:00:00" || len(second.Walks) != 2 ||
		second.Walks[0].Guard != "1111" || second.Walks[1].Guard != "6767" {
		t.Errorf("1 PM hour should have guards 1111 then 6767: %+v", second)
	}
}
