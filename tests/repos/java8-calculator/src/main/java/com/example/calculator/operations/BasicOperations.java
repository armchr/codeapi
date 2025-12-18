package com.example.calculator.operations;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.function.BiFunction;
import java.util.function.BinaryOperator;
import java.util.function.Function;
import java.util.stream.Collectors;

/**
 * Basic arithmetic operations using Java 8 style.
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
        double result = 0;
        for (double num : numbers) {
            result += num;
        }
        return result;
    }

    /**
     * Product of all numbers.
     */
    public static double product(double... numbers) {
        if (numbers.length == 0) {
            return 0;
        }
        double result = 1;
        for (double num : numbers) {
            result *= num;
        }
        return result;
    }

    // Operation map using HashMap (Java 8 style)
    private static final Map<String, BinaryOperator<Double>> OPERATIONS;

    static {
        OPERATIONS = new HashMap<>();
        OPERATIONS.put("+", BasicOperations::add);
        OPERATIONS.put("-", BasicOperations::subtract);
        OPERATIONS.put("*", BasicOperations::multiply);
        OPERATIONS.put("/", (a, b) -> a / b);
        OPERATIONS.put("**", BasicOperations::power);
        OPERATIONS.put("%", (a, b) -> a % b);
    }

    /**
     * Get operation by symbol - returns Optional.
     */
    public static Optional<BinaryOperator<Double>> getOperation(String operator) {
        return Optional.ofNullable(OPERATIONS.get(operator));
    }

    /**
     * Create a curried operation (function returning function).
     * Java 8 style with explicit type declarations.
     */
    public static Function<Double, Function<Double, Double>> createOperation(String operator) {
        BinaryOperator<Double> op = getOperation(operator)
                .orElseThrow(() -> new IllegalArgumentException("Unknown operator: " + operator));
        return new Function<Double, Function<Double, Double>>() {
            @Override
            public Function<Double, Double> apply(Double a) {
                return new Function<Double, Double>() {
                    @Override
                    public Double apply(Double b) {
                        return op.apply(a, b);
                    }
                };
            }
        };
    }

    /**
     * Alternative curried operation using lambdas.
     */
    public static Function<Double, Function<Double, Double>> createOperationLambda(String operator) {
        BinaryOperator<Double> op = getOperation(operator)
                .orElseThrow(() -> new IllegalArgumentException("Unknown operator: " + operator));
        return a -> b -> op.apply(a, b);
    }

    /**
     * Precise decimal operations using BigDecimal.
     */
    public static String preciseAdd(String a, String b) {
        BigDecimal bdA = new BigDecimal(a);
        BigDecimal bdB = new BigDecimal(b);
        return bdA.add(bdB).toString();
    }

    public static String preciseMultiply(String a, String b) {
        BigDecimal bdA = new BigDecimal(a);
        BigDecimal bdB = new BigDecimal(b);
        return bdA.multiply(bdB).toString();
    }

    public static String preciseDivide(String a, String b, int scale) {
        BigDecimal bdA = new BigDecimal(a);
        BigDecimal bdB = new BigDecimal(b);
        return bdA.divide(bdB, scale, RoundingMode.HALF_UP).toString();
    }

    // Lambda expressions stored as static fields (Java 8 style)
    public static final Function<Double, Double> NEGATE = new Function<Double, Double>() {
        @Override
        public Double apply(Double x) {
            return -x;
        }
    };

    public static final Function<Double, Double> ABSOLUTE = Math::abs;

    public static final Function<Double, Double> SQUARE = x -> x * x;

    public static final Function<Double, Optional<Double>> SQRT = x -> {
        if (x >= 0) {
            return Optional.of(Math.sqrt(x));
        } else {
            return Optional.empty();
        }
    };

    /**
     * Compose multiple functions (right to left).
     * Java 8 style with explicit loop.
     */
    @SafeVarargs
    public static <T> Function<T, T> compose(Function<T, T>... functions) {
        Function<T, T> result = Function.identity();
        for (int i = functions.length - 1; i >= 0; i--) {
            result = result.compose(functions[i]);
        }
        return result;
    }

    /**
     * Pipe functions (left to right).
     */
    @SafeVarargs
    public static <T> Function<T, T> pipe(Function<T, T>... functions) {
        Function<T, T> result = Function.identity();
        for (Function<T, T> function : functions) {
            result = result.andThen(function);
        }
        return result;
    }

    /**
     * Map operation over list.
     */
    public static <T, R> List<R> map(List<T> list, Function<T, R> mapper) {
        List<R> result = new ArrayList<>();
        for (T item : list) {
            result.add(mapper.apply(item));
        }
        return result;
    }

    /**
     * Map operation using streams (Java 8).
     */
    public static <T, R> List<R> mapStream(List<T> list, Function<T, R> mapper) {
        return list.stream().map(mapper).collect(Collectors.toList());
    }

    /**
     * Filter list with predicate.
     */
    public static <T> List<T> filter(List<T> list, java.util.function.Predicate<T> predicate) {
        List<T> result = new ArrayList<>();
        for (T item : list) {
            if (predicate.test(item)) {
                result.add(item);
            }
        }
        return result;
    }

    /**
     * Reduce list to single value.
     */
    public static <T> T reduce(List<T> list, T identity, BinaryOperator<T> accumulator) {
        T result = identity;
        for (T item : list) {
            result = accumulator.apply(result, item);
        }
        return result;
    }

    /**
     * Result class (no records in Java 8).
     */
    public static class OperationResult {
        private final double value;
        private final String operation;
        private final double[] inputs;
        private final boolean success;
        private final String errorMessage;

        public OperationResult(double value, String operation, double[] inputs,
                               boolean success, String errorMessage) {
            this.value = value;
            this.operation = operation;
            this.inputs = inputs != null ? inputs.clone() : new double[0];
            this.success = success;
            this.errorMessage = errorMessage;
        }

        public double getValue() {
            return value;
        }

        public String getOperation() {
            return operation;
        }

        public double[] getInputs() {
            return inputs.clone();
        }

        public boolean isSuccess() {
            return success;
        }

        public String getErrorMessage() {
            return errorMessage;
        }

        public String format() {
            if (success) {
                return String.format("%s(%s) = %f", operation, Arrays.toString(inputs), value);
            }
            return "Error: " + errorMessage;
        }

        @Override
        public String toString() {
            return "OperationResult{" +
                    "value=" + value +
                    ", operation='" + operation + '\'' +
                    ", inputs=" + Arrays.toString(inputs) +
                    ", success=" + success +
                    ", errorMessage='" + errorMessage + '\'' +
                    '}';
        }
    }

    /**
     * Batch operation on pairs.
     */
    public static List<OperationResult> batchOperation(List<double[]> pairs, String operator) {
        Optional<BinaryOperator<Double>> opOptional = getOperation(operator);

        List<OperationResult> results = new ArrayList<>();

        for (double[] pair : pairs) {
            if (pair.length != 2) {
                results.add(new OperationResult(0, operator, pair, false, "Invalid pair"));
                continue;
            }

            if (opOptional.isPresent()) {
                BinaryOperator<Double> op = opOptional.get();
                try {
                    double result = op.apply(pair[0], pair[1]);
                    results.add(new OperationResult(result, operator, pair, true, null));
                } catch (Exception e) {
                    results.add(new OperationResult(0, operator, pair, false, e.getMessage()));
                }
            } else {
                results.add(new OperationResult(0, operator, pair, false, "Unknown operator"));
            }
        }

        return results;
    }
}
