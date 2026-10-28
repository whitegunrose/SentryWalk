import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;

/**
 * Runs every test file in tests/data/ and compares the output with tests/expected/.
 * Owner: Ryan
 *
 * Run from the project folder:  java -cp out TestRunner
 * Goal by Oct 29:  "10 of 10 tests passed"
 */
public class TestRunner {

    public static void main(String[] args) throws IOException {
        File[] files = new File("tests/data").listFiles((dir, name) -> name.endsWith(".csv"));
        Arrays.sort(files);

        // TODO (Ryan): for each file f in files:
        //   1. String name = f.getName().replace(".csv", "");          e.g. "04_late"
        //   2. List<String> expected = Files.readAllLines(Path.of("tests", "expected", name + ".txt"));
        //   3. List<String> actual = Main.run("inputs/R_route.txt", "inputs/T_tags.csv", f.getPath());
        //   4. Compare the two lists line by line (they may have different lengths).
        //      All lines equal -> print "PASS  " + name
        //      Otherwise       -> print "FAIL  " + name + " line N: expected \"...\" but got \"...\""
        //                         for the FIRST line that differs, so it's easy to debug.
        //   5. At the end print: passed + " of " + files.length + " tests passed"

        System.out.println("TestRunner is not written yet (" + files.length + " test files found)");
    }
}
