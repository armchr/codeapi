package com.example.calculator.util;

import java.util.Optional;
import java.util.function.Function;
import java.util.function.Predicate;
import java.util.regex.Pattern;

/**
 * Utility class for validation and parsing.
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
        if (input == null || input.isBlank()) {
            return Optional.empty();
        }
        try {
            double value = Double.parseDouble(input.strip());
            return isValidNumber(value) ? Optional.of(value) : Optional.empty();
        } catch (NumberFormatException e) {
            return Optional.empty();
        }
    }

    /**
     * Record for parsed expressions.
     */
    public record ParsedExpression(String operator, double[] operands) {
        public ParsedExpression {
            // Compact constructor validation
            if (operator == null || operator.isBlank()) {
                throw new IllegalArgumentException("Operator cannot be null or blank");
            }
            if (operands == null) {
                operands = new double[0];
            }
        }
    }

    // Patterns for expression parsing
    private static final Pattern FUNC_PATTERN = Pattern.compile("^(\\w+)\\s*\\(\\s*(.+)\\s*\\)$");
    private static final String[] OPERATORS = {"**", "+", "-", "*", "/", "%"};

    /**
     * Parse a mathematical expression.
     */
    public static Optional<ParsedExpression> parseExpression(String expr) {
        if (expr == null || expr.isBlank()) {
            return Optional.empty();
        }

        expr = expr.strip();

        // Try function-style: func(args)
        var funcMatcher = FUNC_PATTERN.matcher(expr);
        if (funcMatcher.matches()) {
            var funcName = funcMatcher.group(1);
            var argsStr = funcMatcher.group(2);
            var args = parseArgs(argsStr);
            if (args != null) {
                return Optional.of(new ParsedExpression(funcName, args));
            }
        }

        // Try binary operations
        for (var op : OPERATORS) {
            int idx = expr.indexOf(op);
            if (idx > 0) {
                var left = expr.substring(0, idx).strip();
                var right = expr.substring(idx + op.length()).strip();

                var leftOpt = parseDouble(left);
                var rightOpt = parseDouble(right);

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
        var parts = argsStr.split(",");
        var results = new double[parts.length];
        for (int i = 0; i < parts.length; i++) {
            var parsed = parseDouble(parts[i].strip());
            if (parsed.isEmpty()) {
                return null;
            }
            results[i] = parsed.get();
        }
        return results;
    }

    /**
     * Format number for display with various options.
     */
    public static String formatNumber(double value, FormatOptions options) {
        var formatted = options.useThousandsSeparator()
                ? String.format("%,." + options.precision() + "f", value)
                : String.format("%." + options.precision() + "f", value);
        return options.prefix() + formatted + options.suffix();
    }

    /**
     * Record for format options with defaults via static factory.
     */
    public record FormatOptions(
            int precision,
            boolean useThousandsSeparator,
            String prefix,
            String suffix
    ) {
        // Static factory with defaults
        public static FormatOptions defaults() {
            return new FormatOptions(2, true, "", "");
        }

        public static FormatOptions withPrecision(int precision) {
            return new FormatOptions(precision, true, "", "");
        }
    }

    /**
     * Result type similar to Rust's Result<T, E>.
     */
    public sealed interface Result<T> permits Ok, Err {
        boolean isOk();
        boolean isErr();
        T unwrap();
        T unwrapOr(T defaultValue);
        <U> Result<U> map(Function<T, U> mapper);
        <U> Result<U> flatMap(Function<T, Result<U>> mapper);
    }

    public record Ok<T>(T value) implements Result<T> {
        @Override
        public boolean isOk() { return true; }

        @Override
        public boolean isErr() { return false; }

        @Override
        public T unwrap() { return value; }

        @Override
        public T unwrapOr(T defaultValue) { return value; }

        @Override
        public <U> Result<U> map(Function<T, U> mapper) {
            return new Ok<>(mapper.apply(value));
        }

        @Override
        public <U> Result<U> flatMap(Function<T, Result<U>> mapper) {
            return mapper.apply(value);
        }
    }

    public record Err<T>(String error) implements Result<T> {
        @Override
        public boolean isOk() { return false; }

        @Override
        public boolean isErr() { return true; }

        @Override
        public T unwrap() {
            throw new RuntimeException("Called unwrap on Err: " + error);
        }

        @Override
        public T unwrapOr(T defaultValue) { return defaultValue; }

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
    }

    /**
     * Memoization helper using ConcurrentHashMap.
     */
    public static <T, R> Function<T, R> memoize(Function<T, R> function) {
        var cache = new java.util.concurrent.ConcurrentHashMap<T, R>();
        return input -> cache.computeIfAbsent(input, function);
    }
}
