package com.example.calculator.util;

import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.function.Function;
import java.util.function.Predicate;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Utility class for validation and parsing.
 * Java 8 style without records or newer features.
 */
public final class ValidationUtils {

    // Private constructor
    private ValidationUtils() {
        throw new AssertionError("Cannot instantiate ValidationUtils");
    }

    /**
     * Check if a value is a valid finite number.
     */
    public static boolean isValidNumber(double value) {
        return Double.isFinite(value) && !Double.isNaN(value);
    }

    /**
     * Validate number with custom predicate.
     */
    public static boolean validateNumber(double value, Predicate<Double> validator) {
        return isValidNumber(value) && validator.test(value);
    }

    /**
     * Parse string to double safely.
     */
    public static Optional<Double> parseDouble(String input) {
        if (input == null || input.trim().isEmpty()) {
            return Optional.empty();
        }
        try {
            double value = Double.parseDouble(input.trim());
            if (isValidNumber(value)) {
                return Optional.of(value);
            }
            return Optional.empty();
        } catch (NumberFormatException e) {
            return Optional.empty();
        }
    }

    /**
     * Class for parsed expressions (not a record in Java 8).
     */
    public static class ParsedExpression {
        private final String operator;
        private final double[] operands;

        public ParsedExpression(String operator, double[] operands) {
            if (operator == null || operator.trim().isEmpty()) {
                throw new IllegalArgumentException("Operator cannot be null or blank");
            }
            this.operator = operator;
            this.operands = operands != null ? operands.clone() : new double[0];
        }

        public String getOperator() {
            return operator;
        }

        public double[] getOperands() {
            return operands.clone();
        }

        @Override
        public String toString() {
            return "ParsedExpression{" +
                    "operator='" + operator + '\'' +
                    ", operands=" + java.util.Arrays.toString(operands) +
                    '}';
        }
    }

    // Patterns for expression parsing
    private static final Pattern FUNC_PATTERN = Pattern.compile("^(\\w+)\\s*\\(\\s*(.+)\\s*\\)$");
    private static final String[] OPERATORS = {"**", "+", "-", "*", "/", "%"};

    /**
     * Parse a mathematical expression.
     */
    public static Optional<ParsedExpression> parseExpression(String expr) {
        if (expr == null || expr.trim().isEmpty()) {
            return Optional.empty();
        }

        expr = expr.trim();

        // Try function-style: func(args)
        Matcher funcMatcher = FUNC_PATTERN.matcher(expr);
        if (funcMatcher.matches()) {
            String funcName = funcMatcher.group(1);
            String argsStr = funcMatcher.group(2);
            double[] args = parseArgs(argsStr);
            if (args != null) {
                return Optional.of(new ParsedExpression(funcName, args));
            }
        }

        // Try binary operations
        for (String op : OPERATORS) {
            int idx = expr.indexOf(op);
            if (idx > 0) {
                String left = expr.substring(0, idx).trim();
                String right = expr.substring(idx + op.length()).trim();

                Optional<Double> leftOpt = parseDouble(left);
                Optional<Double> rightOpt = parseDouble(right);

                if (leftOpt.isPresent() && rightOpt.isPresent()) {
                    return Optional.of(new ParsedExpression(
                            op,
                            new double[]{leftOpt.get(), rightOpt.get()}
                    ));
                }
            }
        }

        return Optional.empty();
    }

    private static double[] parseArgs(String argsStr) {
        String[] parts = argsStr.split(",");
        double[] results = new double[parts.length];
        for (int i = 0; i < parts.length; i++) {
            Optional<Double> parsed = parseDouble(parts[i].trim());
            if (!parsed.isPresent()) {
                return null;
            }
            results[i] = parsed.get();
        }
        return results;
    }

    /**
     * Format options class (not a record in Java 8).
     */
    public static class FormatOptions {
        private final int precision;
        private final boolean useThousandsSeparator;
        private final String prefix;
        private final String suffix;

        public FormatOptions(int precision, boolean useThousandsSeparator, String prefix, String suffix) {
            this.precision = precision;
            this.useThousandsSeparator = useThousandsSeparator;
            this.prefix = prefix != null ? prefix : "";
            this.suffix = suffix != null ? suffix : "";
        }

        public static FormatOptions defaults() {
            return new FormatOptions(2, true, "", "");
        }

        public static FormatOptions withPrecision(int precision) {
            return new FormatOptions(precision, true, "", "");
        }

        public int getPrecision() {
            return precision;
        }

        public boolean isUseThousandsSeparator() {
            return useThousandsSeparator;
        }

        public String getPrefix() {
            return prefix;
        }

        public String getSuffix() {
            return suffix;
        }
    }

    /**
     * Format number for display with various options.
     */
    public static String formatNumber(double value, FormatOptions options) {
        String formatted;
        if (options.isUseThousandsSeparator()) {
            formatted = String.format("%,." + options.getPrecision() + "f", value);
        } else {
            formatted = String.format("%." + options.getPrecision() + "f", value);
        }
        return options.getPrefix() + formatted + options.getSuffix();
    }

    /**
     * Result interface (not sealed in Java 8).
     */
    public interface Result<T> {
        boolean isOk();
        boolean isErr();
        T unwrap();
        T unwrapOr(T defaultValue);
        <U> Result<U> map(Function<T, U> mapper);
        <U> Result<U> flatMap(Function<T, Result<U>> mapper);
    }

    /**
     * Ok result implementation.
     */
    public static class Ok<T> implements Result<T> {
        private final T value;

        public Ok(T value) {
            this.value = value;
        }

        public T getValue() {
            return value;
        }

        @Override
        public boolean isOk() {
            return true;
        }

        @Override
        public boolean isErr() {
            return false;
        }

        @Override
        public T unwrap() {
            return value;
        }

        @Override
        public T unwrapOr(T defaultValue) {
            return value;
        }

        @Override
        public <U> Result<U> map(Function<T, U> mapper) {
            return new Ok<>(mapper.apply(value));
        }

        @Override
        public <U> Result<U> flatMap(Function<T, Result<U>> mapper) {
            return mapper.apply(value);
        }

        @Override
        public String toString() {
            return "Ok{" + value + "}";
        }
    }

    /**
     * Error result implementation.
     */
    public static class Err<T> implements Result<T> {
        private final String error;

        public Err(String error) {
            this.error = error;
        }

        public String getError() {
            return error;
        }

        @Override
        public boolean isOk() {
            return false;
        }

        @Override
        public boolean isErr() {
            return true;
        }

        @Override
        public T unwrap() {
            throw new RuntimeException("Called unwrap on Err: " + error);
        }

        @Override
        public T unwrapOr(T defaultValue) {
            return defaultValue;
        }

        @Override
        @SuppressWarnings("unchecked")
        public <U> Result<U> map(Function<T, U> mapper) {
            return (Result<U>) this;
        }

        @Override
        @SuppressWarnings("unchecked")
        public <U> Result<U> flatMap(Function<T, Result<U>> mapper) {
            return (Result<U>) this;
        }

        @Override
        public String toString() {
            return "Err{" + error + "}";
        }
    }

    /**
     * Memoization helper using ConcurrentHashMap.
     */
    public static <T, R> Function<T, R> memoize(final Function<T, R> function) {
        final Map<T, R> cache = new ConcurrentHashMap<>();
        return new Function<T, R>() {
            @Override
            public R apply(T input) {
                return cache.computeIfAbsent(input, function);
            }
        };
    }

    /**
     * Alternative memoization with lambda (Java 8).
     */
    public static <T, R> Function<T, R> memoizeLambda(Function<T, R> function) {
        Map<T, R> cache = new ConcurrentHashMap<>();
        return input -> cache.computeIfAbsent(input, function);
    }
}
