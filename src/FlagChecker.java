import java.time.Duration;
import java.util.List;

/**
 * Adds LONG GAP and SYNCED LATE flags to one walk (algorithm lines 30-36).
 * Owner: Johnny
 */
public class FlagChecker {

    /** Line 32: a gap this long or longer between two scans in a row is a LONG GAP. */
    public static final int LONG_GAP_MINUTES = 15;

    public static void addFlags(WalkResult walk) {
        List<Event> s = walk.scans;   // already sorted, repeats removed

        // TODO (Johnny), lines 31-35:
        //   For each pair of scans in a row (s.get(k) and s.get(k + 1)):
        //     long gap = Duration.between(first.happened, second.happened).toMinutes();
        //     (toMinutes() rounds down: 15 min 30 s -> 15)
        //     if gap >= LONG_GAP_MINUTES, add exactly this text to walk.flags:
        //       "LONG GAP " + first.checkpoint + "-" + second.checkpoint + " " + gap + " min"
        //       e.g. "LONG GAP B-C 15 min"

        // TODO (Johnny), line 36:
        //   If any scan in s has syncedLate == true, add "SYNCED LATE" to walk.flags (only once).
        //   Add it AFTER the LONG GAP flags so the order matches the expected output.
    }
}
