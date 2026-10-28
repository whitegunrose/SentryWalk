import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.time.LocalDateTime;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;

/**
 * Reads the events file E, sorts it and marks SYNCED LATE (algorithm lines 4-14).
 * Owner: Sean
 *
 * E looks like this (line 1 is the header):
 *   happened,arrived,guard,tag_id
 *   2026-10-20T01:02:00,2026-10-20T01:02:01,G1,04A19F22
 */
public class EventReader {

    /** Line 11: a scan that arrives more than this many minutes after it happened is SYNCED LATE. */
    public static final int SYNCED_LATE_MINUTES = 5;

    /**
     * Lines 4-8: reads every line of E after the header and returns the good ones.
     * For each bad line, adds "warning: line N" to output and skips it.
     */
    public static List<Event> read(String path, TagRegistry tags, List<String> output) throws IOException {
        List<Event> events = new ArrayList<>();
        List<String> lines = Files.readAllLines(Path.of(path));

        // TODO (Sean):
        // Loop over the lines starting at index 1 (index 0 is the header, line 1 in the file).
        // The line number for warnings is index + 1.
        //   1. String[] parts = lines.get(i).split(",", -1);
        //      If parts.length < 4, the line is missing a column -> warning, skip.
        //      (An empty line counts as missing columns too.)
        //   2. Parse parts[0] and parts[1] with LocalDateTime.parse(...).
        //      Wrap it in try/catch (DateTimeParseException e) -> warning, skip.
        //   3. If !tags.contains(parts[3]) -> warning, skip.
        //   4. Otherwise make a new Event, fill in every field (lineNumber, happened,
        //      arrived, guard, tagId, checkpoint = tags.checkpointFor(tagId)) and add it to events.
        // Warning text, exactly:  output.add("warning: line " + lineNumber);

        return events;
    }

    /** Line 9: sort by time happened, earliest first (not by time arrived). */
    public static void sortByHappened(List<Event> events) {
        // TODO (Sean): events.sort(Comparator.comparing(e -> e.happened));
    }

    /** Lines 10-14: mark every event that arrived more than 5 minutes after it happened. */
    public static void markSyncedLate(List<Event> events) {
        // TODO (Sean):
        // For each event: Duration gap = Duration.between(e.happened, e.arrived);
        // if gap is MORE than SYNCED_LATE_MINUTES (exactly 5:00 is not late), set e.syncedLate = true.
        // Tip: gap.compareTo(Duration.ofMinutes(SYNCED_LATE_MINUTES)) > 0
    }
}
