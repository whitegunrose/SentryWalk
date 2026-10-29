import java.time.format.DateTimeFormatter;

/**
 * Turns one walk into its output line (algorithm line 37).
 * Owner: Johnny
 *
 * Exact format (the test runner compares character by character):
 * 1:00 walk: complete
 * 2:00 walk: incomplete (missing C, D, E)
 * 3:00 walk: complete, LONG GAP B-C 15 min, SYNCED LATE
 * 4:00 walk: missed
 * 5:00 walk: late
 */
public class ReportPrinter {

    /** "H:mm" prints 1:00, "HH:mm" prints 01:00. */
    private static final DateTimeFormatter HOUR = DateTimeFormatter.ofPattern("H:mm");

    public static String format(WalkResult walk) {

        // StringBuilder really did make this easy wow
        StringBuilder sb = new StringBuilder();
        
        // Format the walk start time and result
        sb.append(walk.walkStart.format(HOUR))
                .append(" walk: ")
                .append(walk.result);
        
        // If the walk is incomplete, append the missing items
        if (WalkResult.INCOMPLETE.equals(walk.result)) {
            sb.append(" (missing ")
                    .append(String.join(", ", walk.missing))
                    .append(")");
        }
        
        // If there are any flags then append
        for (String flag : walk.flags) {
            sb.append(", ")
                    .append(flag);
        }

        return sb.toString();
    }
}
