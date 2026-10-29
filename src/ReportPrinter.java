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

    /** "H:mm" prints 1:00, not 01:00. */
    private static final DateTimeFormatter HOUR = DateTimeFormatter.ofPattern("H:mm");

    public static String format(WalkResult walk) {
        // TODO (Johnny):
        // 1. Start with walk.walkStart.format(HOUR) + " walk: " + walk.result
        // 2. If the result is WalkResult.INCOMPLETE, add " (missing " + String.join(",
        // ", walk.missing) + ")"
        // 3. For each flag in walk.flags, add ", " + flag
        // Tip: a StringBuilder makes this easy.

        // StringBuilder really did make this easy wow
        StringBuilder sb = new StringBuilder();

        sb.append(walk.walkStart.format(HOUR))
                .append(" walk: ")
                .append(walk.result);

        if (walk.result == WalkResult.INCOMPLETE) {
            sb.append(" (missing ")
                    .append(String.join(", ", walk.missing))
                    .append(")");
        }

        for (String flag : walk.flags) {
            sb.append(", ")
                    .append(flag);
        }

        return sb.toString();
    }
}
