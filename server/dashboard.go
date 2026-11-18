package main

// Admin dashboard for managers and supervisors.
//
//	GET  /dashboard             the page (dashboard.html)
//	GET  /dashboard/data        everything the page shows, as JSON
//	POST /dashboard/min-times   set or remove a minimum walk time
//
// Walks: until guards have scheduled shifts, a walk is all of one guard's
// scans inside one clock hour (walk 1 at 10:00 PM, walk 2 at 11:00 PM, ...).
// When schedules arrive, walkStart is the one place to change.
//
// Too fast: a supervisor sets the minimum minutes it should take to walk
// between two checkpoints (A-B 5 = at least 5 minutes from A to B or B to A).
// Inside one walk, if a guard's scan of one checkpoint follows their scan of
// the other sooner than that, the walk is flagged TOO FAST.
//
// Minimum times are kept in data/min_times.csv (from,to,minutes).

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed dashboard.html
var dashboardHTML []byte

// maxRecentScans is how many scans the page lists.
const maxRecentScans = 200

// MinTime is one supervisor rule: walking between From and To should take
// at least Minutes. From is always the one that sorts first, so A-B and B-A
// are the same rule.
type MinTime struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Minutes int    `json:"minutes"`
}

// DashScan is one scan as the dashboard shows it.
type DashScan struct {
	ScanRecord
	WalkStart string `json:"walk_start"` // start of the hour this scan's walk belongs to
	TooFast   bool   `json:"too_fast"`   // reached this checkpoint too soon after the previous one
}

// Walk is one guard's scans inside one hour.
type Walk struct {
	Guard   string     `json:"guard"`
	Start   string     `json:"start"`
	Scans   []DashScan `json:"scans"` // in the order they happened
	TooFast bool       `json:"too_fast"`
	Flags   []string   `json:"flags"`
}

// DashboardData is the JSON behind the page.
type DashboardData struct {
	Scans       []DashScan `json:"scans"` // newest first
	Walks       []Walk     `json:"walks"` // newest first
	MinTimes    []MinTime  `json:"min_times"`
	Checkpoints []string   `json:"checkpoints"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(dashboardHTML)
}

func (s *Server) handleDashboardData(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	records, err := readScanLog(s.ScanLog)
	var rules []MinTime
	if err == nil {
		rules, err = loadMinTimes(s.MinTimesFile)
	}
	s.mu.Unlock()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	checkpoints := []string{}
	if tags, err := loadTags(s.TagsFile); err == nil {
		set := map[string]bool{}
		for _, cp := range tags {
			if cp != "" && !set[cp] {
				set[cp] = true
				checkpoints = append(checkpoints, cp)
			}
		}
		sort.Strings(checkpoints)
	}

	scans, walks := buildWalks(records, rules)

	// Newest first, and only the most recent scans.
	for i, j := 0, len(scans)-1; i < j; i, j = i+1, j-1 {
		scans[i], scans[j] = scans[j], scans[i]
	}
	if len(scans) > maxRecentScans {
		scans = scans[:maxRecentScans]
	}

	writeJSON(w, http.StatusOK, DashboardData{
		Scans:       scans,
		Walks:       walks,
		MinTimes:    rules,
		Checkpoints: checkpoints,
	})
}

// handleMinTime sets one rule. Body: {"from":"A","to":"B","minutes":5}.
// minutes 0 removes the rule.
func (s *Server) handleMinTime(w http.ResponseWriter, r *http.Request) {
	var in MinTime
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "body is not valid JSON: "+err.Error())
		return
	}
	in.From = strings.TrimSpace(in.From)
	in.To = strings.TrimSpace(in.To)
	switch {
	case in.From == "" || in.To == "":
		writeError(w, http.StatusUnprocessableEntity, "from and to are required")
		return
	case in.From == in.To:
		writeError(w, http.StatusUnprocessableEntity, "from and to must be different checkpoints")
		return
	case strings.ContainsAny(in.From+in.To, ",\"\r\n"):
		writeError(w, http.StatusUnprocessableEntity, "checkpoint names cannot contain commas, quotes or line breaks")
		return
	case in.Minutes < 0 || in.Minutes > 24*60:
		writeError(w, http.StatusUnprocessableEntity, "minutes must be between 0 and 1440")
		return
	}
	in.From, in.To = orderPair(in.From, in.To)

	s.mu.Lock()
	defer s.mu.Unlock()
	rules, err := loadMinTimes(s.MinTimesFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	kept := []MinTime{}
	for _, m := range rules {
		if m.From != in.From || m.To != in.To {
			kept = append(kept, m)
		}
	}
	if in.Minutes > 0 {
		kept = append(kept, in)
	}
	if err := saveMinTimes(s.MinTimesFile, kept); err != nil {
		writeError(w, http.StatusInternalServerError, "saving minimum times: "+err.Error())
		return
	}

	if in.Minutes == 0 {
		log.Printf("[dashboard] min time %s-%s removed", in.From, in.To)
	} else {
		log.Printf("[dashboard] min time %s-%s set to %d min", in.From, in.To, in.Minutes)
	}
	writeJSON(w, http.StatusOK, sortedMinTimes(kept))
}

// buildWalks groups scans into walks (one guard, one hour) and marks the ones
// that are too fast. It returns every scan, oldest first, and the walks,
// newest first.
func buildWalks(records []ScanRecord, rules []MinTime) ([]DashScan, []Walk) {
	minutes := map[[2]string]int{}
	for _, m := range rules {
		minutes[[2]string{m.From, m.To}] = m.Minutes
	}

	type parsed struct {
		DashScan
		at time.Time
	}
	all := []parsed{}
	for _, rec := range records {
		at, err := time.ParseInLocation(javaTime, rec.Happened, time.Local)
		if err != nil {
			continue
		}
		all = append(all, parsed{DashScan: DashScan{ScanRecord: rec, WalkStart: walkStart(at).Format(javaTime)}, at: at})
	}
	// Oldest first. Stable, so scans at the same second keep log order.
	sort.SliceStable(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })

	type key struct{ guard, start string }
	walkIndex := map[key]int{}
	walks := []Walk{}
	lastKnown := map[key]int{} // index in all of the walk's previous scan with a checkpoint
	for i := range all {
		sc := &all[i]
		k := key{sc.Guard, sc.WalkStart}
		wi, ok := walkIndex[k]
		if !ok {
			wi = len(walks)
			walkIndex[k] = wi
			walks = append(walks, Walk{Guard: sc.Guard, Start: sc.WalkStart, Flags: []string{}})
		}

		if sc.Checkpoint != nil {
			if pi, ok := lastKnown[k]; ok {
				prev := all[pi]
				from, to := *prev.Checkpoint, *sc.Checkpoint
				if from != to {
					limit, has := minutes[orderMin(from, to)]
					gap := sc.at.Sub(prev.at)
					if has && gap < time.Duration(limit)*time.Minute {
						sc.TooFast = true
						walks[wi].TooFast = true
						walks[wi].Flags = append(walks[wi].Flags,
							fmt.Sprintf("TOO FAST %s-%s %s (min %d min)", from, to, formatGap(gap), limit))
					}
				}
			}
			lastKnown[k] = i
		}
		walks[wi].Scans = append(walks[wi].Scans, sc.DashScan)
	}

	scans := make([]DashScan, len(all))
	for i := range all {
		scans[i] = all[i].DashScan
	}
	// Walks were made oldest first; show newest first.
	sort.SliceStable(walks, func(i, j int) bool {
		if walks[i].Start != walks[j].Start {
			return walks[i].Start > walks[j].Start
		}
		return walks[i].Guard < walks[j].Guard
	})
	return scans, walks
}

// walkStart is the start of the walk a scan belongs to: the top of its hour.
// Replace this with a lookup in the guard's schedule once shifts exist.
func walkStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// formatGap prints a gap as minutes and seconds, e.g. 2m05s.
func formatGap(d time.Duration) string {
	sec := int(d / time.Second)
	return fmt.Sprintf("%dm%02ds", sec/60, sec%60)
}

func orderPair(a, b string) (string, string) {
	if b < a {
		return b, a
	}
	return a, b
}

func orderMin(a, b string) [2]string {
	x, y := orderPair(a, b)
	return [2]string{x, y}
}

// readScanLog reads every record in scans.jsonl, oldest first. A missing file
// means no scans yet.
func readScanLog(path string) ([]ScanRecord, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []ScanRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	records := []ScanRecord{}
	for _, line := range strings.Split(string(data), "\n") {
		var rec ScanRecord
		if strings.TrimSpace(line) != "" && json.Unmarshal([]byte(line), &rec) == nil {
			records = append(records, rec)
		}
	}
	return records, nil
}

// loadMinTimes reads min_times.csv (from,to,minutes). A missing file means
// no rules yet.
func loadMinTimes(path string) ([]MinTime, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return []MinTime{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	rules := []MinTime{}
	for i, row := range rows {
		if i == 0 || len(row) < 3 { // header or bad line
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(row[2]))
		if err != nil || n <= 0 {
			continue
		}
		from, to := orderPair(strings.TrimSpace(row[0]), strings.TrimSpace(row[1]))
		rules = append(rules, MinTime{From: from, To: to, Minutes: n})
	}
	return sortedMinTimes(rules), nil
}

func saveMinTimes(path string, rules []MinTime) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("from,to,minutes\n")
	for _, m := range sortedMinTimes(rules) {
		fmt.Fprintf(&b, "%s,%s,%d\n", m.From, m.To, m.Minutes)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sortedMinTimes(rules []MinTime) []MinTime {
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].From != rules[j].From {
			return rules[i].From < rules[j].From
		}
		return rules[i].To < rules[j].To
	})
	return rules
}
