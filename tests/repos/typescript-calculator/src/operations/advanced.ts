/**
 * Advanced calculator with class-based design and various OOP patterns.
 */

// Import types
import type {
    CalculatorConfig,
    OperationResult,
    HistoryEntry,
    Observable,
    Observer,
    Unsubscribe,
    AngleMode,
} from './types.js';

// Import values
import {
    CalculationError,
    ErrorCode,
    DEFAULT_CONFIG,
} from './types.js';

// Import from sibling module
import { add, subtract, multiply, divide, sum, product } from './basic.js';

/**
 * Abstract base class for calculators.
 */
export abstract class BaseCalculator {
    protected _precision: number;
    protected _history: HistoryEntry[] = [];

    constructor(config: Partial<CalculatorConfig> = {}) {
        this._precision = config.precision ?? DEFAULT_CONFIG.precision;
    }

    // Abstract methods
    abstract calculate(op: string, ...args: number[]): OperationResult;
    abstract reset(): void;

    // Protected method
    protected roundResult(value: number): number {
        const multiplier = Math.pow(10, this._precision);
        return Math.round(value * multiplier) / multiplier;
    }

    // Getter
    get precision(): number {
        return this._precision;
    }

    // Setter with validation
    set precision(value: number) {
        if (value < 0 || !Number.isInteger(value)) {
            throw new Error('Precision must be a non-negative integer');
        }
        this._precision = value;
    }

    get history(): HistoryEntry[] {
        return [...this._history];
    }
}

/**
 * Advanced calculator implementation.
 */
export class AdvancedCalculator extends BaseCalculator implements Observable<OperationResult> {
    private static instanceCount = 0;

    private _memory: number = 0;
    private _historyLimit: number;
    private _lastResult: number = 0;
    private _observers: Observer<OperationResult>[] = [];

    constructor(config: Partial<CalculatorConfig> = {}) {
        super(config);
        this._historyLimit = config.historyLimit ?? DEFAULT_CONFIG.historyLimit;
        AdvancedCalculator.instanceCount++;
    }

    // Static method
    static getInstanceCount(): number {
        return AdvancedCalculator.instanceCount;
    }

    // Static factory method
    static create(config?: Partial<CalculatorConfig>): AdvancedCalculator {
        return new AdvancedCalculator(config);
    }

    // Implement abstract method
    calculate(op: string, ...args: number[]): OperationResult {
        const startTime = performance.now();

        try {
            let value: number;

            // Switch statement for operation dispatch
            switch (op) {
                case '+':
                case 'add':
                    if (args.length !== 2) throw new Error('Add requires 2 arguments');
                    value = add(args[0], args[1]);
                    break;

                case '-':
                case 'subtract':
                    if (args.length !== 2) throw new Error('Subtract requires 2 arguments');
                    value = subtract(args[0], args[1]);
                    break;

                case '*':
                case 'multiply':
                    if (args.length !== 2) throw new Error('Multiply requires 2 arguments');
                    value = multiply(args[0], args[1]);
                    break;

                case '/':
                case 'divide': {
                    if (args.length !== 2) throw new Error('Divide requires 2 arguments');
                    const result = divide(args[0], args[1]);
                    if (result.isErr()) throw result.error;
                    value = result.unwrap();
                    break;
                }

                case '**':
                case 'pow':
                case 'power':
                    if (args.length !== 2) throw new Error('Power requires 2 arguments');
                    value = Math.pow(args[0], args[1]);
                    break;

                case '%':
                case 'mod':
                case 'modulo':
                    if (args.length !== 2) throw new Error('Modulo requires 2 arguments');
                    if (args[1] === 0) throw new CalculationError('Modulo by zero', ErrorCode.DivisionByZero);
                    value = args[0] % args[1];
                    break;

                // Unary operations
                case 'sqrt':
                    if (args.length !== 1) throw new Error('Sqrt requires 1 argument');
                    if (args[0] < 0) throw new CalculationError('Cannot sqrt negative', ErrorCode.NegativeInput);
                    value = Math.sqrt(args[0]);
                    break;

                case 'log':
                    if (args.length < 1 || args.length > 2) throw new Error('Log requires 1-2 arguments');
                    if (args[0] <= 0) throw new CalculationError('Log of non-positive', ErrorCode.InvalidInput);
                    value = args.length === 2 ? Math.log(args[0]) / Math.log(args[1]) : Math.log(args[0]);
                    break;

                case 'sin':
                    if (args.length !== 1) throw new Error('Sin requires 1 argument');
                    value = Math.sin(args[0]);
                    break;

                case 'cos':
                    if (args.length !== 1) throw new Error('Cos requires 1 argument');
                    value = Math.cos(args[0]);
                    break;

                case 'tan':
                    if (args.length !== 1) throw new Error('Tan requires 1 argument');
                    value = Math.tan(args[0]);
                    break;

                case 'abs':
                    if (args.length !== 1) throw new Error('Abs requires 1 argument');
                    value = Math.abs(args[0]);
                    break;

                case 'floor':
                    if (args.length !== 1) throw new Error('Floor requires 1 argument');
                    value = Math.floor(args[0]);
                    break;

                case 'ceil':
                    if (args.length !== 1) throw new Error('Ceil requires 1 argument');
                    value = Math.ceil(args[0]);
                    break;

                // Reduction operations
                case 'sum':
                    value = sum(...args);
                    break;

                case 'product':
                    value = product(...args);
                    break;

                case 'max':
                    if (args.length === 0) throw new Error('Max requires at least 1 argument');
                    value = Math.max(...args);
                    break;

                case 'min':
                    if (args.length === 0) throw new Error('Min requires at least 1 argument');
                    value = Math.min(...args);
                    break;

                case 'avg':
                case 'mean':
                    if (args.length === 0) throw new Error('Average requires at least 1 argument');
                    value = sum(...args) / args.length;
                    break;

                default:
                    throw new CalculationError(`Unknown operation: ${op}`, ErrorCode.InvalidOperation);
            }

            // Round result
            value = this.roundResult(value);
            this._lastResult = value;

            // Create result
            const result: OperationResult = {
                value,
                operation: op,
                inputs: args,
                success: true,
                duration: performance.now() - startTime,
            };

            // Add to history
            this.addToHistory(op, args, value);

            // Notify observers
            this.notify(result);

            return result;

        } catch (error) {
            const result: OperationResult = {
                value: 0,
                operation: op,
                inputs: args,
                success: false,
                error: error instanceof CalculationError
                    ? error
                    : new CalculationError(String(error), ErrorCode.InvalidOperation),
                duration: performance.now() - startTime,
            };

            this.notify(result);
            return result;
        }
    }

    // Implement abstract method
    reset(): void {
        this._memory = 0;
        this._lastResult = 0;
        this._history = [];
    }

    // Private method
    private addToHistory(op: string, args: number[], result: number): void {
        const entry: HistoryEntry = {
            expression: `${op}(${args.join(', ')})`,
            result,
            timestamp: new Date(),
        };

        this._history.push(entry);

        // Trim history if needed
        if (this._history.length > this._historyLimit) {
            this._history = this._history.slice(-this._historyLimit);
        }
    }

    // Memory operations
    get memory(): number {
        return this._memory;
    }

    get lastResult(): number {
        return this._lastResult;
    }

    memoryAdd(value: number): void {
        this._memory += value;
    }

    memorySubtract(value: number): void {
        this._memory -= value;
    }

    memoryClear(): void {
        this._memory = 0;
    }

    memoryRecall(): number {
        return this._memory;
    }

    // Observer pattern implementation
    subscribe(observer: Observer<OperationResult>): Unsubscribe {
        this._observers.push(observer);
        return () => {
            const index = this._observers.indexOf(observer);
            if (index > -1) {
                this._observers.splice(index, 1);
            }
        };
    }

    notify(value: OperationResult): void {
        for (const observer of this._observers) {
            try {
                observer(value);
            } catch (e) {
                console.error('Observer error:', e);
            }
        }
    }
}

/**
 * Scientific calculator extending AdvancedCalculator.
 */
export class ScientificCalculator extends AdvancedCalculator {
    private _angleMode: AngleMode;

    constructor(config: Partial<CalculatorConfig & { angleMode?: AngleMode }> = {}) {
        super(config);
        this._angleMode = config.angleMode ?? 'radians';
    }

    get angleMode(): AngleMode {
        return this._angleMode;
    }

    set angleMode(mode: AngleMode) {
        if (mode !== 'radians' && mode !== 'degrees') {
            throw new Error("Angle mode must be 'radians' or 'degrees'");
        }
        this._angleMode = mode;
    }

    private toRadians(angle: number): number {
        return this._angleMode === 'degrees' ? angle * Math.PI / 180 : angle;
    }

    // Method overloading via overload signatures
    factorial(n: number): number;
    factorial(n: bigint): bigint;
    factorial(n: number | bigint): number | bigint {
        if (typeof n === 'bigint') {
            if (n < 0n) throw new CalculationError('Negative factorial', ErrorCode.NegativeInput);
            let result = 1n;
            for (let i = 2n; i <= n; i++) {
                result *= i;
            }
            return result;
        }

        if (n < 0) throw new CalculationError('Negative factorial', ErrorCode.NegativeInput);
        if (n > 170) throw new CalculationError('Factorial overflow', ErrorCode.Overflow);

        let result = 1;
        for (let i = 2; i <= n; i++) {
            result *= i;
        }
        return result;
    }

    fibonacci(n: number): number {
        if (n < 0) throw new CalculationError('Negative fibonacci', ErrorCode.NegativeInput);
        if (n <= 1) return n;

        let a = 0, b = 1;
        for (let i = 2; i <= n; i++) {
            [a, b] = [b, a + b];
        }
        return b;
    }

    isPrime(n: number): boolean {
        if (n < 2) return false;
        if (n === 2) return true;
        if (n % 2 === 0) return false;

        const sqrt = Math.sqrt(n);
        for (let i = 3; i <= sqrt; i += 2) {
            if (n % i === 0) return false;
        }
        return true;
    }

    primeFactors(n: number): number[] {
        const factors: number[] = [];
        let d = 2;
        while (d * d <= n) {
            while (n % d === 0) {
                factors.push(d);
                n = Math.floor(n / d);
            }
            d++;
        }
        if (n > 1) factors.push(n);
        return factors;
    }

    gcd(a: number, b: number): number {
        a = Math.abs(a);
        b = Math.abs(b);
        while (b !== 0) {
            [a, b] = [b, a % b];
        }
        return a;
    }

    lcm(a: number, b: number): number {
        if (a === 0 || b === 0) return 0;
        return Math.abs(a * b) / this.gcd(a, b);
    }

    // Async calculation
    async calculateAsync(op: string, ...args: number[]): Promise<OperationResult> {
        // Simulate async operation
        await new Promise(resolve => setTimeout(resolve, 1));
        return this.calculate(op, ...args);
    }

    // Generator for Fibonacci sequence
    *fibonacciSequence(limit: number): Generator<number, void, unknown> {
        let a = 0, b = 1;
        for (let i = 0; i < limit; i++) {
            yield a;
            [a, b] = [b, a + b];
        }
    }

    // Async generator for prime numbers
    async *primeGenerator(limit: number): AsyncGenerator<number, void, unknown> {
        let count = 0;
        let n = 2;
        while (count < limit) {
            if (this.isPrime(n)) {
                yield n;
                count++;
            }
            n++;
            // Yield control periodically
            if (n % 100 === 0) {
                await new Promise(resolve => setTimeout(resolve, 0));
            }
        }
    }
}

// Mixin pattern
type Constructor<T = object> = new (...args: unknown[]) => T;

function WithLogging<TBase extends Constructor<BaseCalculator>>(Base: TBase) {
    return class extends Base {
        calculate(op: string, ...args: number[]): OperationResult {
            console.log(`Calculating: ${op}(${args.join(', ')})`);
            const result = super.calculate(op, ...args);
            console.log(`Result: ${result.value}`);
            return result;
        }
    };
}

function WithValidation<TBase extends Constructor<BaseCalculator>>(Base: TBase) {
    return class extends Base {
        calculate(op: string, ...args: number[]): OperationResult {
            for (const arg of args) {
                if (!Number.isFinite(arg)) {
                    return {
                        value: 0,
                        operation: op,
                        inputs: args,
                        success: false,
                        error: new CalculationError('Invalid input', ErrorCode.InvalidInput),
                    };
                }
            }
            return super.calculate(op, ...args);
        }
    };
}

// Create calculator class with mixins
export const LoggingCalculator = WithLogging(AdvancedCalculator);
export const ValidatingCalculator = WithValidation(AdvancedCalculator);
export const LoggingValidatingCalculator = WithLogging(WithValidation(AdvancedCalculator));

// Singleton pattern
export class CalculatorSingleton {
    private static instance: AdvancedCalculator | null = null;

    private constructor() {}

    static getInstance(config?: Partial<CalculatorConfig>): AdvancedCalculator {
        if (!CalculatorSingleton.instance) {
            CalculatorSingleton.instance = new AdvancedCalculator(config);
        }
        return CalculatorSingleton.instance;
    }

    static resetInstance(): void {
        CalculatorSingleton.instance = null;
    }
}
