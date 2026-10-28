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

        for(String line : lines) {
            line = line.trim();

            // Skip empty lines and comments
            if(line.isEmpty() || line.startsWith("#")) {
                continue;
            }

            // Split "key=value" into its two parts
            String[] keyValue = line.split("=", 2);
            if(keyValue.length < 2) {
                continue;
            }

            String key = keyValue[0].trim();
            String value = keyValue[1].trim();

            if(key.equals("checkpoints")) {
                for(String checkpoint : value.split(",")) {
                    route.checkpoints.add(checkpoint.trim());
                }
            }
            else if(key.equals("walk_starts")) {
                for(String time : value.split(",")) {
                    route.walkStarts.add(LocalDateTime.parse(time.trim()));
                }
            }
            else if(key.equals("shift_end")) {
                route.shiftEnd = LocalDateTime.parse(value);
            }
            else if(key.equals("time_limit_minutes")) {
                route.timeLimitMinutes = Integer.parseInt(value);
            }
        }

        return route;
    }
}
