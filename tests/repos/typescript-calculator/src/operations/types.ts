/**
 * Type definitions for the calculator operations.
 */

// Type aliases
export type NumberInput = number | string | bigint;
export type OperatorSymbol = '+' | '-' | '*' | '/' | '**' | '%';
export type UnaryOperator = 'sqrt' | 'log' | 'sin' | 'cos' | 'tan' | 'abs' | 'neg';
export type AngleMode = 'radians' | 'degrees';

// Literal type
export type OperationType = 'unary' | 'binary' | 'reduction';

// Enum
export enum ErrorCode {
    DivisionByZero = 'DIVISION_BY_ZERO',
    InvalidInput = 'INVALID_INPUT',
    InvalidOperation = 'INVALID_OPERATION',
    NegativeInput = 'NEGATIVE_INPUT',
    Overflow = 'OVERFLOW',
}

// Interface with optional and readonly properties
export interface OperationResult<T = number> {
    readonly value: T;
    readonly operation: string;
    readonly inputs: readonly number[];
    readonly success: boolean;
    readonly error?: CalculationError;
    readonly duration?: number;
}

// Interface for calculator configuration
export interface CalculatorConfig {
    precision?: number;
    historyLimit?: number;
    angleMode?: AngleMode;
    enableCache?: boolean;
}

// Interface with index signature
export interface OperationRegistry {
    [operatorName: string]: OperationHandler;
}

// Function type
export type OperationHandler = (
    ...args: number[]
) => number | OperationResult;

// Generic interface
export interface Result<T, E = Error> {
    isOk(): this is OkResult<T>;
    isErr(): this is ErrResult<E>;
    unwrap(): T;
    unwrapOr(defaultValue: T): T;
    map<U>(fn: (value: T) => U): Result<U, E>;
    flatMap<U>(fn: (value: T) => Result<U, E>): Result<U, E>;
}

// Discriminated union types
interface OkResult<T> extends Result<T, never> {
    readonly ok: true;
    readonly value: T;
}

interface ErrResult<E> extends Result<never, E> {
    readonly ok: false;
    readonly error: E;
}

// Class implementing generic interface
export class Ok<T> implements OkResult<T> {
    readonly ok = true as const;

    constructor(public readonly value: T) {}

    isOk(): this is OkResult<T> {
        return true;
    }

    isErr(): this is ErrResult<never> {
        return false;
    }

    unwrap(): T {
        return this.value;
    }

    unwrapOr(_defaultValue: T): T {
        return this.value;
    }

    map<U>(fn: (value: T) => U): Result<U, never> {
        return new Ok(fn(this.value));
    }

    flatMap<U>(fn: (value: T) => Result<U, never>): Result<U, never> {
        return fn(this.value);
    }
}

export class Err<E> implements ErrResult<E> {
    readonly ok = false as const;

    constructor(public readonly error: E) {}

    isOk(): this is OkResult<never> {
        return false;
    }

    isErr(): this is ErrResult<E> {
        return true;
    }

    unwrap(): never {
        throw this.error;
    }

    unwrapOr<T>(defaultValue: T): T {
        return defaultValue;
    }

    map<U>(_fn: (value: never) => U): Result<U, E> {
        return this as unknown as Err<E>;
    }

    flatMap<U>(_fn: (value: never) => Result<U, E>): Result<U, E> {
        return this as unknown as Err<E>;
    }
}

// Custom error class
export class CalculationError extends Error {
    constructor(
        message: string,
        public readonly code: ErrorCode,
        public readonly operation?: string,
        public readonly inputs?: number[]
    ) {
        super(message);
        this.name = 'CalculationError';

        // Maintain proper stack trace
        if (Error.captureStackTrace) {
            Error.captureStackTrace(this, CalculationError);
        }
    }

    toString(): string {
        return `${this.name}[${this.code}]: ${this.message}`;
    }
}

// History entry interface
export interface HistoryEntry {
    expression: string;
    result: number;
    timestamp: Date;
    duration?: number;
}

// Observer pattern types
export type Observer<T> = (value: T) => void;
export type Unsubscribe = () => void;

export interface Observable<T> {
    subscribe(observer: Observer<T>): Unsubscribe;
    notify(value: T): void;
}

// Utility types
export type PartialConfig = Partial<CalculatorConfig>;
export type RequiredConfig = Required<CalculatorConfig>;
export type ReadonlyResult = Readonly<OperationResult>;

// Mapped type
export type Nullable<T> = {
    [P in keyof T]: T[P] | null;
};

// Conditional type
export type ResultType<T> = T extends number
    ? OperationResult<number>
    : T extends bigint
      ? OperationResult<bigint>
      : OperationResult<unknown>;

// Template literal type
export type OperationMethod = `calculate${Capitalize<string>}`;

// Infer type
export type ExtractResultValue<T> = T extends OperationResult<infer V>
    ? V
    : never;

// Intersection type
export type FullConfig = CalculatorConfig & {
    readonly version: string;
    readonly name: string;
};

// Union type with type guards
export type MathValue = number | bigint | null | undefined;

export function isNumber(value: MathValue): value is number {
    return typeof value === 'number' && !Number.isNaN(value);
}

export function isBigInt(value: MathValue): value is bigint {
    return typeof value === 'bigint';
}

export function isValidMathValue(value: MathValue): value is number | bigint {
    return isNumber(value) || isBigInt(value);
}

// Const assertion
export const DEFAULT_CONFIG = {
    precision: 10,
    historyLimit: 100,
    angleMode: 'radians',
    enableCache: true,
} as const;

// Satisfies operator (TypeScript 4.9+)
export const OPERATORS = {
    add: '+',
    subtract: '-',
    multiply: '*',
    divide: '/',
    power: '**',
    modulo: '%',
} as const satisfies Record<string, OperatorSymbol>;
