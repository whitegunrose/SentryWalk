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

        
int passed = 0;

for (File f : files) {
    String name = f.getName().replace(".csv", "");

    List<String> expected = Files.readAllLines(
        Path.of("tests", "expected", name + ".txt")
    );

    List<String> actual = Main.run(
        "inputs/R_route.txt",
        "inputs/T_tags.csv",
        f.getPath()
    );

    int maxLines = Math.max(expected.size(), actual.size());
    boolean match = true;

    for (int i = 0; i < maxLines; i++) {
        String expectedLine = i < expected.size() ? expected.get(i) : "<no line>";
        String actualLine = i < actual.size() ? actual.get(i) : "<no line>";

        if (!expectedLine.equals(actualLine)) {
            System.out.println(
                "FAIL  " + name + " line " + (i + 1)
                + ": expected \"" + expectedLine
                + "\" but got \"" + actualLine + "\""
            );
            match = false;
            break;
        }
    }

    if (match) {
        System.out.println("PASS  " + name);
        passed++;
    }
}

System.out.println(passed + " of " + files.length + " tests passed");

    }
}
