# SentryWalk server (Go)

Receives scans from the phone app and writes them to `data/E_live.csv`, the events file E the Java checker reads.
Uses only the Go standard library, so there's nothing to install except Go 1.22 or later (`brew install go`, or https://go.dev/dl).

## Run

```
cd server
go run .
```

It listens on port 8000 on every network interface, so phones on the same Wi-Fi can reach it at `http://<your computer's IP>:8000`.
On a Mac, `ipconfig getifaddr en0` prints that IP. Allow incoming connections if macOS asks.

Options (the defaults assume you're in `server/`):

```
go run . -addr :8000 -tags ../inputs/T_tags.csv -events ../data/E_live.csv -log ../data/scans.jsonl
```

To build a single program file instead: `go build -o sentrywalk-server .`, then run `./sentrywalk-server`.

## Test

```
go test ./...          # all tests
go test -v -run Dup    # only tests whose name contains "Dup", with details
go vet ./...           # common mistakes
gofmt -l .             # lists files that need formatting (empty = fine)
```

Try it without a phone:

```
curl -X POST localhost:8000/scans \
  -d '{"scan_id":"test-1","tag_id":"04A19F22","guard_id":"G1","scanned_at":"2026-10-20T01:02:00"}'
```

## API

| Endpoint | What it does |
|---|---|
| `POST /scans` | Saves one scan. Body: `scan_id`, `tag_id`, `guard_id`, `scanned_at` (required), `ndef_url`, `tag_type` (optional). Returns `{"status":"saved"/"duplicate","checkpoint":"A","known_tag":true}`. |
| `GET /scans?limit=50` | Most recent scans first, from `data/scans.jsonl` |
| `GET /health` | `{"ok":true}` |
| `GET /scan?tag=04A19F22` | The link written on a tag. Saves a scan for the guard in this browser's cookie, then shows the result page. No guard set yet → sends the browser to `/guard` first. |
| `GET /guard`, `POST /guard` | Page where a guard sets their ID once per phone (kept in a cookie for a year) |
| `GET /scan/ok` | Result page after a tap. Shows the scan, saves nothing. |
| `GET /dashboard` | Admin dashboard for supervisors (`/` redirects here) |
| `GET /dashboard/data` | Everything the dashboard shows: recent scans, walks, minimum times, checkpoints |
| `POST /dashboard/min-times` | Sets a minimum walk time. Body: `{"from":"A","to":"B","minutes":5}`. `minutes: 0` removes it. |

## Tag links (test without the app)

Write a URL record to each tag (e.g. with the NFC Tools app: Write → Add a record → URL):

```
http://<your computer's IP>:8000/scan?tag=04A19F22
```

Use a tag ID from `inputs/T_tags.csv`, so the scan gets a checkpoint. Tapping the tag with an unlocked Android phone opens the link in the browser; an iPhone XS or newer shows a banner to tap instead. The first tap on a phone asks for the guard ID, then records the scan.

The result page also shows the guard's progress on the current walk (one guard's scans inside one clock hour, the same as the dashboard): how many of the route's checkpoints are done, which are left, and "Round complete!" when none are. Tapping a checkpoint already scanned on this walk shows **Already scanned** (the scan is still saved). The checkpoint list comes from the `checkpoints=` line of `inputs/R_route.txt` (`-route` to change); with no route file the page shows no progress.

Link scans go to the same `E_live.csv` and `scans.jsonl` as app scans (`tag_type` is `ndef-link`). `happened` and `arrived` are both the server's time, because the browser sends the scan right away. Differences from the app: no offline queue, and anyone who knows a link can type it in (NTAG 424 DNA signed links fix that later).

Bad input gets a 400 (not JSON) or 422 (missing or invalid field) with `{"error": "..."}`.

## Dashboard

Open `http://localhost:8000/dashboard` on the server computer (or the Tailscale address from another device). It refreshes every 5 seconds and shows:

- **Recent scans:** the same scans the terminal prints, newest first.
- **Walks:** grouped by hour (the 1:00 PM walk, the 2:00 PM walk, ...), newest first, then one card per guard. Each card shows a progress bar ("3 of 5 checkpoints"), a chip for every route checkpoint (green = scanned, dashed = still to scan, red dashed = missed once the hour is over), a COMPLETE / IN PROGRESS / INCOMPLETE badge, and the guard's scans in order. Until guards have scheduled shifts, a walk is simply one guard's scans inside one clock hour; `walkStart` in `dashboard.go` is the one place to change when schedules arrive.
- **Walk progress on each scan:** the Recent scans list shows where the guard's walk stood right after that scan, e.g. "Checkpoint 3 of 5", with the just-scanned checkpoint highlighted in blue. A repeat scan shows "Already scanned" and does not count. Checkpoints can be scanned in any order for now; the route comes from the `checkpoints=` line of `inputs/R_route.txt`, and with no route file no progress is shown.
- **Minimum walk times:** a supervisor sets the fewest minutes it should take to walk between two checkpoints. The rule works in both directions (A↔B). Inside one walk, if a guard scans one checkpoint and then the other sooner than that, the scan and the walk are flagged **TOO FAST**. Exactly the minimum is fine. Rules are saved in `data/min_times.csv` (`-mintimes` to change).

The dashboard has no login yet, so anyone who can reach the server can open it and change minimum times.

## Files

| File | What it does |
|---|---|
| `main.go` | Reads the command-line options and starts the server |
| `server.go` | Endpoints, input checks, writing E and the scan log, reading T |
| `server_test.go` | Tests (each test runs a server on files in a temp folder) |
| `tap.go` | Tag links: `/scan`, `/guard`, and the result page |
| `tap_test.go` | Tests for the tag links |
| `dashboard.go` | Dashboard endpoints, walks, TOO FAST check, minimum times file |
| `dashboard.html` | The dashboard page (built into the program with `go:embed`) |
| `dashboard_test.go` | Tests for the dashboard and TOO FAST |
| `progress.go` | Reads the route and works out a guard's remaining checkpoints for the tag-link page |
| `progress_test.go` | Tests for remaining checkpoints and Already scanned |

## Rules it follows

- **Times:** the phone sends UTC. The server converts it to this computer's local time and drops the zone and fractions of a second (`2026-10-20T01:02:00`), because that's what `LocalDateTime.parse` reads.
- **`arrived`:** the time the server received the scan, so scans the phone queued offline are flagged SYNCED LATE by the Java checker.
- **Unknown tags:** still written to E, so the Java checker prints `warning: line N` for them, the same as the simulation.
- **Duplicates:** the phone may resend a queued scan. A repeated `scan_id` is accepted but not written twice, even after a restart.
- **Tag list:** `inputs/T_tags.csv` is re-read on every scan, so new tags work without a restart.
