import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

/**
 * Everything we know about one walk after checking it.
 * Created by WalkChecker (lines 16-29), flags added by FlagChecker (lines 30-36),
 * printed by ReportPrinter (line 37).
 * Owner: whole team (shared class - agree on it together before changing it)
 */
public class WalkResult {
    public static final String COMPLETE = "complete";
    public static final String LATE = "late";
    public static final String INCOMPLETE = "incomplete";
    public static final String MISSED = "missed";

    /** Walk start time w. */
    public LocalDateTime walkStart;

    /** S: this walk's scans, sorted by time happened, repeats removed (lines 18-19). */
    public List<Event> scans = new ArrayList<>();

    /** One of COMPLETE, LATE, INCOMPLETE, MISSED (lines 21-29). */
    public String result;

    /** Checkpoints not scanned, in route order (line 20). */
    public List<String> missing = new ArrayList<>();

    /** e.g. "LONG GAP B-C 15 min", "SYNCED LATE" (lines 30-36). */
    public List<String> flags = new ArrayList<>();
}
