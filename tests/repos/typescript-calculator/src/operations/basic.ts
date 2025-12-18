/**
 * Basic arithmetic operations module.
 */

// Named imports
import {
    NumberInput,
    OperatorSymbol,
    OperationResult,
    CalculationError,
    ErrorCode,
    Ok,
    Err,
    type Result,
} from './types.js';

// Default import
import Decimal from 'decimal.js';

// Re-export
export { CalculationError, ErrorCode } from './types.js';

/**
 * Add two numbers.
 */
export function add(a: number, b: number): number {
    return a + b;
}

/**
 * Subtract b from a.
 */
export function subtract(a: number, b: number): number {
    return a - b;
}

/**
 * Multiply two numbers.
 */
export function multiply(a: number, b: number): number {
    return a * b;
}

/**
 * Divide a by b.
 * Returns Result type for safe error handling.
 */
export function divide(a: number, b: number): Result<number, CalculationError> {
    if (b === 0) {
        return new Err(
            new CalculationError(
                'Division by zero',
                ErrorCode.DivisionByZero,
                'divide',
                [a, b]
            )
        );
    }
    return new Ok(a / b);
}

/**
 * Calculate power.
 */
export function power(base: number, exponent: number): number {
    return Math.pow(base, exponent);
}

/**
 * Calculate modulo.
 */
export function modulo(a: number, b: number): Result<number, CalculationError> {
    if (b === 0) {
        return new Err(
            new CalculationError(
                'Modulo by zero',
                ErrorCode.DivisionByZero,
                'modulo',
                [a, b]
            )
        );
    }
    return new Ok(a % b);
}

/**
 * Sum all provided numbers using rest parameters.
 */
export function sum(...numbers: number[]): number {
    return numbers.reduce((acc, n) => acc + n, 0);
}

/**
 * Calculate product of all numbers.
 */
export function product(...numbers: number[]): number {
    if (numbers.length === 0) return 0;
    return numbers.reduce((acc, n) => acc * n, 1);
}

/**
 * Get operation function by operator symbol.
 * Demonstrates function as return type.
 */
export function getOperationBySymbol(
    operator: OperatorSymbol
): ((a: number, b: number) => number) | null {
    const operations: Record<OperatorSymbol, (a: number, b: number) => number> = {
        '+': add,
        '-': subtract,
        '*': multiply,
        '/': (a, b) => a / b,
        '**': power,
        '%': (a, b) => a % b,
    };

    return operations[operator] ?? null;
}

/**
 * Create a curried operation function.
 */
export function createOperation(
    operator: OperatorSymbol
): (a: number) => (b: number) => number {
    const op = getOperationBySymbol(operator);
    if (!op) {
        throw new CalculationError(
            `Unknown operator: ${operator}`,
            ErrorCode.InvalidOperation
        );
    }
    return (a: number) => (b: number) => op(a, b);
}

/**
 * Precise decimal arithmetic using Decimal.js library.
 */
export function preciseAdd(a: NumberInput, b: NumberInput): string {
    return new Decimal(a.toString()).plus(new Decimal(b.toString())).toString();
}

export function preciseMultiply(a: NumberInput, b: NumberInput): string {
    return new Decimal(a.toString()).times(new Decimal(b.toString())).toString();
}

export function preciseDivide(
    a: NumberInput,
    b: NumberInput,
    precision: number = 20
): string {
    Decimal.set({ precision });
    return new Decimal(a.toString()).dividedBy(new Decimal(b.toString())).toString();
}

// Lambda/arrow function expressions
export const negate = (x: number): number => -x;
export const absolute = (x: number): number => Math.abs(x);
export const square = (x: number): number => x * x;
export const sqrt = (x: number): number | null => (x >= 0 ? Math.sqrt(x) : null);

// IIFE (Immediately Invoked Function Expression)
export const PI_SQUARED = (() => Math.PI * Math.PI)();

// Higher-order function
export function compose<T>(
    ...fns: Array<(arg: T) => T>
): (arg: T) => T {
    return (arg: T) => fns.reduceRight((acc, fn) => fn(acc), arg);
}

export function pipe<T>(
    ...fns: Array<(arg: T) => T>
): (arg: T) => T {
    return (arg: T) => fns.reduce((acc, fn) => fn(acc), arg);
}

// Generic higher-order functions
export function map<T, U>(fn: (item: T) => U): (arr: T[]) => U[] {
    return (arr: T[]) => arr.map(fn);
}

export function filter<T>(predicate: (item: T) => boolean): (arr: T[]) => T[] {
    return (arr: T[]) => arr.filter(predicate);
}

export function reduce<T, U>(
    reducer: (acc: U, item: T) => U,
    initial: U
): (arr: T[]) => U {
    return (arr: T[]) => arr.reduce(reducer, initial);
}

// Batch operation using array methods
export function batchOperation(
    pairs: [number, number][],
    operator: OperatorSymbol
): OperationResult[] {
    const op = getOperationBySymbol(operator);
    if (!op) {
        return pairs.map(([a, b]) => ({
            value: 0,
            operation: operator,
            inputs: [a, b],
            success: false,
            error: new CalculationError(
                'Unknown operator',
                ErrorCode.InvalidOperation
            ),
        }));
    }

    return pairs.map(([a, b]) => {
        try {
            const value = op(a, b);
            return {
                value,
                operation: operator,
                inputs: [a, b],
                success: true,
            };
        } catch (e) {
            return {
                value: 0,
                operation: operator,
                inputs: [a, b],
                success: false,
                error:
                    e instanceof CalculationError
                        ? e
                        : new CalculationError(
                              String(e),
                              ErrorCode.InvalidOperation
                          ),
            };
        }
    });
}

// Default export
export default {
    add,
    subtract,
    multiply,
    divide,
    power,
    modulo,
    sum,
    product,
};
