package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// get sends a GET, optionally with the guard cookie set.
func get(h http.Handler, target, guard string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if guard != "" {
		req.AddCookie(&http.Cookie{Name: guardCookie, Value: guard})
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestTapWithoutGuardRedirectsToGuardPage(t *testing.T) {
	s, h := newTestServer(t)
	rec := get(h, "/scan?tag=04A19F22", "")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("code = %d, want 303", rec.Code)
	}
	want := "/guard?next=" + url.QueryEscape("/scan?tag=04A19F22")
	if loc := rec.Header().Get("Location"); loc != want {
		t.Errorf("Location = %q, want %q", loc, want)
	}
	if _, err := readFileIfExists(s.EventsFile); err == nil {
		t.Error("a scan was written without a guard ID")
	}
}

func TestGuardSaveSetsCookieAndReturnsToScan(t *testing.T) {
	_, h := newTestServer(t)
	form := url.Values{"id": {" G7 "}, "next": {"/scan?tag=04A19F22"}}
	req := httptest.NewRequest(http.MethodPost, "/guard", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/scan?tag=04A19F22" {
		t.Fatalf("got %d to %q, want 303 to the scan link", rec.Code, rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != guardCookie || cookies[0].Value != "G7" {
		t.Errorf("cookies = %v, want %s=G7", cookies, guardCookie)
	}
}

func TestGuardSaveRejectsBadIDAndOffsiteNext(t *testing.T) {
	_, h := newTestServer(t)
	for _, id := range []string{"", "G1,G2", strings.Repeat("x", 33)} {
		form := url.Values{"id": {id}}
		req := httptest.NewRequest(http.MethodPost, "/guard", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("id %q: code = %d, want 422", id, rec.Code)
		}
	}
	if got := safeNext("https://evil.example/scan?tag=1"); got != "" {
		t.Errorf("safeNext allowed an outside link: %q", got)
	}
}

func TestTapWritesEventAndRedirectsToResult(t *testing.T) {
	s, h := newTestServer(t)
	rec := get(h, "/scan?tag=04a19f22", "G1") // lower case is accepted
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("code = %d, want 303; body: %s", rec.Code, rec.Body)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/scan/ok?") || !strings.Contains(loc, "checkpoint=A") {
		t.Errorf("Location = %q, want /scan/ok with checkpoint=A", loc)
	}

	lines := eLines(t, s)
	want := []string{eHeader, "2026-10-20T01:02:01,2026-10-20T01:02:01,G1,04A19F22"}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("E file =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}

	// The result page shows the checkpoint and saves nothing more.
	page := get(h, loc, "G1")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Checkpoint A") {
		t.Errorf("result page: %d %s", page.Code, page.Body)
	}
	if n := len(eLines(t, s)); n != 2 {
		t.Errorf("E has %d lines after viewing the result page, want 2", n)
	}
}

func TestEachTapIsANewScan(t *testing.T) {
	s, h := newTestServer(t)
	get(h, "/scan?tag=04A19F22", "G1")
	get(h, "/scan?tag=04B27C10", "G1")
	get(h, "/scan?tag=04A19F22", "G1")
	if n := len(eLines(t, s)); n != 4 {
		t.Errorf("E has %d lines, want header + 3 scans", n)
	}
	// They also show up in GET /scans.
	list := get(h, "/scans", "")
	if c := strings.Count(list.Body.String(), `"tag_type":"ndef-link"`); c != 3 {
		t.Errorf("GET /scans lists %d link scans, want 3", c)
	}
}

func TestTapUnknownTagIsSavedWithWarning(t *testing.T) {
	s, h := newTestServer(t)
	rec := get(h, "/scan?tag=DEADBEEF", "G1")
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "checkpoint=") {
		t.Errorf("unknown tag got a checkpoint: %q", loc)
	}
	if lines := eLines(t, s); lines[len(lines)-1] != "2026-10-20T01:02:01,2026-10-20T01:02:01,G1,DEADBEEF" {
		t.Errorf("last E line = %q", lines[len(lines)-1])
	}
	page := get(h, loc, "G1")
	if !strings.Contains(page.Body.String(), "unknown tag") {
		t.Errorf("result page doesn't warn about the unknown tag: %s", page.Body)
	}
}

func TestTapBadTagIsRejected(t *testing.T) {
	s, h := newTestServer(t)
	for _, target := range []string{"/scan", "/scan?tag=", "/scan?tag=CP01", "/scan?tag=04A1%2C9F22"} {
		if rec := get(h, target, "G1"); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: code = %d, want 422", target, rec.Code)
		}
	}
	if _, err := readFileIfExists(s.EventsFile); err == nil {
		t.Error("a bad tag link wrote to E")
	}
}

func readFileIfExists(path string) ([]byte, error) {
	return os.ReadFile(path)
}
