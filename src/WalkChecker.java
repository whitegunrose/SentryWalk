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

        // TODO (Sean), lines 18-19: build S in walk.scans
        //   For each event e: if w <= e.happened < windowEnd, and its checkpoint
        //   hasn't been seen yet in this walk, add it to walk.scans.
        //   "w <= happened"       ->  !e.happened.isBefore(w)
        //   "happened < windowEnd" ->  e.happened.isBefore(windowEnd)
        //   Use a Set<String> of seen checkpoints: seen.add(...) returns false for a repeat.
        Set<String> seen = new HashSet<>();

        // TODO (Sean), line 20: for each checkpoint in route.checkpoints (route order),
        //   if it is not in seen, add it to walk.missing.

        // TODO (Sean), lines 21-29: set walk.result, checking in this order:
        //   S empty, or first scan >= startDeadline  -> WalkResult.MISSED
        //   walk.missing not empty                   -> WalkResult.INCOMPLETE
        //   last scan >= walkDeadline                -> WalkResult.LATE   (exactly at the deadline is late)
        //   otherwise                                -> WalkResult.COMPLETE
        //   "a >= b" for times  ->  !a.isBefore(b)
        walk.result = "TODO";

        return walk;
    }
}
