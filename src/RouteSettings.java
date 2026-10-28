import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

/**
 * Route settings R (algorithm line 1). Filled in by RouteReader.
 * Owner: whole team (shared class - agree on it together before changing it)
 */
public class RouteSettings {
    /** Checkpoints in route order, e.g. [A, B, C, D, E]. */
    public List<String> checkpoints = new ArrayList<>();

    /** Walk start times w, in order, e.g. 1:00, 2:00, 3:00 ... */
    public List<LocalDateTime> walkStarts = new ArrayList<>();

    /** End of the shift. The last walk's window ends here (line 18). */
    public LocalDateTime shiftEnd;

    /** Time limit for a walk (line 17). 50 in our settings. */
    public int timeLimitMinutes;

    /**
     * End of walk i's window (line 18): the next walk's start time,
     * or the shift end for the last walk.
     */
    public LocalDateTime windowEnd(int i) {
        if (i + 1 < walkStarts.size()) {
            return walkStarts.get(i + 1);
        }
        return shiftEnd;
    }
}
