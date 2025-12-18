package com.example.calculator.operations;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Deque;
import java.util.List;
import java.util.Objects;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

import static com.example.calculator.operations.BasicOperations.*;

/**
 * Advanced calculator with Java 8/11 style.
 * No records, sealed classes, or pattern matching.
 */
public class AdvancedCalculator implements Calculator, AutoCloseable {

    // Instance counter
    private static int instanceCount = 0;

    private final int precision;
    private final int historyLimit;
    private double memory = 0;
    private final Deque<HistoryEntry> history;
    private double lastResult = 0;
    private final List<Consumer<CalculationResult>> observers = new ArrayList<>();
    private final ExecutorService executor;

    /**
     * History entry class (not a record in Java 8).
     */
    public static class HistoryEntry {
        private final String expression;
        private final double result;
        private final Instant timestamp;
        private final Duration duration;

        public HistoryEntry(String expression, double result, Instant timestamp, Duration duration) {
            this.expression = expression;
            this.result = result;
            this.timestamp = timestamp;
            this.duration = duration;
        }

        public HistoryEntry(String expression, double result) {
            this(expression, result, Instant.now(), Duration.ZERO);
        }

        public String getExpression() {
            return expression;
        }

        public double getResult() {
            return result;
        }

        public Instant getTimestamp() {
            return timestamp;
        }

        public Duration getDuration() {
            return duration;
        }

        @Override
        public String toString() {
            return "HistoryEntry{" +
                    "expression='" + expression + '\'' +
                    ", result=" + result +
                    ", timestamp=" + timestamp +
                    '}';
        }
    }

    /**
     * Interface for calculation results (not sealed in Java 8).
     */
    public interface CalculationResult {
        double getValue();
        String getOperation();
        double[] getInputs();
        boolean isSuccess();
        Duration getDuration();

        default String format() {
            return String.format("%s(%s) = %f", getOperation(), Arrays.toString(getInputs()), getValue());
        }
    }

    /**
     * Success result class.
     */
    public static class SuccessResult implements CalculationResult {
        private final double value;
        private final String operation;
        private final double[] inputs;
        private final Duration duration;

        public SuccessResult(double value, String operation, double[] inputs, Duration duration) {
            this.value = value;
            this.operation = operation;
            this.inputs = inputs != null ? inputs.clone() : new double[0];
            this.duration = duration;
        }

        @Override
        public double getValue() {
            return value;
        }

        @Override
        public String getOperation() {
            return operation;
        }

        @Override
        public double[] getInputs() {
            return inputs.clone();
        }

        @Override
        public boolean isSuccess() {
            return true;
        }

        @Override
        public Duration getDuration() {
            return duration;
        }
    }

    /**
     * Error result class.
     */
    public static class ErrorResult implements CalculationResult {
        private final String operation;
        private final double[] inputs;
        private final String errorMessage;
        private final Duration duration;

        public ErrorResult(String operation, double[] inputs, String errorMessage, Duration duration) {
            this.operation = operation;
            this.inputs = inputs != null ? inputs.clone() : new double[0];
            this.errorMessage = errorMessage;
            this.duration = duration;
        }

        @Override
        public double getValue() {
            return 0;
        }

        @Override
        public String getOperation() {
            return operation;
        }

        @Override
        public double[] getInputs() {
            return inputs.clone();
        }

        @Override
        public boolean isSuccess() {
            return false;
        }

        @Override
        public Duration getDuration() {
            return duration;
        }

        public String getErrorMessage() {
            return errorMessage;
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

    public static AdvancedCalculator create() {
        return builder().build();
    }

    public static int getInstanceCount() {
        return instanceCount;
    }

    @Override
    public CalculationResult calculate(String op, double... args) {
        Instant startTime = Instant.now();

        try {
            double value;

            // Switch statement (no arrow syntax or yield in Java 8)
            switch (op) {
                case "+":
                case "add":
                    requireArgs(args, 2);
                    value = add(args[0], args[1]);
                    break;

                case "-":
                case "subtract":
                    requireArgs(args, 2);
                    value = subtract(args[0], args[1]);
                    break;

                case "*":
                case "multiply":
                    requireArgs(args, 2);
                    value = multiply(args[0], args[1]);
                    break;

                case "/":
                case "divide":
                    requireArgs(args, 2);
                    value = divide(args[0], args[1])
                            .orElseThrow(() -> new CalculationException("Division by zero"));
                    break;

                case "**":
                case "pow":
                case "power":
                    requireArgs(args, 2);
                    value = power(args[0], args[1]);
                    break;

                case "%":
                case "mod":
                case "modulo":
                    requireArgs(args, 2);
                    value = modulo(args[0], args[1])
                            .orElseThrow(() -> new CalculationException("Modulo by zero"));
                    break;

                case "sqrt":
                    requireArgs(args, 1);
                    if (args[0] < 0) {
                        throw new CalculationException("Cannot sqrt negative number");
                    }
                    value = Math.sqrt(args[0]);
                    break;

                case "log":
                    requireMinArgs(args, 1, 2);
                    if (args[0] <= 0) {
                        throw new CalculationException("Log of non-positive number");
                    }
                    if (args.length == 2) {
                        value = Math.log(args[0]) / Math.log(args[1]);
                    } else {
                        value = Math.log(args[0]);
                    }
                    break;

                case "sin":
                    requireArgs(args, 1);
                    value = Math.sin(args[0]);
                    break;

                case "cos":
                    requireArgs(args, 1);
                    value = Math.cos(args[0]);
                    break;

                case "tan":
                    requireArgs(args, 1);
                    value = Math.tan(args[0]);
                    break;

                case "abs":
                    requireArgs(args, 1);
                    value = Math.abs(args[0]);
                    break;

                case "floor":
                    requireArgs(args, 1);
                    value = Math.floor(args[0]);
                    break;

                case "ceil":
                    requireArgs(args, 1);
                    value = Math.ceil(args[0]);
                    break;

                case "sum":
                    value = sum(args);
                    break;

                case "product":
                    value = product(args);
                    break;

                case "max":
                    requireMinArgs(args, 1);
                    value = args[0];
                    for (int i = 1; i < args.length; i++) {
                        if (args[i] > value) {
                            value = args[i];
                        }
                    }
                    break;

                case "min":
                    requireMinArgs(args, 1);
                    value = args[0];
                    for (int i = 1; i < args.length; i++) {
                        if (args[i] < value) {
                            value = args[i];
                        }
                    }
                    break;

                case "avg":
                case "mean":
                    requireMinArgs(args, 1);
                    value = sum(args) / args.length;
                    break;

                default:
                    throw new CalculationException("Unknown operation: " + op);
            }

            // Round result
            value = roundToPrecision(value);
            lastResult = value;

            Duration duration = Duration.between(startTime, Instant.now());
            addToHistory(op, args, value, duration);

            SuccessResult result = new SuccessResult(value, op, args, duration);
            notifyObservers(result);
            return result;

        } catch (CalculationException e) {
            Duration duration = Duration.between(startTime, Instant.now());
            ErrorResult result = new ErrorResult(op, args, e.getMessage(), duration);
            notifyObservers(result);
            return result;
        }
    }

    private void requireArgs(double[] args, int expected) {
        if (args.length != expected) {
            throw new CalculationException("Operation requires exactly " + expected + " arguments");
        }
    }

    private void requireMinArgs(double[] args, int min) {
        requireMinArgs(args, min, Integer.MAX_VALUE);
    }

    private void requireMinArgs(double[] args, int min, int max) {
        if (args.length < min || args.length > max) {
            throw new CalculationException("Operation requires " + min + "-" + max + " arguments");
        }
    }

    private double roundToPrecision(double value) {
        double multiplier = Math.pow(10, precision);
        return Math.round(value * multiplier) / multiplier;
    }

    private void addToHistory(String op, double[] args, double result, Duration duration) {
        String expression = String.format("%s(%s)", op, Arrays.toString(args));
        HistoryEntry entry = new HistoryEntry(expression, result, Instant.now(), duration);

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
            return new ArrayList<>(history);
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
        for (Consumer<CalculationResult> observer : observers) {
            try {
                observer.accept(result);
            } catch (Exception e) {
                System.err.println("Observer error: " + e.getMessage());
            }
        }
    }

    // Async operations using Future (Java 8 style)
    public Future<CalculationResult> calculateAsync(final String op, final double... args) {
        return executor.submit(new Callable<CalculationResult>() {
            @Override
            public CalculationResult call() {
                return calculate(op, args);
            }
        });
    }

    // Alternative with lambda
    public Future<CalculationResult> calculateAsyncLambda(String op, double... args) {
        final double[] argsCopy = args.clone();
        return executor.submit(() -> calculate(op, argsCopy));
    }

    // Batch calculation (Java 8 style)
    public List<CalculationResult> batchCalculate(List<CalculationRequest> requests) {
        List<CalculationResult> results = new ArrayList<>();
        for (CalculationRequest request : requests) {
            results.add(calculate(request.getOperation(), request.getArgs()));
        }
        return results;
    }

    /**
     * Request class (no records in Java 8).
     */
    public static class CalculationRequest {
        private final String operation;
        private final double[] args;

        public CalculationRequest(String operation, double[] args) {
            this.operation = operation;
            this.args = args != null ? args.clone() : new double[0];
        }

        public String getOperation() {
            return operation;
        }

        public double[] getArgs() {
            return args.clone();
        }
    }

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

        public CalculationException(String message, Throwable cause) {
            super(message, cause);
        }
    }
}

/**
 * Scientific calculator extending AdvancedCalculator.
 */
class ScientificCalculator extends AdvancedCalculator {

    /**
     * Enum for angle modes (enums work in Java 8).
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
        if (angleMode == AngleMode.DEGREES) {
            return Math.toRadians(angle);
        }
        return angle;
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
        long result = 1;
        for (int i = 2; i <= n; i++) {
            result *= i;
        }
        return result;
    }

    /**
     * Fibonacci using iteration (no Stream.iterate in Java 8).
     */
    public long fibonacci(int n) {
        if (n < 0) {
            throw new CalculationException("Negative fibonacci index");
        }
        if (n <= 1) {
            return n;
        }
        long a = 0;
        long b = 1;
        for (int i = 2; i <= n; i++) {
            long temp = a + b;
            a = b;
            b = temp;
        }
        return b;
    }

    /**
     * Check if a number is prime.
     */
    public boolean isPrime(int n) {
        if (n < 2) return false;
        if (n == 2) return true;
        if (n % 2 == 0) return false;

        int sqrt = (int) Math.sqrt(n);
        for (int i = 3; i <= sqrt; i += 2) {
            if (n % i == 0) return false;
        }
        return true;
    }

    /**
     * Get prime factors.
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
        return factors;
    }

    /**
     * GCD using recursion.
     */
    public int gcd(int a, int b) {
        a = Math.abs(a);
        b = Math.abs(b);
        if (b == 0) {
            return a;
        }
        return gcd(b, a % b);
    }

    /**
     * LCM using GCD.
     */
    public int lcm(int a, int b) {
        if (a == 0 || b == 0) return 0;
        return Math.abs(a * b) / gcd(a, b);
    }
}

/**
 * Calculator interface.
 */
interface Calculator {
    AdvancedCalculator.CalculationResult calculate(String op, double... args);
    void reset();
    List<AdvancedCalculator.HistoryEntry> getHistory();
}
