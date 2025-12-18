package com.example.calculator.operations;

import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.function.Consumer;
import java.util.stream.IntStream;
import java.util.stream.Stream;

import static com.example.calculator.operations.BasicOperations.*;

/**
 * Advanced calculator with modern Java features.
 * Demonstrates sealed classes, pattern matching, records, and more.
 */
public class AdvancedCalculator implements Calculator, AutoCloseable {

    // Instance counter using static field
    private static int instanceCount = 0;

    private final int precision;
    private final int historyLimit;
    private double memory = 0;
    private final Deque<HistoryEntry> history;
    private double lastResult = 0;
    private final List<Consumer<CalculationResult>> observers = new ArrayList<>();
    private final ExecutorService executor;

    /**
     * Record for history entries.
     */
    public record HistoryEntry(
            String expression,
            double result,
            Instant timestamp,
            Duration duration
    ) {
        public HistoryEntry(String expression, double result) {
            this(expression, result, Instant.now(), Duration.ZERO);
        }
    }

    /**
     * Sealed interface for calculation results - Java 17 feature.
     */
    public sealed interface CalculationResult permits SuccessResult, ErrorResult {
        double value();
        String operation();
        double[] inputs();
        boolean success();
        Duration duration();

        default String format() {
            return "%s(%s) = %f".formatted(operation(), Arrays.toString(inputs()), value());
        }
    }

    /**
     * Record implementing sealed interface.
     */
    public record SuccessResult(
            double value,
            String operation,
            double[] inputs,
            Duration duration
    ) implements CalculationResult {
        @Override
        public boolean success() {
            return true;
        }
    }

    public record ErrorResult(
            String operation,
            double[] inputs,
            String errorMessage,
            Duration duration
    ) implements CalculationResult {
        @Override
        public double value() {
            return 0;
        }

        @Override
        public boolean success() {
            return false;
        }
    }

    /**
     * Builder pattern for configuration.
     */
    public static class Builder {
        private int precision = 10;
        private int historyLimit = 100;
        private int threadPoolSize = 4;

        public Builder precision(int precision) {
            this.precision = precision;
            return this;
        }

        public Builder historyLimit(int historyLimit) {
            this.historyLimit = historyLimit;
            return this;
        }

        public Builder threadPoolSize(int size) {
            this.threadPoolSize = size;
            return this;
        }

        public AdvancedCalculator build() {
            return new AdvancedCalculator(this);
        }
    }

    public static Builder builder() {
        return new Builder();
    }

    private AdvancedCalculator(Builder builder) {
        this.precision = builder.precision;
        this.historyLimit = builder.historyLimit;
        this.history = new ArrayDeque<>(historyLimit);
        this.executor = Executors.newFixedThreadPool(builder.threadPoolSize);
        instanceCount++;
    }

    // Factory method with default configuration
    public static AdvancedCalculator create() {
        return builder().build();
    }

    public static int getInstanceCount() {
        return instanceCount;
    }

    @Override
    public CalculationResult calculate(String op, double... args) {
        var startTime = Instant.now();

        try {
            double value = switch (op) {
                // Binary operations using pattern matching for switch
                case "+", "add" -> {
                    requireArgs(args, 2);
                    yield add(args[0], args[1]);
                }
                case "-", "subtract" -> {
                    requireArgs(args, 2);
                    yield subtract(args[0], args[1]);
                }
                case "*", "multiply" -> {
                    requireArgs(args, 2);
                    yield multiply(args[0], args[1]);
                }
                case "/", "divide" -> {
                    requireArgs(args, 2);
                    yield divide(args[0], args[1])
                            .orElseThrow(() -> new CalculationException("Division by zero"));
                }
                case "**", "pow", "power" -> {
                    requireArgs(args, 2);
                    yield power(args[0], args[1]);
                }
                case "%", "mod", "modulo" -> {
                    requireArgs(args, 2);
                    yield modulo(args[0], args[1])
                            .orElseThrow(() -> new CalculationException("Modulo by zero"));
                }

                // Unary operations
                case "sqrt" -> {
                    requireArgs(args, 1);
                    if (args[0] < 0) {
                        throw new CalculationException("Cannot sqrt negative number");
                    }
                    yield Math.sqrt(args[0]);
                }
                case "log" -> {
                    requireMinArgs(args, 1, 2);
                    if (args[0] <= 0) {
                        throw new CalculationException("Log of non-positive number");
                    }
                    yield args.length == 2 ? Math.log(args[0]) / Math.log(args[1]) : Math.log(args[0]);
                }
                case "sin" -> {
                    requireArgs(args, 1);
                    yield Math.sin(args[0]);
                }
                case "cos" -> {
                    requireArgs(args, 1);
                    yield Math.cos(args[0]);
                }
                case "tan" -> {
                    requireArgs(args, 1);
                    yield Math.tan(args[0]);
                }
                case "abs" -> {
                    requireArgs(args, 1);
                    yield Math.abs(args[0]);
                }
                case "floor" -> {
                    requireArgs(args, 1);
                    yield Math.floor(args[0]);
                }
                case "ceil" -> {
                    requireArgs(args, 1);
                    yield Math.ceil(args[0]);
                }

                // Reduction operations
                case "sum" -> sum(args);
                case "product" -> product(args);
                case "max" -> {
                    requireMinArgs(args, 1);
                    yield Arrays.stream(args).max().orElse(0);
                }
                case "min" -> {
                    requireMinArgs(args, 1);
                    yield Arrays.stream(args).min().orElse(0);
                }
                case "avg", "mean" -> {
                    requireMinArgs(args, 1);
                    yield Arrays.stream(args).average().orElse(0);
                }

                default -> throw new CalculationException("Unknown operation: " + op);
            };

            // Round result
            value = roundToPrecision(value);
            lastResult = value;

            var duration = Duration.between(startTime, Instant.now());
            addToHistory(op, args, value, duration);

            var result = new SuccessResult(value, op, args, duration);
            notifyObservers(result);
            return result;

        } catch (CalculationException e) {
            var duration = Duration.between(startTime, Instant.now());
            var result = new ErrorResult(op, args, e.getMessage(), duration);
            notifyObservers(result);
            return result;
        }
    }

    private void requireArgs(double[] args, int expected) {
        if (args.length != expected) {
            throw new CalculationException("%s requires exactly %d arguments".formatted("Operation", expected));
        }
    }

    private void requireMinArgs(double[] args, int min) {
        requireMinArgs(args, min, Integer.MAX_VALUE);
    }

    private void requireMinArgs(double[] args, int min, int max) {
        if (args.length < min || args.length > max) {
            throw new CalculationException("Operation requires %d-%d arguments".formatted(min, max));
        }
    }

    private double roundToPrecision(double value) {
        double multiplier = Math.pow(10, precision);
        return Math.round(value * multiplier) / multiplier;
    }

    private void addToHistory(String op, double[] args, double result, Duration duration) {
        var expression = "%s(%s)".formatted(op, Arrays.toString(args));
        var entry = new HistoryEntry(expression, result, Instant.now(), duration);

        synchronized (history) {
            if (history.size() >= historyLimit) {
                history.removeFirst();
            }
            history.addLast(entry);
        }
    }

    @Override
    public void reset() {
        memory = 0;
        lastResult = 0;
        synchronized (history) {
            history.clear();
        }
    }

    @Override
    public List<HistoryEntry> getHistory() {
        synchronized (history) {
            return List.copyOf(history);
        }
    }

    // Memory operations
    public double getMemory() {
        return memory;
    }

    public double getLastResult() {
        return lastResult;
    }

    public void memoryAdd(double value) {
        memory += value;
    }

    public void memorySubtract(double value) {
        memory -= value;
    }

    public void memoryClear() {
        memory = 0;
    }

    public double memoryRecall() {
        return memory;
    }

    // Observer pattern
    public void subscribe(Consumer<CalculationResult> observer) {
        observers.add(observer);
    }

    public void unsubscribe(Consumer<CalculationResult> observer) {
        observers.remove(observer);
    }

    private void notifyObservers(CalculationResult result) {
        for (var observer : observers) {
            try {
                observer.accept(result);
            } catch (Exception e) {
                System.err.println("Observer error: " + e.getMessage());
            }
        }
    }

    // Async operations using CompletableFuture
    public CompletableFuture<CalculationResult> calculateAsync(String op, double... args) {
        return CompletableFuture.supplyAsync(() -> calculate(op, args), executor);
    }

    // Batch calculation with parallel streams
    public List<CalculationResult> batchCalculate(List<CalculationRequest> requests) {
        return requests.parallelStream()
                .map(req -> calculate(req.operation(), req.args()))
                .toList();
    }

    // Record for batch requests
    public record CalculationRequest(String operation, double[] args) {}

    @Override
    public void close() {
        executor.shutdown();
        try {
            if (!executor.awaitTermination(5, TimeUnit.SECONDS)) {
                executor.shutdownNow();
            }
        } catch (InterruptedException e) {
            executor.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Custom exception for calculations.
     */
    public static class CalculationException extends RuntimeException {
        public CalculationException(String message) {
            super(message);
        }
    }
}

/**
 * Scientific calculator extending AdvancedCalculator.
 */
class ScientificCalculator extends AdvancedCalculator {

    /**
     * Enum for angle modes.
     */
    public enum AngleMode {
        RADIANS, DEGREES
    }

    private AngleMode angleMode = AngleMode.RADIANS;

    private ScientificCalculator(Builder builder) {
        super(builder);
    }

    public static ScientificCalculator create() {
        return new ScientificCalculator(builder());
    }

    public AngleMode getAngleMode() {
        return angleMode;
    }

    public void setAngleMode(AngleMode mode) {
        this.angleMode = Objects.requireNonNull(mode);
    }

    private double toRadians(double angle) {
        return angleMode == AngleMode.DEGREES ? Math.toRadians(angle) : angle;
    }

    /**
     * Factorial using iteration.
     */
    public long factorial(int n) {
        if (n < 0) {
            throw new CalculationException("Factorial of negative number");
        }
        if (n > 20) {
            throw new CalculationException("Factorial overflow (max 20)");
        }
        return IntStream.rangeClosed(1, n)
                .mapToLong(i -> i)
                .reduce(1, (a, b) -> a * b);
    }

    /**
     * Fibonacci using Stream.iterate - Java 9+ feature.
     */
    public long fibonacci(int n) {
        if (n < 0) {
            throw new CalculationException("Negative fibonacci index");
        }
        return Stream.iterate(new long[]{0, 1}, arr -> new long[]{arr[1], arr[0] + arr[1]})
                .limit(n + 1)
                .map(arr -> arr[0])
                .reduce((first, second) -> second)
                .orElse(0L);
    }

    /**
     * Check if a number is prime.
     */
    public boolean isPrime(int n) {
        if (n < 2) return false;
        if (n == 2) return true;
        if (n % 2 == 0) return false;

        return IntStream.iterate(3, i -> i <= Math.sqrt(n), i -> i + 2)
                .noneMatch(i -> n % i == 0);
    }

    /**
     * Get prime factors using streams.
     */
    public List<Integer> primeFactors(int n) {
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
        return List.copyOf(factors);
    }

    /**
     * GCD using recursion.
     */
    public int gcd(int a, int b) {
        a = Math.abs(a);
        b = Math.abs(b);
        return b == 0 ? a : gcd(b, a % b);
    }

    /**
     * LCM using GCD.
     */
    public int lcm(int a, int b) {
        if (a == 0 || b == 0) return 0;
        return Math.abs(a * b) / gcd(a, b);
    }

    /**
     * Generate Fibonacci sequence as Stream.
     */
    public Stream<Long> fibonacciStream() {
        return Stream.iterate(new long[]{0, 1}, arr -> new long[]{arr[1], arr[0] + arr[1]})
                .map(arr -> arr[0]);
    }

    /**
     * Generate primes as Stream using infinite iteration.
     */
    public Stream<Integer> primeStream() {
        return Stream.iterate(2, n -> n + 1)
                .filter(this::isPrime);
    }
}

/**
 * Calculator interface - sealed to restrict implementations.
 */
sealed interface Calculator permits AdvancedCalculator {
    AdvancedCalculator.CalculationResult calculate(String op, double... args);
    void reset();
    List<AdvancedCalculator.HistoryEntry> getHistory();
}
