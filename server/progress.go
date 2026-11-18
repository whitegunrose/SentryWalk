package main

// Walk progress for the tag-link result page: which checkpoints this guard
// still has to scan in the current walk.
//
// The phone browser remembers nothing between taps, so the server works it
// out on every tap from the scans already saved in scans.jsonl. A walk is the
// same as on the dashboard: one guard's scans inside one clock hour
// (walkStart in dashboard.go), so a new round starts by itself every hour.

import (
	"errors"
	"os"
	"strings"
	"time"
)

// Progress is where a guard is in their current walk.
type Progress struct {
	Visited        []string // route checkpoints scanned so far, in route order
	Remaining      []string // route checkpoints not scanned yet, in route order
	AlreadyScanned bool     // this checkpoint was already scanned earlier in the walk
}

// loadRoute reads the checkpoints line of the route file R, e.g.
//
//	checkpoints=A,B,C,D,E
//
// the same file the Java checker reads. A missing file means no route, and
// the result page then shows no progress.
func loadRoute(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "checkpoints" {
			continue
		}
		route := []string{}
		for _, cp := range strings.Split(value, ",") {
			if cp = strings.TrimSpace(cp); cp != "" {
				route = append(route, cp)
			}
		}
		return route, nil
	}
	return nil, nil
}

// walkProgress looks at guard's scans in the walk that contains at, up to
// and including at, and reports what is visited and what is left. checkpoint
// is the checkpoint scanned at time at.
func walkProgress(records []ScanRecord, route []string, guard, checkpoint string, at time.Time) Progress {
	start := walkStart(at)
	end := start.Add(time.Hour)

	visited := map[string]bool{}
	timesThisCheckpoint := 0
	for _, rec := range records {
		if rec.Guard != guard || rec.Checkpoint == nil {
			continue
		}
		happened, err := time.ParseInLocation(javaTime, rec.Happened, time.Local)
		if err != nil || happened.Before(start) || !happened.Before(end) || happened.After(at) {
			continue
		}
		visited[*rec.Checkpoint] = true
		if *rec.Checkpoint == checkpoint {
			timesThisCheckpoint++
		}
	}

	p := Progress{Visited: []string{}, Remaining: []string{}, AlreadyScanned: timesThisCheckpoint > 1}
	for _, cp := range route {
		if visited[cp] {
			p.Visited = append(p.Visited, cp)
		} else {
			p.Remaining = append(p.Remaining, cp)
		}
	}
	return p
}
