import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

/**
 * Runs Algorithm 1 from start to finish. Each step calls one teammate's class.
 *
 * Compile:  javac -d out src/*.java
 * Run:      java -cp out Main inputs/R_route.txt inputs/T_tags.csv inputs/E_events.csv
 */
public class Main {

    public static void main(String[] args) throws IOException {
        if (args.length != 3) {
            System.out.println("Usage: java -cp out Main <route file R> <tag file T> <events file E>");
            return;
        }
        List<String> output = run(args[0], args[1], args[2]);
        if (output.isEmpty()) {
            System.out.println("(no output yet - fill in the TODOs)");
        }
        for (String line : output) {
            System.out.println(line);
        }
    }

    /**
     * Runs the whole algorithm and returns every line it prints, in order:
     * first the warnings, then one line per walk.
     * Returning the lines (instead of printing them right away) lets TestRunner
     * compare them with the expected output.
     */
    public static List<String> run(String routeFile, String tagFile, String eventFile) throws IOException {
        List<String> output = new ArrayList<>();

        RouteSettings route = RouteReader.read(routeFile);               // line 3   (Brandon)
        TagRegistry tags = TagRegistry.read(tagFile);                    // line 3   (Kevin)

        List<Event> events = EventReader.read(eventFile, tags, output);  // lines 4-8   (Sean)
        EventReader.sortByHappened(events);                              // line 9      (Sean)
        EventReader.markSyncedLate(events);                              // lines 10-14 (Sean)

        for (int i = 0; i < route.walkStarts.size(); i++) {              // line 15
            WalkResult walk = WalkChecker.check(route, i, events);       // lines 16-29 (Sean)
            FlagChecker.addFlags(walk);                                  // lines 30-36 (Johnny)
            output.add(ReportPrinter.format(walk));                      // line 37     (Johnny)
        }                                                                // line 38
        return output;
    }
}
