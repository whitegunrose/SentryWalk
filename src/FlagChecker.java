
import java.time.Duration;
import java.util.List;

/**
 * Adds LONG GAP and SYNCED LATE flags to one walk (algorithm lines 30-36).
 * Owner: Ryan
 */
public class FlagChecker {

    /** Line 32: a gap this long or longer between two scans in a row is a LONG GAP. */
    public static final int LONG_GAP_MINUTES = 15;

    public static void addFlags(WalkResult walk) {
        List<Event> s = walk.scans;   // already sorted, repeats removed

        // Lines 31-35: Check the gap between each pair of consecutive scans.
        for (int k = 0; k < s.size() - 1; k++) {
            Event first = s.get(k);
            Event second = s.get(k + 1);

            long gap = Duration.between(
                first.happened,
                second.happened
            ).toMinutes();

            if (gap >= LONG_GAP_MINUTES) {
                walk.flags.add(
                    "LONG GAP " + first.checkpoint + "-"
                    + second.checkpoint + " " + gap + " min"
                );
            }
        }

        // Line 36: Add SYNCED LATE once, after all LONG GAP flags.
        for (Event event : s) {
            if (event.syncedLate) {
                walk.flags.add("SYNCED LATE");
                break;
            }
        }
    }
}
