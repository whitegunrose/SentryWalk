import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * The tag list T: tag ID -> checkpoint (algorithm line 3).
 * Owner: Kevin
 *
 * T looks like this (line 1 is the header):
 *   tag_id,checkpoint
 *   04A19F22,A
 */
public class TagRegistry {

    private final Map<String, String> tagToCheckpoint = new HashMap<>();

    public static TagRegistry read(String path) throws IOException {
        TagRegistry tags = new TagRegistry();
        List<String> lines = Files.readAllLines(Path.of(path));

        // Line 3: tag ID -> checkpoint. Start at index 1 to skip the header
        for(int i = 1; i < lines.size(); i++) {
            String[] parts = lines.get(i).split(",");
            if(parts.length < 2) {
                // skip blank or broken lines
                continue;
            }
            String tagId = parts[0].trim();
            String checkpoint = parts[1].trim();
            tags.tagToCheckpoint.put(tagId, checkpoint);
        }

        return tags;
    }

    /** True if this tag ID is in T. Used by EventReader for line 5. */
    public boolean contains(String tagId) {
        return tagToCheckpoint.containsKey(tagId);
    }

    /** The checkpoint for this tag ID, e.g. "C". */
    public String checkpointFor(String tagId) {
        return tagToCheckpoint.get(tagId);
    }
}
