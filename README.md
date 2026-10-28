# SentryWalk

Sprint 1: a Java command-line simulation that checks one guard's shift, following **Algorithm 1: Check One Guard's Shift (Version 0)**.
It reads three files (route settings R, tag list T, events E) and prints one line per walk.

Sprint 1 goal (due **Oct 29**): `java -cp out TestRunner` prints **10 of 10 tests passed**.

## Make sure Java is installed

Download the JDK 25 LTS installer for Windows x64, unless the team has agreed on a different version.
Run the installer.
If the installer offers the options, enable adding Java to PATH and setting JAVA_HOME.
Restart your terminal or VS Code.

Check it worked: `java -version` and `javac -version` should both print 25.

## Folder layout

```
SentryWalk/
  README.md
  inputs/
    R_route.txt      R: checkpoints, walk start times, shift end, time limit
    T_tags.csv       T: tag ID -> checkpoint
    E_events.csv     E: a sample shift (same as tests/data/01_sample_shift.csv)
  src/               the Java classes (one per piece below)
  tests/
    data/            E files, one per test (01_sample_shift.csv ... 10_bad_lines.csv)
    expected/        the exact output each E file should print
```

## Compile and run

From the `SentryWalk` folder (the repo root):

```
javac -d out src/*.java
java -cp out Main inputs/R_route.txt inputs/T_tags.csv inputs/E_events.csv
java -cp out TestRunner
```

`javac -d out` puts the compiled `.class` files in an `out/` folder, which git ignores.
The skeleton compiles and runs today. Until the TODOs are filled in, it prints placeholder lines like `1:00 walk: TODO`.

## Classes

| Class | Algorithm lines | Job |
|---|---|---|
| `Main` | 15, 38 | Calls every piece in order (already written) |
| `RouteSettings`, `Event`, `WalkResult` | — | Shared data classes (already written) |
| `RouteReader` | 1, 3 | Read R into a `RouteSettings` |
| `TagRegistry` | 3 | Read T into a tag ID -> checkpoint map |
| `EventReader` | 4–14 | Read E, print warnings, sort by time happened, mark SYNCED LATE |
| `WalkChecker` | 15–29 | Build S for one walk, remove repeats, find missing, decide the result |
| `FlagChecker` | 30–36 | LONG GAP and SYNCED LATE flags |
| `ReportPrinter` | 37 | Turn one walk into its output line |
| `TestRunner` | — | Run every test file and compare with tests/expected/ |

Each unfinished class has `TODO` comments that say what to write, with the algorithm line numbers.
When you add a rule, add a test file to `tests/data/` and its expected output to `tests/expected/`.

## Output format (agreed)

```
warning: line 12
1:00 walk: complete
2:00 walk: incomplete (missing C, D, E)
3:00 walk: complete, LONG GAP B-C 15 min, SYNCED LATE
4:00 walk: missed
5:00 walk: late
```

- Warnings come first, in line-number order. Line 1 of E is the header, so the first scan is line 2.
- Walk times print as `H:mm` (1:00, not 01:00).
- Missing checkpoints are listed in route order.
- LONG GAP minutes are rounded down (15 min 30 s prints as 15).
- Flags come after the result: LONG GAPs first (in scan order), then SYNCED LATE.

## The rules in one place

| Rule | Value | Exactly at the line |
|---|---|---|
| Walk window (line 18) | w ≤ happened < next walk start (shift end for the last walk) | a scan at exactly 2:00 belongs to the 2:00 walk |
| Start deadline (line 21) | w + 5 min | first scan at exactly 1:05 → missed |
| Walk deadline (line 25) | w + 50 min | last scan at exactly 3:50 → late |
| SYNCED LATE (line 11) | arrived − happened > 5 min | exactly 5 min → not SYNCED LATE |
| LONG GAP (line 32) | gap ≥ 15 min | exactly 15 min → LONG GAP |

## The 10 tests

| File | What it tests |
|---|---|
| 01_sample_shift | Everything at once: every result, a repeat, out-of-order line, 3 bad lines |
| 02_complete | A normal walk |
| 03_incomplete | E never scanned |
| 04_late | Last scan exactly at the 50-min deadline |
| 05_long_gap | 15-min gap between B and C |
| 06_late_start | First scan at 1:04:59 (on time) vs. 4:05:00 (missed) |
| 07_synced_late | Arrived 36 min late (flagged) vs. exactly 5 min late (not flagged) |
| 08_duplicates | B and A scanned twice |
| 09_window_boundary | E scanned at exactly 2:00 goes to the 2:00 walk |
| 10_bad_lines | Missing column, bad time, unknown tag, empty line |

## Java you'll use

- Times: `LocalDateTime.parse(...)`, `plusMinutes(...)`, `isBefore(...)`, `Duration.between(a, b).toMinutes()`
- "a ≥ b" for times: `!a.isBefore(b)`
- Bad lines: `line.split(",", -1)` and `try { ... } catch (DateTimeParseException e) { ... }`
- Sorting: `events.sort(Comparator.comparing(e -> e.happened))`
- Repeats: a `HashSet<String>` — `seen.add(x)` returns `false` if `x` was already there
- Tag lookup: a `HashMap<String, String>`
