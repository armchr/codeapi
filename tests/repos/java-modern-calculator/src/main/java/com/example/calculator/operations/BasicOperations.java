package com.example.calculator.operations;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.Arrays;
import java.util.List;
import java.util.Optional;
import java.util.function.BiFunction;
import java.util.function.BinaryOperator;
import java.util.function.Function;
import java.util.stream.Collectors;

/**
 * Basic arithmetic operations using modern Java features.
 */
public final class BasicOperations {

    // Private constructor to prevent instantiation
    private BasicOperations() {
        throw new AssertionError("Cannot instantiate BasicOperations");
    }

    /**
     * Add two numbers.
     */
    public static double add(double a, double b) {
        return a + b;
    }

    /**
     * Subtract b from a.
     */
    public static double subtract(double a, double b) {
        return a - b;
    }

    /**
     * Multiply two numbers.
     */
    public static double multiply(double a, double b) {
        return a * b;
    }

    /**
     * Divide a by b, returning Optional.empty() on division by zero.
     */
    public static Optional<Double> divide(double a, double b) {
        if (b == 0) {
            return Optional.empty();
        }
        return Optional.of(a / b);
    }

    /**
     * Calculate power.
     */
    public static double power(double base, double exponent) {
        return Math.pow(base, exponent);
    }

    /**
     * Calculate modulo with Optional result.
     */
    public static Optional<Double> modulo(double a, double b) {
        if (b == 0) {
            return Optional.empty();
        }
        return Optional.of(a % b);
    }

    /**
     * Sum variable number of arguments using varargs.
     */
    public static double sum(double... numbers) {
        return Arrays.stream(numbers).sum();
    }

    /**
     * Product of all numbers using streams.
     */
    public static double product(double... numbers) {
        if (numbers.length == 0) {
            return 0;
        }
        return Arrays.stream(numbers).reduce(1, (a, b) -> a * b);
    }

    /**
     * Get operation by symbol - returns functional interface.
     */
    public static Optional<BinaryOperator<Double>> getOperation(String operator) {
        return Optional.ofNullable(switch (operator) {
            case "+" -> BasicOperations::add;
            case "-" -> BasicOperations::subtract;
            case "*" -> BasicOperations::multiply;
            case "/" -> (a, b) -> a / b;
            case "**" -> BasicOperations::power;
            case "%" -> (a, b) -> a % b;
            default -> null;
        });
    }

    /**
     * Create a curried operation (function returning function).
     */
    public static Function<Double, Function<Double, Double>> createOperation(String operator) {
        var op = getOperation(operator)
                .orElseThrow(() -> new IllegalArgumentException("Unknown operator: " + operator));
        return a -> b -> op.apply(a, b);
    }

    /**
     * Precise decimal operations using BigDecimal.
     */
    public static String preciseAdd(String a, String b) {
        return new BigDecimal(a).add(new BigDecimal(b)).toString();
    }

    public static String preciseMultiply(String a, String b) {
        return new BigDecimal(a).multiply(new BigDecimal(b)).toString();
    }

    public static String preciseDivide(String a, String b, int scale) {
        return new BigDecimal(a)
                .divide(new BigDecimal(b), scale, RoundingMode.HALF_UP)
                .toString();
    }

    // Lambda expressions stored as static fields
    public static final Function<Double, Double> NEGATE = x -> -x;
    public static final Function<Double, Double> ABSOLUTE = Math::abs;
    public static final Function<Double, Double> SQUARE = x -> x * x;
    public static final Function<Double, Optional<Double>> SQRT =
            x -> x >= 0 ? Optional.of(Math.sqrt(x)) : Optional.empty();

    /**
     * Compose multiple functions (right to left).
     */
    @SafeVarargs
    public static <T> Function<T, T> compose(Function<T, T>... functions) {
        return Arrays.stream(functions)
                .reduce(Function.identity(), (f, g) -> x -> f.apply(g.apply(x)));
    }

    /**
     * Pipe functions (left to right).
     */
    @SafeVarargs
    public static <T> Function<T, T> pipe(Function<T, T>... functions) {
        return Arrays.stream(functions)
                .reduce(Function.identity(), Function::andThen);
    }

    /**
     * Map operation over list - demonstrates method reference.
     */
    public static <T, R> List<R> map(List<T> list, Function<T, R> mapper) {
        return list.stream().map(mapper).collect(Collectors.toList());
    }

    /**
     * Filter list with predicate.
     */
    public static <T> List<T> filter(List<T> list, java.util.function.Predicate<T> predicate) {
        return list.stream().filter(predicate).collect(Collectors.toList());
    }

    /**
     * Reduce list to single value.
     */
    public static <T> T reduce(List<T> list, T identity, BinaryOperator<T> accumulator) {
        return list.stream().reduce(identity, accumulator);
    }

    /**
     * Batch operation on pairs - demonstrates record and stream operations.
     */
    public static List<OperationResult> batchOperation(
            List<double[]> pairs,
            String operator) {
        var op = getOperation(operator);

        return pairs.stream()
                .map(pair -> {
                    if (pair.length != 2) {
                        return new OperationResult(0, operator, pair, false, "Invalid pair");
                    }
                    return op.map(o -> new OperationResult(
                            o.apply(pair[0], pair[1]),
                            operator,
                            pair,
                            true,
                            null
                    )).orElse(new OperationResult(0, operator, pair, false, "Unknown operator"));
                })
                .toList();
    }

    /**
     * Record for operation result - modern Java feature.
     */
    public record OperationResult(
            double value,
            String operation,
            double[] inputs,
            boolean success,
            String errorMessage
    ) {
        // Compact constructor with validation
        public OperationResult {
            if (inputs == null) {
                inputs = new double[0];
            }
        }

        // Instance method on record
        public String format() {
            if (success) {
                return "%s(%s) = %f".formatted(
                        operation,
                        Arrays.toString(inputs),
                        value
                );
            }
            return "Error: " + errorMessage;
        }
    }
}
