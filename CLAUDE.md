# SentryWalk

Security guard patrol system (COMP 490, CSUN). Guards tap NFC stickers at checkpoints; a Go server saves scans; supervisors watch a web dashboard; a Java checker judges whole shifts.

## Layout

| Path | Part |
|---|---|
| `src/`, `tests/`, `inputs/` | Java checker: route R, tags T, events E → walk verdicts |
| `server/` | Go server (standard library only): scans API, tag links, walk progress, admin dashboard |
| `mobile/` | Flutter app (NFC scanning, offline queue) |
| `data/` | Live data written by the server (gitignored) |
| `aidlc-docs/` | Local planning docs: plans, stories, prompts, session state (gitignored) |

## Run

```bash
cd server && go run .            # http://localhost:8000/dashboard
javac -d out src/*.java && java -cp out Main inputs/R_route.txt inputs/T_tags.csv data/E_live.csv
cd mobile && flutter run         # iPhone needs the paid Apple account for NFC
```

## Test (run before every commit)

```bash
cd server && gofmt -l . && go vet ./... && go test ./...
javac -d out src/*.java && java -cp out TestRunner
cd mobile && flutter analyze && flutter test
```

## Rules for working on this repo

- Work on a `feature/…` branch; merge into `main` by pull request.
- Plan before work: write the plan in `aidlc-docs/plans/`, get approval, then do one step at a time. Log prompts in `aidlc-docs/prompts.md`; save progress to `aidlc-docs/session-state.md`.
- Roadmap: `aidlc-docs/plans/feature_roadmap_plan.md`, stories in `aidlc-docs/story-artifacts/`, and the "SentryWalk Feature Roadmap" page. Keep all three in sync when a story starts, ships, or changes.
- Claude never commits, pushes, tags, merges or opens pull requests. It gives the commands for the user to run. Commit messages carry no `Co-Authored-By` or "Generated with" lines.
- `inputs/` and the formats of E, T and R are shared by Java and Go: change them in both, with tests.
- Study guide: "How SentryWalk Works" (Claude Doc). Revise it at the end of each roadmap phase.
