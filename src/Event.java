import java.time.LocalDateTime;

/**
 * One line of the events file E: a guard scanning a tag.
 * Created by EventReader (lines 4-8).
 * Owner: whole team (shared class - agree on it together before changing it)
 */
public class Event {
    /** Line number in E (line 1 is the header), used in warnings. */
    public int lineNumber;

    /** When the scan happened. Walks are judged by this time. */
    public LocalDateTime happened;

    /** When the scan reached the server. */
    public LocalDateTime arrived;

    public String guard;
    public String tagId;

    /** Checkpoint for this tag, looked up in T (e.g. "C"). */
    public String checkpoint;

    /** Set by EventReader.markSyncedLate (lines 10-14). */
    public boolean syncedLate;

    @Override
    public String toString() {
        // Handy for debugging: System.out.println(event);
        return "line " + lineNumber + ": " + checkpoint + " at " + happened
                + (syncedLate ? " (SYNCED LATE)" : "");
    }
}
