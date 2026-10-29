
import java.io.IOException;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;

public class DemoRunner {

    private static final DateTimeFormatter TIME_FORMAT =
            DateTimeFormatter.ofPattern("HH:mm");

    private static final DateTimeFormatter DATE_TIME_FORMAT =
            DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm");

    public static void main(String[] args) throws IOException {
        String routeFile = args.length > 0
                ? args[0] : "inputs/R_route.txt";

        String tagFile = args.length > 1
                ? args[1] : "inputs/T_tags.csv";

        String eventFile = args.length > 2
                ? args[2] : "inputs/E_events.csv";

        // Load the actual input files using the existing project classes.
        RouteSettings route = RouteReader.read(routeFile);
        TagRegistry tags = TagRegistry.read(tagFile);

        List<String> warnings = new ArrayList<>();
        List<Event> events = EventReader.read(eventFile, tags, warnings);

        EventReader.sortByHappened(events);
        EventReader.markSyncedLate(events);

        int complete = 0;
        int late = 0;
        int incomplete = 0;
        int missed = 0;
        int totalFlags = 0;

        printHeader("SENTRYWALK | SHIFT MONITOR");

        System.out.println("Route file : " + routeFile);
        System.out.println("Tag file   : " + tagFile);
        System.out.println("Events file: " + eventFile);
        System.out.println("Checkpoints: " + String.join(" -> ", route.checkpoints));
        System.out.println("Walk limit : " + route.timeLimitMinutes + " minutes");

        if (route.shiftEnd != null) {
            System.out.println("Shift ends : " +
                    route.shiftEnd.format(DATE_TIME_FORMAT));
        }

        System.out.println();

        // Display any warnings produced while reading the event file.
        if (!warnings.isEmpty()) {
            System.out.println("INPUT WARNINGS");
            for (String warning : warnings) {
                System.out.println("  ! " + warning);
            }
            System.out.println();
        }

        // Run the same walk-checking algorithm used by Main.
        for (int i = 0; i < route.walkStarts.size(); i++) {
            WalkResult walk = WalkChecker.check(route, i, events);
            FlagChecker.addFlags(walk);

            switch (walk.result) {
                case WalkResult.COMPLETE -> complete++;
                case WalkResult.LATE -> late++;
                case WalkResult.INCOMPLETE -> incomplete++;
                case WalkResult.MISSED -> missed++;
            }

            totalFlags += walk.flags.size();

            printWalk(i + 1, walk, route.checkpoints.size());
        }

        int totalWalks = route.walkStarts.size();
        int problemWalks = late + incomplete + missed;

        printHeader("SHIFT SUMMARY");

        System.out.println("Total walks      : " + totalWalks);
        System.out.println("Complete         : " + complete);
        System.out.println("Late             : " + late);
        System.out.println("Incomplete       : " + incomplete);
        System.out.println("Missed           : " + missed);
        System.out.println("Walks with issues: " + problemWalks);
        System.out.println("Total flags      : " + totalFlags);

        if (totalWalks > 0) {
            double completionRate = complete * 100.0 / totalWalks;
            System.out.printf("Completion rate  : %.1f%%%n", completionRate);
        }

        System.out.println();
        System.out.println("Demo finished.");
    }

    private static void printWalk(
            int walkNumber, WalkResult walk, int checkpointCount) {

        System.out.println("--------------------------------------------------");
        System.out.println("WALK " + walkNumber + " | " +
                walk.walkStart.format(DATE_TIME_FORMAT));

        System.out.println("Result: " + walk.result.toUpperCase());

        System.out.println();
        System.out.println("CHECKPOINT SCANS");

        if (walk.scans.isEmpty()) {
            System.out.println("  No valid checkpoint scans recorded.");
        } else {
            for (Event event : walk.scans) {
                String status = event.syncedLate
                        ? " [SYNCED LATE]"
                        : "";

                System.out.printf("  %-12s %s%s%n",
                        event.checkpoint,
                        event.happened.format(TIME_FORMAT),
                        status);
            }
        }

        System.out.printf("%nCheckpoints visited: %d/%d%n",
                walk.scans.size(), checkpointCount);

        if (!walk.missing.isEmpty()) {
            System.out.println("Missing checkpoints: " +
                    String.join(", ", walk.missing));
        }

        if (!walk.flags.isEmpty()) {
            System.out.println("FLAGS");
            for (String flag : walk.flags) {
                System.out.println("  ! " + flag);
            }
        } else {
            System.out.println("Flags: None");
        }

        System.out.println();
    }

    private static void printHeader(String title) {
        System.out.println("==================================================");
        System.out.println("  " + title);
        System.out.println("==================================================");
    }
}
