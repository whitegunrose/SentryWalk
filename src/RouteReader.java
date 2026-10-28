import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.LocalDateTime;
import java.util.List;

/**
 * Reads the route settings file R (algorithm line 3).
 * Owner: Brandon
 *
 * R looks like this (lines starting with # are comments):
 *   checkpoints=A,B,C,D,E
 *   walk_starts=2026-10-20T01:00,2026-10-20T02:00,...
 *   shift_end=2026-10-20T06:00
 *   time_limit_minutes=50
 */
public class RouteReader {

    public static RouteSettings read(String path) throws IOException {
        RouteSettings route = new RouteSettings();
        List<String> lines = Files.readAllLines(Path.of(path));

        // TODO (Brandon):
        // 1. Loop over the lines. Skip empty lines and lines that start with "#".
        // 2. Split each line into key and value:  String[] kv = line.split("=", 2);
        // 3. Depending on the key, fill in the matching field of route:
        //      "checkpoints"        -> split the value on "," and add each one to route.checkpoints
        //      "walk_starts"        -> split on ",", LocalDateTime.parse(...) each, add to route.walkStarts
        //      "shift_end"          -> route.shiftEnd = LocalDateTime.parse(value)
        //      "time_limit_minutes" -> route.timeLimitMinutes = Integer.parseInt(value)
        // Tip: call .trim() on keys and values so extra spaces don't break anything.

        return route;
    }
}
