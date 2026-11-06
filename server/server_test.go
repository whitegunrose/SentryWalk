package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// pacific is a fixed UTC-7 zone, so time conversion tests give the same
// answer on any computer.
var pacific = time.FixedZone("PDT", -7*60*60)

// newTestServer makes a server whose files live in a temp folder, with a
// small tag list and a fixed "now".
func newTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	old := time.Local
	time.Local = pacific
	t.Cleanup(func() { time.Local = old })

	dir := t.TempDir()
	tags := filepath.Join(dir, "T_tags.csv")
	if err := os.WriteFile(tags, []byte("tag_id,checkpoint\n04A19F22,A\n04B27C10,B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(tags, filepath.Join(dir, "data", "E_live.csv"), filepath.Join(dir, "data", "scans.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return time.Date(2026, 10, 20, 1, 2, 1, 500, pacific) }
	return s, s.Routes()
}

func scanBody(overrides map[string]any) map[string]any {
	b := map[string]any{
		"scan_id":    "abc-1",
		"tag_id":     "04A19F22",
		"guard_id":   "G1",
		"scanned_at": "2026-10-20T01:02:00", // no zone: taken as local time
	}
	for k, v := range overrides {
		b[k] = v
	}
	return b
}

func post(t *testing.T, h http.Handler, body any) (int, map[string]any) {
	t.Helper()
	var raw string
	if s, ok := body.(string); ok {
		raw = s
	} else {
		b, _ := json.Marshal(body)
		raw = string(b)
	}
	req := httptest.NewRequest(http.MethodPost, "/scans", strings.NewReader(raw))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func eLines(t *testing.T, s *Server) []string {
	t.Helper()
	data, err := os.ReadFile(s.EventsFile)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}

func TestKnownTagWrittenInEFormat(t *testing.T) {
	s, h := newTestServer(t)
	code, resp := post(t, h, scanBody(nil))
	if code != 200 || resp["status"] != "saved" || resp["checkpoint"] != "A" || resp["known_tag"] != true {
		t.Fatalf("got %d %v", code, resp)
	}
	want := []string{
		"happened,arrived,guard,tag_id",
		"2026-10-20T01:02:00,2026-10-20T01:02:01,G1,04A19F22",
	}
	if got := eLines(t, s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("E file:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestHeaderWrittenOnce(t *testing.T) {
	s, h := newTestServer(t)
	post(t, h, scanBody(map[string]any{"scan_id": "1"}))
	post(t, h, scanBody(map[string]any{"scan_id": "2"}))
	lines := eLines(t, s)
	if len(lines) != 3 || lines[0] != eHeader || strings.HasPrefix(lines[2], "happened") {
		t.Fatalf("E file: %q", lines)
	}
}

func TestUTCConvertedToLocalJavaTime(t *testing.T) {
	s, h := newTestServer(t)
	// What the Flutter app sends: UTC with fractions of a second.
	post(t, h, scanBody(map[string]any{"scanned_at": "2026-10-20T08:02:00.123456Z"}))
	happened := strings.Split(eLines(t, s)[1], ",")[0]
	if happened != "2026-10-20T01:02:00" {
		t.Fatalf("happened = %q, want 2026-10-20T01:02:00 (UTC-7, no zone, no fraction)", happened)
	}
}

func TestUnknownTagStillWritten(t *testing.T) {
	s, h := newTestServer(t)
	_, resp := post(t, h, scanBody(map[string]any{"tag_id": "deadbeef"}))
	if resp["known_tag"] != false || resp["checkpoint"] != nil {
		t.Fatalf("resp = %v", resp)
	}
	if line := eLines(t, s)[1]; !strings.HasSuffix(line, ",DEADBEEF") {
		t.Fatalf("line = %q", line)
	}
}

func TestDuplicateNotWrittenTwice(t *testing.T) {
	s, h := newTestServer(t)
	post(t, h, scanBody(nil))
	_, resp := post(t, h, scanBody(nil))
	if resp["status"] != "duplicate" {
		t.Fatalf("resp = %v", resp)
	}
	if n := len(eLines(t, s)); n != 2 {
		t.Fatalf("E file has %d lines, want 2", n)
	}
}

func TestDuplicateCaughtAfterRestart(t *testing.T) {
	s, h := newTestServer(t)
	post(t, h, scanBody(nil))

	restarted, err := NewServer(s.TagsFile, s.EventsFile, s.ScanLog)
	if err != nil {
		t.Fatal(err)
	}
	_, resp := post(t, restarted.Routes(), scanBody(nil))
	if resp["status"] != "duplicate" {
		t.Fatalf("resp = %v", resp)
	}
}

func TestTagAddedWithoutRestart(t *testing.T) {
	s, h := newTestServer(t)
	f, _ := os.OpenFile(s.TagsFile, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString("04C3D5E8,C\n")
	f.Close()
	_, resp := post(t, h, scanBody(map[string]any{"tag_id": "04C3D5E8"}))
	if resp["checkpoint"] != "C" {
		t.Fatalf("resp = %v", resp)
	}
}

func TestBadInputRejected(t *testing.T) {
	cases := []struct {
		name string
		body any
		code int
	}{
		{"not JSON", "{oops", 400},
		{"missing everything", map[string]any{"tag_id": "04A19F22"}, 422},
		{"missing scan_id", scanBody(map[string]any{"scan_id": ""}), 422},
		{"missing guard", scanBody(map[string]any{"guard_id": " "}), 422},
		{"comma in guard", scanBody(map[string]any{"guard_id": "G1,X"}), 422},
		{"newline in guard", scanBody(map[string]any{"guard_id": "G1\nfake line"}), 422},
		{"non-hex tag", scanBody(map[string]any{"tag_id": "04A1,ZZ"}), 422},
		{"bad time", scanBody(map[string]any{"scanned_at": "2026-10-20T25:99:00"}), 422},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, h := newTestServer(t)
			code, resp := post(t, h, c.body)
			if code != c.code || resp["error"] == nil {
				t.Fatalf("got %d %v, want %d with an error message", code, resp, c.code)
			}
			if _, err := os.Stat(s.EventsFile); err == nil {
				t.Fatal("rejected scan was written to the E file")
			}
		})
	}
}

func TestListScansNewestFirst(t *testing.T) {
	_, h := newTestServer(t)
	for _, id := range []string{"1", "2", "3"} {
		post(t, h, scanBody(map[string]any{"scan_id": id}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/scans?limit=2", nil))
	var got []ScanRecord
	json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 2 || got[0].ScanID != "3" || got[1].ScanID != "2" {
		t.Fatalf("got %+v", got)
	}
}

func TestListScansEmpty(t *testing.T) {
	_, h := newTestServer(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/scans", nil))
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHealth(t *testing.T) {
	_, h := newTestServer(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
