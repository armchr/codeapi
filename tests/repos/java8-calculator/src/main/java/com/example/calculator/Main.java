package com.example.calculator;

import com.example.calculator.operations.AdvancedCalculator;
import com.example.calculator.operations.BasicOperations;
import com.example.calculator.util.ValidationUtils;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Scanner;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Future;
import java.util.function.Function;
import java.util.function.Supplier;
import java.util.stream.Collectors;
import java.util.stream.IntStream;

import static com.example.calculator.util.ValidationUtils.*;

/**
 * Main entry point for the calculator application.
 * Java 8 style without modern features like text blocks, records, or pattern matching.
 */
public class Main {

    private static final String VERSION = "1.0.0";
    private static final String APP_NAME = "Java 8 Calculator";

    // Lazy initialization using holder pattern
    private static class CalculatorHolder {
        static final AdvancedCalculator INSTANCE = AdvancedCalculator.builder()
                .precision(10)
                .historyLimit(100)
                .threadPoolSize(4)
                .build();
    }

    private static AdvancedCalculator getCalculator() {
        return CalculatorHolder.INSTANCE;
    }

    // Memoized fibonacci (Java 8 style)
    private static final Function<Integer, Long> fibonacciCached = memoize(Main::fibonacciImpl);

    private static long fibonacciImpl(int n) {
        if (n <= 1) {
            return n;
        }
        return fibonacciCached.apply(n - 1) + fibonacciCached.apply(n - 2);
    }

    // Command interface (not sealed in Java 8)
    interface Command {
        String execute(String[] args);
        String getDescription();
    }

    // Simple command implementation (not a record)
    static class SimpleCommand implements Command {
        private final String description;
        private final Supplier<String> handler;

        SimpleCommand(String description, Supplier<String> handler) {
            this.description = description;
            this.handler = handler;
        }

        @Override
        public String execute(String[] args) {
            return handler.get();
        }

        @Override
        public String getDescription() {
            return description;
        }
    }

    // Argument command implementation
    static class ArgumentCommand implements Command {
        private final String description;
        private final Function<String[], String> handler;

        ArgumentCommand(String description, Function<String[], String> handler) {
            this.description = description;
            this.handler = handler;
        }

        @Override
        public String execute(String[] args) {
            return handler.apply(args);
        }

        @Override
        public String getDescription() {
            return description;
        }
    }

    // Commands map (using HashMap instead of Map.ofEntries)
    private static final Map<String, Command> COMMANDS;

    static {
        COMMANDS = new HashMap<>();

        COMMANDS.put("help", new SimpleCommand("Show help information", Main::getHelpText));
        COMMANDS.put("history", new SimpleCommand("Show calculation history", Main::handleHistory));

        COMMANDS.put("mc", new SimpleCommand("Clear memory", () -> {
            getCalculator().memoryClear();
            return "Memory cleared";
        }));

        COMMANDS.put("mr", new SimpleCommand("Recall memory", () -> {
            return "Memory: " + getCalculator().memoryRecall();
        }));

        COMMANDS.put("m+", new ArgumentCommand("Add to memory", args -> {
            double value = parseDouble(args.length > 0 ? args[0] : "0").orElse(0.0);
            getCalculator().memoryAdd(value);
            return "Added to memory: " + getCalculator().getMemory();
        }));

        COMMANDS.put("m-", new ArgumentCommand("Subtract from memory", args -> {
            double value = parseDouble(args.length > 0 ? args[0] : "0").orElse(0.0);
            getCalculator().memorySubtract(value);
            return "Subtracted from memory: " + getCalculator().getMemory();
        }));

        COMMANDS.put("prime", new ArgumentCommand("Check if prime", args -> {
            int n = parseDouble(args.length > 0 ? args[0] : "0")
                    .map(Double::intValue)
                    .orElse(0);
            boolean isPrime = isPrimeCheck(n);
            return n + " is " + (isPrime ? "" : "not ") + "prime";
        }));

        COMMANDS.put("factors", new ArgumentCommand("Get prime factors", args -> {
            int n = parseDouble(args.length > 0 ? args[0] : "0")
                    .map(Double::intValue)
                    .orElse(0);
            List<Integer> factors = primeFactors(n);
            return "Prime factors of " + n + ": " + factors;
        }));

        COMMANDS.put("fib", new ArgumentCommand("Calculate Fibonacci", args -> {
            int n = parseDouble(args.length > 0 ? args[0] : "0")
                    .map(Double::intValue)
                    .orElse(0);
            return String.format("Fibonacci(%d) = %d", n, fibonacciCached.apply(n));
        }));
    }

    private static String getHelpText() {
        // No text blocks in Java 8, using string concatenation
        StringBuilder sb = new StringBuilder();
        sb.append(APP_NAME).append(" Commands:\n");
        sb.append("  Basic:     2 + 3, 10 - 5, 4 * 3, 20 / 4\n");
        sb.append("  Power:     2 ** 8, pow(2, 8)\n");
        sb.append("  Functions: sqrt(16), log(100), sin(0.5), cos(0.5), tan(0.5)\n");
        sb.append("  Stats:     sum(1,2,3), avg(1,2,3), max(1,2,3), min(1,2,3)\n");
        sb.append("\n");
        sb.append("Memory:\n");
        sb.append("  mc - Clear memory\n");
        sb.append("  mr - Recall memory\n");
        sb.append("  m+ <value> - Add to memory\n");
        sb.append("  m- <value> - Subtract from memory\n");
        sb.append("\n");
        sb.append("Other:\n");
        sb.append("  history - Show calculation history\n");
        sb.append("  prime <n> - Check if n is prime\n");
        sb.append("  factors <n> - Get prime factors of n\n");
        sb.append("  fib <n> - Calculate nth Fibonacci number\n");
        sb.append("  help - Show this help\n");
        sb.append("  quit - Exit calculator\n");
        return sb.toString();
    }

    private static String handleHistory() {
        List<AdvancedCalculator.HistoryEntry> history = getCalculator().getHistory();
        if (history.isEmpty()) {
            return "No history";
        }
        StringBuilder sb = new StringBuilder();
        for (AdvancedCalculator.HistoryEntry entry : history) {
            sb.append(String.format("  %s = %f%n", entry.getExpression(), entry.getResult()));
        }
        return sb.toString().trim();
    }

    private static boolean isPrimeCheck(int n) {
        if (n < 2) return false;
        if (n == 2) return true;
        if (n % 2 == 0) return false;

        int sqrt = (int) Math.sqrt(n);
        for (int i = 3; i <= sqrt; i += 2) {
            if (n % i == 0) return false;
        }
        return true;
    }

    private static List<Integer> primeFactors(int n) {
        List<Integer> factors = new ArrayList<>();
        int d = 2;
        while (d * d <= n) {
            while (n % d == 0) {
                factors.add(d);
                n /= d;
            }
            d++;
        }
        if (n > 1) {
            factors.add(n);
        }
        return factors;
    }

    /**
     * Process result class (not a record in Java 8).
     */
    static class ProcessResult {
        private final String output;
        private final boolean exit;

        ProcessResult(String output, boolean exit) {
            this.output = output;
            this.exit = exit;
        }

        String getOutput() {
            return output;
        }

        boolean isExit() {
            return exit;
        }
    }

    /**
     * Process a single command.
     */
    private static ProcessResult processCommand(String input) {
        input = input.trim();

        // Empty input
        if (input.isEmpty()) {
            return new ProcessResult("", false);
        }

        // Exit commands (no pattern matching in Java 8)
        String lowerInput = input.toLowerCase();
        if (lowerInput.equals("exit") || lowerInput.equals("quit") || lowerInput.equals("q")) {
            return new ProcessResult("Goodbye!", true);
        }

        // Check built-in commands
        String[] parts = input.split("\\s+");
        String cmdName = parts[0].toLowerCase();
        String[] args = Arrays.copyOfRange(parts, 1, parts.length);

        Command command = COMMANDS.get(cmdName);
        if (command != null) {
            try {
                return new ProcessResult(command.execute(args), false);
            } catch (Exception e) {
                return new ProcessResult("Error: " + e.getMessage(), false);
            }
        }

        // Try to parse as expression
        Optional<ValidationUtils.ParsedExpression> parsedOpt = parseExpression(input);
        if (parsedOpt.isPresent()) {
            ValidationUtils.ParsedExpression parsed = parsedOpt.get();
            AdvancedCalculator.CalculationResult result =
                    getCalculator().calculate(parsed.getOperator(), parsed.getOperands());

            // Check result type (no pattern matching, using instanceof)
            if (result.isSuccess()) {
                return new ProcessResult(
                        formatNumber(result.getValue(), FormatOptions.withPrecision(6)),
                        false
                );
            } else if (result instanceof AdvancedCalculator.ErrorResult) {
                AdvancedCalculator.ErrorResult errorResult = (AdvancedCalculator.ErrorResult) result;
                return new ProcessResult("Error: " + errorResult.getErrorMessage(), false);
            } else {
                return new ProcessResult("Error: Unknown error", false);
            }
        }

        return new ProcessResult("Error: Cannot parse expression: " + input, false);
    }

    /**
     * Run interactive mode.
     */
    private static void runInteractive() {
        System.out.println(APP_NAME + " v" + VERSION);
        System.out.println("Type 'help' for commands, 'quit' to exit");
        System.out.println();

        Scanner scanner = new Scanner(System.in);
        try {
            while (true) {
                System.out.print("calc> ");
                if (!scanner.hasNextLine()) {
                    break;
                }

                String input = scanner.nextLine();
                ProcessResult result = processCommand(input);

                if (!result.getOutput().isEmpty()) {
                    System.out.println(result.getOutput());
                }

                if (result.isExit()) {
                    break;
                }
            }
        } finally {
            scanner.close();
        }
    }

    /**
     * Run batch mode.
     */
    private static void runBatch(String[] expressions) {
        // Filter comments and empty lines (Java 8 style)
        List<String> filtered = new ArrayList<>();
        for (String expr : expressions) {
            String trimmed = expr.trim();
            if (!trimmed.isEmpty() && !trimmed.startsWith("#")) {
                filtered.add(trimmed);
            }
        }

        // Process each expression
        for (String expr : filtered) {
            ProcessResult result = processCommand(expr);
            System.out.println(expr + " = " + result.getOutput());
        }
    }

    /**
     * Run demo mode.
     */
    private static void runDemo() {
        System.out.println("Calculator Demo");
        System.out.println("===============");
        System.out.println();

        AdvancedCalculator calc = getCalculator();

        // Basic operations
        System.out.println("Basic operations:");
        System.out.println("  add(5, 3) = " + calc.calculate("+", 5, 3).getValue());
        System.out.println("  subtract(10, 4) = " + calc.calculate("-", 10, 4).getValue());
        System.out.println("  multiply(7, 8) = " + calc.calculate("*", 7, 8).getValue());
        System.out.println("  divide(20, 4) = " + calc.calculate("/", 20, 4).getValue());
        System.out.println();

        // Scientific operations
        System.out.println("Scientific operations:");
        System.out.println("  sqrt(144) = " + calc.calculate("sqrt", 144).getValue());
        System.out.println("  log(Math.E) = " + calc.calculate("log", Math.E).getValue());
        System.out.println("  sin(PI/2) = " + calc.calculate("sin", Math.PI / 2).getValue());
        System.out.println();

        // Reduction operations
        System.out.println("Reduction operations:");
        System.out.println("  sum(1,2,3,4,5) = " + calc.calculate("sum", 1, 2, 3, 4, 5).getValue());
        System.out.println("  avg(1,2,3,4,5) = " + calc.calculate("avg", 1, 2, 3, 4, 5).getValue());
        System.out.println("  max(1,5,3,9,2) = " + calc.calculate("max", 1, 5, 3, 9, 2).getValue());
        System.out.println("  min(1,5,3,9,2) = " + calc.calculate("min", 1, 5, 3, 9, 2).getValue());
        System.out.println();

        // Functional operations (Java 8 style)
        System.out.println("Functional operations:");
        List<Integer> numbers = Arrays.asList(1, 2, 3, 4, 5);
        List<Integer> squares = BasicOperations.map(numbers, n -> n * n);
        List<Integer> evens = BasicOperations.filter(numbers, n -> n % 2 == 0);
        Integer sum = BasicOperations.reduce(numbers, 0, Integer::sum);
        System.out.println("  numbers = " + numbers);
        System.out.println("  squares = " + squares);
        System.out.println("  evens = " + evens);
        System.out.println("  sum = " + sum);
        System.out.println();

        // Number theory
        System.out.println("Number theory:");
        System.out.println("  fibonacci(20) = " + fibonacciCached.apply(20));
        System.out.println("  isPrime(17) = " + isPrimeCheck(17));
        System.out.println("  primeFactors(84) = " + primeFactors(84));
        System.out.println();

        // Async calculation (Java 8 style with Future)
        System.out.println("Async calculation:");
        Future<AdvancedCalculator.CalculationResult> future = calc.calculateAsync("sqrt", 256);
        try {
            AdvancedCalculator.CalculationResult result = future.get();
            System.out.println("  async sqrt(256) = " + result.getValue());
        } catch (InterruptedException | ExecutionException e) {
            System.out.println("  async error: " + e.getMessage());
        }
    }

    public static void main(String[] args) {
        // Parse command line arguments (Java 8 style, no enhanced switch)
        boolean showHelp = false;
        boolean showVersion = false;
        boolean runDemoMode = false;
        boolean batchMode = false;
        List<String> expressions = new ArrayList<>();

        for (int i = 0; i < args.length; i++) {
            String arg = args[i];
            if (arg.equals("-h") || arg.equals("--help")) {
                showHelp = true;
            } else if (arg.equals("-v") || arg.equals("--version")) {
                showVersion = true;
            } else if (arg.equals("--demo")) {
                runDemoMode = true;
            } else if (arg.equals("--batch")) {
                batchMode = true;
            } else {
                expressions.add(arg);
            }
        }

        // Handle flags (if-else chain instead of switch expressions)
        if (showHelp) {
            System.out.println(getHelpText());
            return;
        }

        if (showVersion) {
            System.out.println(APP_NAME + " v" + VERSION);
            return;
        }

        if (runDemoMode) {
            runDemo();
            return;
        }

        // Run appropriate mode
        if (batchMode || !expressions.isEmpty()) {
            runBatch(expressions.toArray(new String[0]));
        } else {
            runInteractive();
        }

        // Cleanup
        getCalculator().close();
    }
}
