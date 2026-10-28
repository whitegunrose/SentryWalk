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

        for(int i = 1; i < lines.size(); i++) {
            int lineNumber = i+1;

            // -1 keeps empty fields, so "a,b,," counts as 4 parts
            String[] parts = lines.get(i).split(",", -1);

            //Line 5: missing a column (an empty line counts too)
            if(parts.length < 4) {
                output.add("warning: line " + lineNumber);
                continue;
            }

            //Line 5: a time that cannot be read
            LocalDateTime happened;
            LocalDateTime arrived;
            try {
                happened = LocalDateTime.parse(parts[0].trim());
                arrived = LocalDateTime.parse(parts[1].trim());
            } catch(DateTimeParseException ex) {
                output.add("warning: line " + lineNumber);
                continue;
            }

            //Line 5: tag ID not in T
            String tagID = parts[3];
            if(!tags.contains(tagID)) {
                output.add("warning: line " + lineNumber);
                continue;
            }

            //A good line: turn it into an Event
            Event e = new Event();
            e.lineNumber = lineNumber;
            e.happened = happened;
            e.arrived = arrived;
            e.guard = parts[2];
            e.tagId = tagID;
            e.checkpoint = tags.checkpointFor(tagID);
            events.add(e);
        }

        return events;
    }

    /** Line 9: sort by time happened, earliest first (not by time arrived). */
    public static void sortByHappened(List<Event> events) {
        events.sort(Comparator.comparing(e -> e.happened));
    }

    /** Lines 10-14: mark every event that arrived more than 5 minutes after it happened. */
    public static void markSyncedLate(List<Event> events) {
        Duration limit = Duration.ofMinutes(SYNCED_LATE_MINUTES);
        for(Event e : events) {
            Duration gap = Duration.between(e.happened, e.arrived);
            // "> 5 min": exactly 5min is NOT late. Anything after is late
            if(gap.compareTo(limit) > 0) {
                e.syncedLate = true;
            }
        }
    }
}
