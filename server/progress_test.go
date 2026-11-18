package main

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// newRouteServer is newTestServer plus a route file with checkpoints A,B,
// the two checkpoints in the test tag list.
func newRouteServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	s, h := newTestServer(t)
	if err := os.WriteFile(s.RouteFile, []byte("checkpoints=A,B\nshift_end=2026-10-20T06:00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return s, h
}

// tapAt taps a tag as guard at a local time on 2026-10-20 and returns the
// result page that the browser is sent to.
func tapAt(t *testing.T, s *Server, h http.Handler, guard, tag, clock string) string {
	t.Helper()
	at, err := time.ParseInLocation(javaTime, "2026-10-20T"+clock, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return at }
	rec := get(h, "/scan?tag="+tag, guard)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("tap: %d %s", rec.Code, rec.Body)
	}
	page := get(h, rec.Header().Get("Location"), guard)
	if page.Code != http.StatusOK {
		t.Fatalf("result page: %d %s", page.Code, page.Body)
	}
	return page.Body.String()
}

func TestTapShowsRemainingCheckpoints(t *testing.T) {
	s, h := newRouteServer(t)

	page := tapAt(t, s, h, "G1", "04A19F22", "01:02:00")
	if !strings.Contains(page, "1 of 2 checkpoints") || !strings.Contains(page, "Remaining: <span class=\"mono\">B</span>") {
		t.Errorf("after A, want 1 of 2 and B remaining:\n%s", page)
	}
	if !strings.Contains(page, "1:00 AM walk") {
		t.Errorf("want the 1:00 AM walk:\n%s", page)
	}

	page = tapAt(t, s, h, "G1", "04B27C10", "01:10:00")
	if !strings.Contains(page, "2 of 2 checkpoints") || !strings.Contains(page, "Round complete!") {
		t.Errorf("after A and B, want round complete:\n%s", page)
	}
}

func TestTapSameCheckpointTwiceIsAlreadyScanned(t *testing.T) {
	s, h := newRouteServer(t)
	if page := tapAt(t, s, h, "G1", "04A19F22", "01:02:00"); strings.Contains(page, "Already scanned") {
		t.Errorf("first tap should not be already scanned:\n%s", page)
	}
	page := tapAt(t, s, h, "G1", "04A19F22", "01:05:00")
	if !strings.Contains(page, "Already scanned") || !strings.Contains(page, "1 of 2 checkpoints") {
		t.Errorf("second tap of A should be already scanned, still 1 of 2:\n%s", page)
	}
}

func TestNewHourAndOtherGuardsStartFresh(t *testing.T) {
	s, h := newRouteServer(t)
	tapAt(t, s, h, "G1", "04A19F22", "01:50:00")

	// Same guard, next hour: a new walk.
	page := tapAt(t, s, h, "G1", "04A19F22", "02:01:00")
	if strings.Contains(page, "Already scanned") || !strings.Contains(page, "1 of 2 checkpoints") {
		t.Errorf("2:00 walk should start fresh:\n%s", page)
	}

	// Another guard in the same hour: their own walk.
	page = tapAt(t, s, h, "G2", "04A19F22", "02:03:00")
	if strings.Contains(page, "Already scanned") || !strings.Contains(page, "1 of 2 checkpoints") {
		t.Errorf("G2 should have their own walk:\n%s", page)
	}
}

func TestNoRouteFileShowsNoProgress(t *testing.T) {
	s, h := newTestServer(t) // no R_route.txt
	page := tapAt(t, s, h, "G1", "04A19F22", "01:02:00")
	if strings.Contains(page, "checkpoints") || !strings.Contains(page, "Checkpoint A") {
		t.Errorf("want the plain result page without progress:\n%s", page)
	}
}

func TestLoadRoute(t *testing.T) {
	path := t.TempDir() + "/R.txt"
	os.WriteFile(path, []byte("walk_starts=x\r\ncheckpoints= A, B ,C\r\n"), 0o644)
	route, err := loadRoute(path)
	if err != nil || strings.Join(route, ",") != "A,B,C" {
		t.Errorf("route = %v, %v", route, err)
	}
}
