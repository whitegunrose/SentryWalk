import java.time.LocalDateTime;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * Checks one walk and decides its result (algorithm lines 16-29).
 * Owner: Sean
 */
public class WalkChecker {

    /** Line 16: the guard must start within this many minutes of w. */
    public static final int START_GRACE_MINUTES = 5;

    /**
     * Checks walk number i (0 = the first walk).
     * events are already sorted by time happened, so S comes out sorted too.
     */
    public static WalkResult check(RouteSettings route, int i, List<Event> events) {
        WalkResult walk = new WalkResult();
        LocalDateTime w = route.walkStarts.get(i);
        walk.walkStart = w;

        LocalDateTime startDeadline = w.plusMinutes(START_GRACE_MINUTES);     // line 16
        LocalDateTime walkDeadline = w.plusMinutes(route.timeLimitMinutes);   // line 17
        LocalDateTime windowEnd = route.windowEnd(i);                         // line 18

        // Lines 18 - 19: S = scans with w <= happened < windowEnd, keeping only
        // first scan of each checkpoint. seen.add(...) returns false for a repeat
        Set<String> seen = new HashSet<>();
        for(Event e : events) {
            boolean inWindow = !e.happened.isBefore(w) && e.happened.isBefore(windowEnd);
            if(inWindow && seen.add(e.checkpoint)) {
                walk.scans.add(e);
            }
        }

        // Line 20: missing = checkpoints in R that are not in S, in route order
        for(String checkpoint : route.checkpoints) {
            if(!seen.contains(checkpoint)) {
                walk.missing.add(checkpoint);
            }
        }

        // Lines 21-29: checked in order: missed, incomplete, late, complete
        List<Event> s = walk.scans;
        if (s.isEmpty() || !s.get(0).happened.isBefore(startDeadline)) {
            //Line 21: no scans, or first scan >=  start deadline (late start counts as missed)
            walk.result = WalkResult.MISSED;
        } else if (!walk.missing.isEmpty()) {
            //Line 23
            walk.result = WalkResult.INCOMPLETE;
        } else if (!s.get(s.size() - 1).happened.isBefore(walkDeadline)) {
            //Line 25: last scan >= walk deadline (exactly at the deadline is late)
            walk.result = WalkResult.LATE;
        } else {
            //Line 27
            walk.result = WalkResult.COMPLETE;
        }

        return walk;
    }
}
