/**
 * Utility functions and helpers.
 */

import type { NumberInput, OperationResult } from '../operations/types.js';

/**
 * Validate that a value is a valid finite number.
 */
export function isValidNumber(value: unknown): value is number {
    return (
        typeof value === 'number' &&
        Number.isFinite(value) &&
        !Number.isNaN(value)
    );
}

/**
 * Parse a number from various input types.
 */
export function parseNumber(input: NumberInput): number {
    if (typeof input === 'number') {
        return input;
    }
    if (typeof input === 'bigint') {
        return Number(input);
    }
    const parsed = parseFloat(input);
    if (Number.isNaN(parsed)) {
        throw new Error(`Invalid number: ${input}`);
    }
    return parsed;
}

/**
 * Format a number for display.
 */
export function formatNumber(
    value: number,
    options: {
        precision?: number;
        thousandsSeparator?: boolean;
        prefix?: string;
        suffix?: string;
    } = {}
): string {
    const {
        precision = 2,
        thousandsSeparator = true,
        prefix = '',
        suffix = '',
    } = options;

    let formatted: string;

    if (thousandsSeparator) {
        formatted = value.toLocaleString('en-US', {
            minimumFractionDigits: precision,
            maximumFractionDigits: precision,
        });
    } else {
        formatted = value.toFixed(precision);
    }

    return `${prefix}${formatted}${suffix}`;
}

/**
 * Expression parser interface.
 */
interface ParsedExpression {
    operator: string;
    operands: number[];
}

/**
 * Parse a simple mathematical expression.
 */
export function parseExpression(expr: string): ParsedExpression {
    expr = expr.trim();

    // Function-style: func(args)
    const funcMatch = expr.match(/^(\w+)\s*\(\s*(.+)\s*\)$/);
    if (funcMatch) {
        const [, funcName, argsStr] = funcMatch;
        const operands = argsStr.split(',').map((s) => parseNumber(s.trim()));
        return { operator: funcName, operands };
    }

    // Binary operations: a op b
    const operators = ['**', '+', '-', '*', '/', '%'];
    for (const op of operators) {
        const idx = expr.indexOf(op);
        if (idx > 0) {
            const left = expr.slice(0, idx).trim();
            const right = expr.slice(idx + op.length).trim();
            try {
                const a = parseNumber(left);
                const b = parseNumber(right);
                return { operator: op, operands: [a, b] };
            } catch {
                continue;
            }
        }
    }

    throw new Error(`Cannot parse expression: ${expr}`);
}

// Decorator factory (using experimental decorators or as a function)
export function memoize<T extends (...args: unknown[]) => unknown>(
    fn: T,
    keyFn: (...args: Parameters<T>) => string = (...args) => JSON.stringify(args)
): T {
    const cache = new Map<string, ReturnType<T>>();

    return ((...args: Parameters<T>): ReturnType<T> => {
        const key = keyFn(...args);
        if (cache.has(key)) {
            return cache.get(key)!;
        }
        const result = fn(...args) as ReturnType<T>;
        cache.set(key, result);
        return result;
    }) as T;
}

/**
 * Debounce function calls.
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
    fn: T,
    delay: number
): (...args: Parameters<T>) => void {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    return (...args: Parameters<T>): void => {
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
        timeoutId = setTimeout(() => {
            fn(...args);
            timeoutId = null;
        }, delay);
    };
}

/**
 * Throttle function calls.
 */
export function throttle<T extends (...args: unknown[]) => unknown>(
    fn: T,
    limit: number
): (...args: Parameters<T>) => void {
    let inThrottle = false;

    return (...args: Parameters<T>): void => {
        if (!inThrottle) {
            fn(...args);
            inThrottle = true;
            setTimeout(() => {
                inThrottle = false;
            }, limit);
        }
    };
}

/**
 * Retry a function with exponential backoff.
 */
export async function retry<T>(
    fn: () => Promise<T>,
    options: {
        maxAttempts?: number;
        baseDelay?: number;
        maxDelay?: number;
    } = {}
): Promise<T> {
    const { maxAttempts = 3, baseDelay = 100, maxDelay = 5000 } = options;

    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        try {
            return await fn();
        } catch (error) {
            lastError = error instanceof Error ? error : new Error(String(error));

            if (attempt < maxAttempts) {
                const delay = Math.min(
                    baseDelay * Math.pow(2, attempt - 1),
                    maxDelay
                );
                await new Promise((resolve) => setTimeout(resolve, delay));
            }
        }
    }

    throw lastError;
}

/**
 * Deep clone an object using structured clone or JSON.
 */
export function deepClone<T>(obj: T): T {
    if (typeof structuredClone === 'function') {
        return structuredClone(obj);
    }
    return JSON.parse(JSON.stringify(obj));
}

/**
 * Deep merge objects.
 */
export function deepMerge<T extends Record<string, unknown>>(
    target: T,
    ...sources: Partial<T>[]
): T {
    const result = { ...target };

    for (const source of sources) {
        for (const key in source) {
            if (Object.prototype.hasOwnProperty.call(source, key)) {
                const targetValue = result[key];
                const sourceValue = source[key];

                if (
                    isPlainObject(targetValue) &&
                    isPlainObject(sourceValue)
                ) {
                    (result as Record<string, unknown>)[key] = deepMerge(
                        targetValue as Record<string, unknown>,
                        sourceValue as Record<string, unknown>
                    );
                } else if (sourceValue !== undefined) {
                    (result as Record<string, unknown>)[key] = sourceValue;
                }
            }
        }
    }

    return result;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
    return (
        typeof value === 'object' &&
        value !== null &&
        !Array.isArray(value) &&
        Object.getPrototypeOf(value) === Object.prototype
    );
}

/**
 * Create a simple event emitter.
 */
export class EventEmitter<T extends Record<string, unknown[]>> {
    private listeners = new Map<keyof T, Array<(...args: unknown[]) => void>>();

    on<K extends keyof T>(event: K, callback: (...args: T[K]) => void): () => void {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, []);
        }
        this.listeners.get(event)!.push(callback as (...args: unknown[]) => void);

        // Return unsubscribe function
        return () => {
            const callbacks = this.listeners.get(event);
            if (callbacks) {
                const index = callbacks.indexOf(callback as (...args: unknown[]) => void);
                if (index > -1) {
                    callbacks.splice(index, 1);
                }
            }
        };
    }

    emit<K extends keyof T>(event: K, ...args: T[K]): void {
        const callbacks = this.listeners.get(event);
        if (callbacks) {
            for (const callback of callbacks) {
                callback(...args);
            }
        }
    }

    off<K extends keyof T>(event: K, callback?: (...args: T[K]) => void): void {
        if (!callback) {
            this.listeners.delete(event);
        } else {
            const callbacks = this.listeners.get(event);
            if (callbacks) {
                const index = callbacks.indexOf(callback as (...args: unknown[]) => void);
                if (index > -1) {
                    callbacks.splice(index, 1);
                }
            }
        }
    }
}

/**
 * Promise utilities.
 */
export function timeout<T>(promise: Promise<T>, ms: number): Promise<T> {
    return Promise.race([
        promise,
        new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error('Timeout')), ms)
        ),
    ]);
}

export function delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function* asyncPool<T, R>(
    concurrency: number,
    items: T[],
    fn: (item: T) => Promise<R>
): AsyncGenerator<R, void, unknown> {
    const executing = new Set<Promise<R>>();

    for (const item of items) {
        const promise = fn(item).then((result) => {
            executing.delete(promise);
            return result;
        });
        executing.add(promise);

        if (executing.size >= concurrency) {
            yield await Promise.race(executing);
        }
    }

    while (executing.size > 0) {
        yield await Promise.race(executing);
    }
}
