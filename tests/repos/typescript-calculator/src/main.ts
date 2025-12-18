#!/usr/bin/env ts-node
/**
 * Main entry point for the TypeScript Calculator application.
 */

// Import from barrel export
import {
    AdvancedCalculator,
    ScientificCalculator,
    CalculationError,
    ErrorCode,
    DEFAULT_CONFIG,
    type OperationResult,
    type CalculatorConfig,
} from './operations/index.js';

// Import utilities
import {
    formatNumber,
    parseExpression,
    EventEmitter,
    memoize,
} from './utils/helpers.js';

// Import Node.js built-ins
import * as readline from 'readline';
import { stdin, stdout, argv } from 'process';

// Version
const VERSION = '1.0.0';
const APP_NAME = 'TypeScript Calculator';

// Global calculator instance
let calculator: ScientificCalculator | null = null;

function getCalculator(): ScientificCalculator {
    if (!calculator) {
        calculator = new ScientificCalculator({
            precision: 10,
            historyLimit: 100,
        });
    }
    return calculator;
}

// Memoized fibonacci
const fibonacciCached = memoize((n: number): number => {
    if (n <= 1) return n;
    return fibonacciCached(n - 1) + fibonacciCached(n - 2);
});

// Command handlers
type CommandHandler = (args: string[]) => string | Promise<string>;

const commands: Record<string, { description: string; handler: CommandHandler }> = {
    help: {
        description: 'Show help information',
        handler: handleHelp,
    },
    history: {
        description: 'Show calculation history',
        handler: handleHistory,
    },
    mc: {
        description: 'Clear memory',
        handler: () => {
            getCalculator().memoryClear();
            return 'Memory cleared';
        },
    },
    mr: {
        description: 'Recall memory',
        handler: () => `Memory: ${getCalculator().memoryRecall()}`,
    },
    'm+': {
        description: 'Add to memory',
        handler: (args) => {
            const value = parseFloat(args[0] ?? '0');
            getCalculator().memoryAdd(value);
            return `Added to memory: ${getCalculator().memory}`;
        },
    },
    'm-': {
        description: 'Subtract from memory',
        handler: (args) => {
            const value = parseFloat(args[0] ?? '0');
            getCalculator().memorySubtract(value);
            return `Subtracted from memory: ${getCalculator().memory}`;
        },
    },
    prime: {
        description: 'Check if a number is prime',
        handler: (args) => {
            const n = parseInt(args[0] ?? '0', 10);
            const isPrime = getCalculator().isPrime(n);
            return `${n} is ${isPrime ? '' : 'not '}prime`;
        },
    },
    factors: {
        description: 'Get prime factors',
        handler: (args) => {
            const n = parseInt(args[0] ?? '0', 10);
            const factors = getCalculator().primeFactors(n);
            return `Prime factors of ${n}: [${factors.join(', ')}]`;
        },
    },
    fib: {
        description: 'Calculate Fibonacci number',
        handler: (args) => {
            const n = parseInt(args[0] ?? '0', 10);
            return `Fibonacci(${n}) = ${fibonacciCached(n)}`;
        },
    },
};

function handleHelp(): string {
    return `
${APP_NAME} Commands:
  Basic:     2 + 3, 10 - 5, 4 * 3, 20 / 4
  Power:     2 ** 8, pow(2, 8)
  Functions: sqrt(16), log(100), sin(0.5), cos(0.5), tan(0.5)
  Stats:     sum(1,2,3), avg(1,2,3), max(1,2,3), min(1,2,3)

Memory:
  mc - Clear memory
  mr - Recall memory
  m+ <value> - Add to memory
  m- <value> - Subtract from memory

Other:
  history - Show calculation history
  prime <n> - Check if n is prime
  factors <n> - Get prime factors of n
  fib <n> - Calculate nth Fibonacci number
  help - Show this help
  quit - Exit calculator
`.trim();
}

function handleHistory(): string {
    const history = getCalculator().history;
    if (history.length === 0) {
        return 'No history';
    }
    return history.map((entry) => `  ${entry.expression} = ${entry.result}`).join('\n');
}

async function processCommand(input: string): Promise<{ result: string; exit: boolean }> {
    input = input.trim();

    // Empty input
    if (!input) {
        return { result: '', exit: false };
    }

    // Exit commands
    const exitCommands = ['exit', 'quit', 'q'];
    if (exitCommands.includes(input.toLowerCase())) {
        return { result: 'Goodbye!', exit: true };
    }

    // Check built-in commands
    const [cmdName, ...args] = input.split(/\s+/);
    const cmd = commands[cmdName.toLowerCase()];

    if (cmd) {
        try {
            const result = await cmd.handler(args);
            return { result, exit: false };
        } catch (error) {
            return {
                result: `Error: ${error instanceof Error ? error.message : String(error)}`,
                exit: false,
            };
        }
    }

    // Try to parse as expression
    try {
        const parsed = parseExpression(input);
        const result = getCalculator().calculate(parsed.operator, ...parsed.operands);

        if (result.success) {
            return { result: formatNumber(result.value, { precision: 6 }), exit: false };
        } else {
            return { result: `Error: ${result.error?.message ?? 'Unknown error'}`, exit: false };
        }
    } catch (error) {
        return {
            result: `Error: ${error instanceof Error ? error.message : String(error)}`,
            exit: false,
        };
    }
}

// Event emitter for calculator events
interface CalculatorEvents {
    calculate: [OperationResult];
    error: [Error];
    command: [string, string];
}

const events = new EventEmitter<CalculatorEvents>();

// Subscribe to calculation events
getCalculator().subscribe((result) => {
    events.emit('calculate', result);
});

async function runInteractive(): Promise<void> {
    const rl = readline.createInterface({
        input: stdin,
        output: stdout,
    });

    console.log(`${APP_NAME} v${VERSION}`);
    console.log("Type 'help' for commands, 'quit' to exit\n");

    const prompt = (): void => {
        rl.question('calc> ', async (input) => {
            const { result, exit } = await processCommand(input);

            if (result) {
                console.log(result);
            }

            if (exit) {
                rl.close();
                return;
            }

            prompt();
        });
    };

    prompt();

    // Handle close
    rl.on('close', () => {
        console.log('\nGoodbye!');
        process.exit(0);
    });
}

async function runBatch(expressions: string[]): Promise<void> {
    // Filter comments and empty lines
    const filtered = expressions.filter((e) => {
        const trimmed = e.trim();
        return trimmed && !trimmed.startsWith('#');
    });

    // Process with Promise.all for parallel execution
    const results = await Promise.all(
        filtered.map(async (expr) => {
            const { result } = await processCommand(expr);
            return `${expr} = ${result}`;
        })
    );

    results.forEach((r) => console.log(r));
}

function runDemo(): void {
    console.log('Calculator Demo');
    console.log('===============\n');

    const calc = getCalculator();

    // Basic operations
    console.log('Basic operations:');
    console.log(`  add(5, 3) = ${calc.calculate('+', 5, 3).value}`);
    console.log(`  subtract(10, 4) = ${calc.calculate('-', 10, 4).value}`);
    console.log(`  multiply(7, 8) = ${calc.calculate('*', 7, 8).value}`);
    console.log(`  divide(20, 4) = ${calc.calculate('/', 20, 4).value}`);
    console.log();

    // Scientific operations
    console.log('Scientific operations:');
    console.log(`  sqrt(144) = ${calc.calculate('sqrt', 144).value}`);
    console.log(`  log(Math.E) = ${calc.calculate('log', Math.E).value}`);
    console.log(`  sin(Math.PI/2) = ${calc.calculate('sin', Math.PI / 2).value}`);
    console.log();

    // Number theory
    console.log('Number theory:');
    console.log(`  factorial(10) = ${calc.factorial(10)}`);
    console.log(`  fibonacci(20) = ${calc.fibonacci(20)}`);
    console.log(`  isPrime(17) = ${calc.isPrime(17)}`);
    console.log(`  primeFactors(84) = [${calc.primeFactors(84).join(', ')}]`);
    console.log(`  gcd(48, 18) = ${calc.gcd(48, 18)}`);
    console.log(`  lcm(4, 6) = ${calc.lcm(4, 6)}`);
    console.log();

    // Reduction operations
    console.log('Reduction operations:');
    console.log(`  sum(1, 2, 3, 4, 5) = ${calc.calculate('sum', 1, 2, 3, 4, 5).value}`);
    console.log(`  avg(1, 2, 3, 4, 5) = ${calc.calculate('avg', 1, 2, 3, 4, 5).value}`);
    console.log(`  max(1, 5, 3, 9, 2) = ${calc.calculate('max', 1, 5, 3, 9, 2).value}`);
    console.log(`  min(1, 5, 3, 9, 2) = ${calc.calculate('min', 1, 5, 3, 9, 2).value}`);
    console.log();

    // Generators
    console.log('Fibonacci sequence (first 10):');
    const fibs: number[] = [];
    for (const n of calc.fibonacciSequence(10)) {
        fibs.push(n);
    }
    console.log(`  [${fibs.join(', ')}]`);
    console.log();

    // Higher-order functions with arrays
    console.log('Array operations:');
    const numbers = [1, 2, 3, 4, 5];
    const squares = numbers.map((n) => n * n);
    const evens = numbers.filter((n) => n % 2 === 0);
    const sum = numbers.reduce((acc, n) => acc + n, 0);
    console.log(`  numbers = [${numbers.join(', ')}]`);
    console.log(`  squares = [${squares.join(', ')}]`);
    console.log(`  evens = [${evens.join(', ')}]`);
    console.log(`  sum = ${sum}`);
    console.log();

    // Async demo
    console.log('Async calculation:');
    calc.calculateAsync('sqrt', 256).then((result) => {
        console.log(`  async sqrt(256) = ${result.value}`);
    });
}

// Parse command line arguments
function parseArgs(): { help: boolean; version: boolean; demo: boolean; batch: boolean; expressions: string[] } {
    const args = argv.slice(2);
    const result = {
        help: false,
        version: false,
        demo: false,
        batch: false,
        expressions: [] as string[],
    };

    let i = 0;
    while (i < args.length) {
        const arg = args[i];
        switch (arg) {
            case '-h':
            case '--help':
                result.help = true;
                break;
            case '-v':
            case '--version':
                result.version = true;
                break;
            case '--demo':
                result.demo = true;
                break;
            case '--batch':
                result.batch = true;
                break;
            default:
                result.expressions.push(arg);
        }
        i++;
    }

    return result;
}

// Main entry point
async function main(): Promise<void> {
    const args = parseArgs();

    if (args.help) {
        console.log(handleHelp());
        return;
    }

    if (args.version) {
        console.log(`${APP_NAME} v${VERSION}`);
        return;
    }

    if (args.demo) {
        runDemo();
        return;
    }

    if (args.batch || args.expressions.length > 0) {
        await runBatch(args.expressions);
        return;
    }

    await runInteractive();
}

// Run
main().catch((error) => {
    console.error('Fatal error:', error);
    process.exit(1);
});
