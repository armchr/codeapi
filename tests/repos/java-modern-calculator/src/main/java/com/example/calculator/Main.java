package com.example.calculator;

import com.example.calculator.operations.AdvancedCalculator;
import com.example.calculator.operations.BasicOperations;
import com.example.calculator.util.ValidationUtils;

import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.function.Function;
import java.util.stream.Collectors;
import java.util.stream.IntStream;

import static com.example.calculator.util.ValidationUtils.*;

/**
 * Main entry point for the calculator application.
 * Demonstrates modern Java features.
 */
public class Main {

    private static final String VERSION = "1.0.0";
    private static final String APP_NAME = "Modern Java Calculator";

    // Lazy initialization with holder pattern
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

    // Memoized fibonacci
    private static final Function<Integer, Long> fibonacciCached = memoize(Main::fibonacciImpl);

    private static long fibonacciImpl(int n) {
        if (n <= 1) return n;
        return fibonacciCached.apply(n - 1) + fibonacciCached.apply(n - 2);
    }

    // Command handlers using sealed interface
    sealed interface Command permits SimpleCommand, ArgumentCommand {
        String execute(String[] args);
        String description();
    }

    record SimpleCommand(String description, java.util.function.Supplier<String> handler) implements Command {
        @Override
        public String execute(String[] args) {
            return handler.get();
        }
    }

    record ArgumentCommand(String description, Function<String[], String> handler) implements Command {
        @Override
        public String execute(String[] args) {
            return handler.apply(args);
        }
    }

    // Commands map
    private static final Map<String, Command> COMMANDS = Map.ofEntries(
            Map.entry("help", new SimpleCommand("Show help information", Main::getHelpText)),
            Map.entry("history", new SimpleCommand("Show calculation history", Main::handleHistory)),
            Map.entry("mc", new SimpleCommand("Clear memory", () -> {
                getCalculator().memoryClear();
                return "Memory cleared";
            })),
            Map.entry("mr", new SimpleCommand("Recall memory", () ->
                    "Memory: " + getCalculator().memoryRecall())),
            Map.entry("m+", new ArgumentCommand("Add to memory", args -> {
                var value = parseDouble(args.length > 0 ? args[0] : "0").orElse(0.0);
                getCalculator().memoryAdd(value);
                return "Added to memory: " + getCalculator().getMemory();
            })),
            Map.entry("m-", new ArgumentCommand("Subtract from memory", args -> {
                var value = parseDouble(args.length > 0 ? args[0] : "0").orElse(0.0);
                getCalculator().memorySubtract(value);
                return "Subtracted from memory: " + getCalculator().getMemory();
            })),
            Map.entry("prime", new ArgumentCommand("Check if prime", args -> {
                var n = parseDouble(args.length > 0 ? args[0] : "0").map(Double::intValue).orElse(0);
                var isPrime = isPrimeCheck(n);
                return n + " is " + (isPrime ? "" : "not ") + "prime";
            })),
            Map.entry("factors", new ArgumentCommand("Get prime factors", args -> {
                var n = parseDouble(args.length > 0 ? args[0] : "0").map(Double::intValue).orElse(0);
                var factors = primeFactors(n);
                return "Prime factors of " + n + ": " + factors;
            })),
            Map.entry("fib", new ArgumentCommand("Calculate Fibonacci", args -> {
                var n = parseDouble(args.length > 0 ? args[0] : "0").map(Double::intValue).orElse(0);
                return "Fibonacci(%d) = %d".formatted(n, fibonacciCached.apply(n));
            }))
    );

    private static String getHelpText() {
        return """
                %s Commands:
                  Basic:     2 + 3, 10 - 5, 4 * 3, 20 / 4
                  Power:     2 ** 8, pow(2, 8)
                  Functions: sqrt(16), log(100), sin(0.5), cos(0.5), tan(0.5)
                  Stats:     sum(1,2,3), avg(1,2,3), max(1,2,3), min(1,2,3)

                Memory:
                  mc - Clear memory
                  mr - Recall memory
                  m+ <value> - Add to memory
                  m- <value> - Subtract from memory

                Other:
                  history - Show calculation history
                  prime <n> - Check if n is prime
                  factors <n> - Get prime factors of n
                  fib <n> - Calculate nth Fibonacci number
                  help - Show this help
                  quit - Exit calculator
                """.formatted(APP_NAME);
    }

    private static String handleHistory() {
        var history = getCalculator().getHistory();
        if (history.isEmpty()) {
            return "No history";
        }
        return history.stream()
                .map(entry -> "  %s = %f".formatted(entry.expression(), entry.result()))
                .collect(Collectors.joining("\n"));
    }

    private static boolean isPrimeCheck(int n) {
        if (n < 2) return false;
        if (n == 2) return true;
        if (n % 2 == 0) return false;
        return IntStream.iterate(3, i -> i <= Math.sqrt(n), i -> i + 2)
                .noneMatch(i -> n % i == 0);
    }

    private static List<Integer> primeFactors(int n) {
        var factors = new ArrayList<Integer>();
        int d = 2;
        while (d * d <= n) {
            while (n % d == 0) {
                factors.add(d);
                n /= d;
            }
            d++;
        }
        if (n > 1) factors.add(n);
        return factors;
    }

    /**
     * Process a single command.
     */
    private static ProcessResult processCommand(String input) {
        input = input.strip();

        // Empty input
        if (input.isEmpty()) {
            return new ProcessResult("", false);
        }

        // Exit commands using pattern matching for instanceof (Java 16+)
        var lowerInput = input.toLowerCase();
        if (lowerInput.equals("exit") || lowerInput.equals("quit") || lowerInput.equals("q")) {
            return new ProcessResult("Goodbye!", true);
        }

        // Check built-in commands
        var parts = input.split("\\s+");
        var cmdName = parts[0].toLowerCase();
        var args = Arrays.copyOfRange(parts, 1, parts.length);

        var command = COMMANDS.get(cmdName);
        if (command != null) {
            try {
                return new ProcessResult(command.execute(args), false);
            } catch (Exception e) {
                return new ProcessResult("Error: " + e.getMessage(), false);
            }
        }

        // Try to parse as expression
        var parsedOpt = parseExpression(input);
        if (parsedOpt.isPresent()) {
            var parsed = parsedOpt.get();
            var result = getCalculator().calculate(parsed.operator(), parsed.operands());

            // Pattern matching for sealed interface (Java 17+)
            return switch (result) {
                case AdvancedCalculator.SuccessResult success ->
                        new ProcessResult(formatNumber(success.value(), FormatOptions.withPrecision(6)), false);
                case AdvancedCalculator.ErrorResult error ->
                        new ProcessResult("Error: " + error.errorMessage(), false);
            };
        }

        return new ProcessResult("Error: Cannot parse expression: " + input, false);
    }

    // Record for process result
    private record ProcessResult(String output, boolean exit) {}

    /**
     * Run interactive mode.
     */
    private static void runInteractive() {
        System.out.printf("%s v%s%n", APP_NAME, VERSION);
        System.out.println("Type 'help' for commands, 'quit' to exit\n");

        try (var scanner = new Scanner(System.in)) {
            while (true) {
                System.out.print("calc> ");
                if (!scanner.hasNextLine()) {
                    break;
                }

                var input = scanner.nextLine();
                var result = processCommand(input);

                if (!result.output().isEmpty()) {
                    System.out.println(result.output());
                }

                if (result.exit()) {
                    break;
                }
            }
        }
    }

    /**
     * Run batch mode with parallel processing.
     */
    private static void runBatch(String[] expressions) {
        // Filter comments and empty lines
        var filtered = Arrays.stream(expressions)
                .map(String::strip)
                .filter(e -> !e.isEmpty() && !e.startsWith("#"))
                .toList();

        // Process with CompletableFuture for parallel execution
        var futures = filtered.stream()
                .map(expr -> CompletableFuture.supplyAsync(() -> {
                    var result = processCommand(expr);
                    return expr + " = " + result.output();
                }))
                .toList();

        // Wait for all and print results
        CompletableFuture.allOf(futures.toArray(CompletableFuture[]::new))
                .thenRun(() -> futures.stream()
                        .map(CompletableFuture::join)
                        .forEach(System.out::println))
                .join();
    }

    /**
     * Run demo mode.
     */
    private static void runDemo() {
        System.out.println("Calculator Demo");
        System.out.println("===============\n");

        var calc = getCalculator();

        // Basic operations
        System.out.println("Basic operations:");
        System.out.println("  add(5, 3) = " + calc.calculate("+", 5, 3).value());
        System.out.println("  subtract(10, 4) = " + calc.calculate("-", 10, 4).value());
        System.out.println("  multiply(7, 8) = " + calc.calculate("*", 7, 8).value());
        System.out.println("  divide(20, 4) = " + calc.calculate("/", 20, 4).value());
        System.out.println();

        // Scientific operations
        System.out.println("Scientific operations:");
        System.out.println("  sqrt(144) = " + calc.calculate("sqrt", 144).value());
        System.out.println("  log(Math.E) = " + calc.calculate("log", Math.E).value());
        System.out.println("  sin(PI/2) = " + calc.calculate("sin", Math.PI / 2).value());
        System.out.println();

        // Reduction operations
        System.out.println("Reduction operations:");
        System.out.println("  sum(1,2,3,4,5) = " + calc.calculate("sum", 1, 2, 3, 4, 5).value());
        System.out.println("  avg(1,2,3,4,5) = " + calc.calculate("avg", 1, 2, 3, 4, 5).value());
        System.out.println("  max(1,5,3,9,2) = " + calc.calculate("max", 1, 5, 3, 9, 2).value());
        System.out.println("  min(1,5,3,9,2) = " + calc.calculate("min", 1, 5, 3, 9, 2).value());
        System.out.println();

        // Functional operations with streams
        System.out.println("Stream operations:");
        var numbers = List.of(1, 2, 3, 4, 5);
        var squares = BasicOperations.map(numbers, n -> n * n);
        var evens = BasicOperations.filter(numbers, n -> n % 2 == 0);
        var sum = BasicOperations.reduce(numbers, 0, Integer::sum);
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

        // Async calculation
        System.out.println("Async calculation:");
        calc.calculateAsync("sqrt", 256)
                .thenAccept(result -> System.out.println("  async sqrt(256) = " + result.value()))
                .join();
    }

    public static void main(String[] args) {
        // Parse command line arguments
        boolean showHelp = false;
        boolean showVersion = false;
        boolean runDemoMode = false;
        boolean batchMode = false;
        var expressions = new ArrayList<String>();

        for (int i = 0; i < args.length; i++) {
            switch (args[i]) {
                case "-h", "--help" -> showHelp = true;
                case "-v", "--version" -> showVersion = true;
                case "--demo" -> runDemoMode = true;
                case "--batch" -> batchMode = true;
                default -> expressions.add(args[i]);
            }
        }

        // Handle flags
        if (showHelp) {
            System.out.println(getHelpText());
            return;
        }

        if (showVersion) {
            System.out.printf("%s v%s%n", APP_NAME, VERSION);
            return;
        }

        if (runDemoMode) {
            runDemo();
            return;
        }

        // Run appropriate mode
        if (batchMode || !expressions.isEmpty()) {
            runBatch(expressions.toArray(String[]::new));
        } else {
            runInteractive();
        }

        // Cleanup
        getCalculator().close();
    }
}
