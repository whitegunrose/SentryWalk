package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// eHeader is line 1 of the events file E, exactly as EventReader.java expects.
const eHeader = "happened,arrived,guard,tag_id"

// javaTime is the layout Java's LocalDateTime.parse reads: no time zone,
// whole seconds. (Go layouts are written using the reference time
// Mon Jan 2 15:04:05 2006.)
const javaTime = "2006-01-02T15:04:05"

var hexOnly = regexp.MustCompile(`^[0-9A-F]+$`)

// ScanIn is the JSON the phone app POSTs to /scans.
type ScanIn struct {
	ScanID    string  `json:"scan_id"`    // unique per scan, used to drop duplicate retries
	TagID     string  `json:"tag_id"`     // tag UID as hex, e.g. 04A19F22
	GuardID   string  `json:"guard_id"`   // e.g. G1
	ScannedAt string  `json:"scanned_at"` // ISO 8601, usually UTC from the phone
	NdefURL   *string `json:"ndef_url"`   // URL stored on the tag, if any (NTAG 424 DNA later)
	TagType   *string `json:"tag_type"`
}

// ScanRecord is one line of the scans.jsonl detail log.
type ScanRecord struct {
	ScanID     string  `json:"scan_id"`
	Happened   string  `json:"happened"`
	Arrived    string  `json:"arrived"`
	Guard      string  `json:"guard"`
	TagID      string  `json:"tag_id"`
	Checkpoint *string `json:"checkpoint"` // null = tag not in T
	NdefURL    *string `json:"ndef_url"`
	TagType    *string `json:"tag_type"`
}

// ScanResponse is what /scans sends back to the phone.
type ScanResponse struct {
	Status     string  `json:"status"` // "saved" or "duplicate"
	Checkpoint *string `json:"checkpoint"`
	KnownTag   bool    `json:"known_tag"`
}

// Server holds the file paths and the set of scan IDs already written.
type Server struct {
	TagsFile   string
	EventsFile string
	ScanLog    string

	// Now returns the current time. Tests replace it to get fixed times.
	Now func() time.Time

	mu   sync.Mutex // one write at a time, so lines never interleave
	seen map[string]bool
}

// NewServer loads the scan IDs already in the log so duplicates are still
// caught after a restart.
func NewServer(tagsFile, eventsFile, scanLog string) (*Server, error) {
	s := &Server{
		TagsFile:   tagsFile,
		EventsFile: eventsFile,
		ScanLog:    scanLog,
		Now:        time.Now,
		seen:       map[string]bool{},
	}
	f, err := os.Open(scanLog)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r ScanRecord
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.ScanID != "" {
			s.seen[r.ScanID] = true
		}
	}
	return s, sc.Err()
}

// Routes returns the HTTP handler with every endpoint.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /scans", s.handleScan)
	mux.HandleFunc("GET /scans", s.handleList)

	// Tag links opened in the phone's browser (see tap.go)
	mux.HandleFunc("GET /guard", s.handleGuardForm)
	mux.HandleFunc("POST /guard", s.handleGuardSave)
	mux.HandleFunc("GET /scan", s.handleTap)
	mux.HandleFunc("GET /scan/ok", s.handleTapResult)
	return mux
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	var in ScanIn
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "body is not valid JSON: "+err.Error())
		return
	}

	// Go fills missing JSON fields with "" instead of failing, so check them here.
	in.ScanID = strings.TrimSpace(in.ScanID)
	in.GuardID = strings.TrimSpace(in.GuardID)
	tagID := strings.ToUpper(strings.TrimSpace(in.TagID))
	switch {
	case in.ScanID == "":
		writeError(w, http.StatusUnprocessableEntity, "scan_id is required")
		return
	case tagID == "":
		writeError(w, http.StatusUnprocessableEntity, "tag_id is required")
		return
	case !hexOnly.MatchString(tagID):
		writeError(w, http.StatusUnprocessableEntity, "tag_id must be hex digits only, e.g. 04A19F22")
		return
	case in.GuardID == "":
		writeError(w, http.StatusUnprocessableEntity, "guard_id is required")
		return
	case strings.ContainsAny(in.GuardID, ",\r\n"):
		// A comma or line break would break the CSV columns.
		writeError(w, http.StatusUnprocessableEntity, "guard_id cannot contain commas or line breaks")
		return
	}
	scannedAt, err := parseTime(in.ScannedAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "scanned_at must be an ISO 8601 time, e.g. 2026-10-20T01:02:00Z")
		return
	}

	tags, err := loadTags(s.TagsFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reading tag list: "+err.Error())
		return
	}
	var checkpoint *string
	if name, ok := tags[tagID]; ok {
		checkpoint = &name
	}
	resp := ScanResponse{Status: "saved", Checkpoint: checkpoint, KnownTag: checkpoint != nil}

	rec := ScanRecord{
		ScanID:     in.ScanID,
		Happened:   toJavaTime(scannedAt),
		Arrived:    toJavaTime(s.Now()),
		Guard:      in.GuardID,
		TagID:      tagID,
		Checkpoint: checkpoint,
		NdefURL:    in.NdefURL,
		TagType:    in.TagType,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// The phone retries scans that were queued offline, so the same scan can
	// arrive twice. Accept it but don't write it again.
	if s.seen[in.ScanID] {
		resp.Status = "duplicate"
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Unknown tags are still written: the Java checker turns them into
	// "warning: line N", the same as in the simulation.
	if err := s.appendEvent(rec); err != nil {
		writeError(w, http.StatusInternalServerError, "writing events file: "+err.Error())
		return
	}
	if err := s.appendLog(rec); err != nil {
		writeError(w, http.StatusInternalServerError, "writing scan log: "+err.Error())
		return
	}
	s.seen[in.ScanID] = true

	name := "UNKNOWN TAG"
	if checkpoint != nil {
		name = *checkpoint
	}
	log.Printf("[scan] %s  guard=%s  tag=%s  -> %s", rec.Happened, rec.Guard, rec.TagID, name)
	writeJSON(w, http.StatusOK, resp)
}

// handleList returns the most recent scans first. ?limit=N (default 50).
func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusUnprocessableEntity, "limit must be a positive number")
			return
		}
		limit = n
	}

	s.mu.Lock()
	data, err := os.ReadFile(s.ScanLog)
	s.mu.Unlock()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	records := []ScanRecord{}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := len(lines) - 1; i >= 0 && len(records) < limit; i-- {
		var rec ScanRecord
		if json.Unmarshal([]byte(lines[i]), &rec) == nil {
			records = append(records, rec)
		}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) appendEvent(rec ScanRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.EventsFile), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.EventsFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() == 0 {
		if _, err := fmt.Fprintln(f, eHeader); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(f, "%s,%s,%s,%s\n", rec.Happened, rec.Arrived, rec.Guard, rec.TagID)
	return err
}

func (s *Server) appendLog(rec ScanRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.ScanLog), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.ScanLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(rec) // Encode adds the newline
}

// loadTags reads T (tag_id,checkpoint) the same way TagRegistry.java does:
// skip the header and any line without two columns. It is re-read on every
// scan, so tags added to the file work without a restart.
func loadTags(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tags := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for _, line := range lines[1:] {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		tags[strings.ToUpper(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}
	return tags, nil
}

// parseTime accepts a time with a zone (the phone sends UTC, e.g.
// 2026-10-20T08:02:00.123Z) or without one (taken as this computer's local time).
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02T15:04:05.999999999", s, time.Local)
}

// toJavaTime converts to this computer's local time and drops the zone and
// fractions of a second, which is what LocalDateTime.parse reads.
func toJavaTime(t time.Time) string {
	return t.In(time.Local).Format(javaTime)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
