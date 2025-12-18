#!/usr/bin/env python3
"""
Main entry point for the Python Calculator application.

Demonstrates various Python patterns including:
- Module imports (absolute and relative)
- Async/await
- Type hints
- Error handling
- List/dict comprehensions
- Generator expressions
"""

import sys
import asyncio
from typing import Optional
from pathlib import Path

# Absolute imports from package
from operations import add, subtract, multiply, divide, VERSION
from operations.advanced import AdvancedCalculator, ScientificCalculator, OperationResult
from operations.utils import (
    validate_number,
    format_result,
    parse_expression,
    CalculationError,
    log_call,
    memoize,
)

# Conditional import
try:
    import readline  # For better input handling
    HAS_READLINE = True
except ImportError:
    HAS_READLINE = False


# Global calculator instance
_calculator: Optional[AdvancedCalculator] = None


def get_calculator() -> AdvancedCalculator:
    """Get or create global calculator instance (lazy singleton pattern)."""
    global _calculator
    if _calculator is None:
        _calculator = ScientificCalculator(precision=10, memory_size=50)
    return _calculator


@memoize
def fibonacci_cached(n: int) -> int:
    """Cached Fibonacci using memoization decorator."""
    if n <= 1:
        return n
    return fibonacci_cached(n - 1) + fibonacci_cached(n - 2)


@log_call
def process_command(command: str) -> str:
    """
    Process a calculator command and return result string.

    Demonstrates complex control flow and pattern matching.
    """
    command = command.strip().lower()

    # Empty command
    if not command:
        return ""

    # Help command
    if command in ('help', '?', 'h'):
        return get_help_text()

    # Exit commands
    if command in ('exit', 'quit', 'q'):
        return "__EXIT__"

    # History command
    if command == 'history':
        calc = get_calculator()
        history = calc.get_history(10)
        if not history:
            return "No history"
        return '\n'.join(f"  {h.expression} = {h.result}" for h in history)

    # Memory commands using match
    match command.split():
        case ['mc'] | ['memory', 'clear']:
            get_calculator().memory_clear()
            return "Memory cleared"
        case ['mr'] | ['memory', 'recall']:
            return f"Memory: {get_calculator().memory_recall()}"
        case ['m+', value] | ['memory', 'add', value]:
            get_calculator().memory_add(float(value))
            return f"Added to memory: {get_calculator().memory}"
        case ['m-', value] | ['memory', 'sub', value]:
            get_calculator().memory_subtract(float(value))
            return f"Subtracted from memory: {get_calculator().memory}"
        case _:
            pass  # Continue to expression parsing

    # Try to parse as expression
    try:
        op, args = parse_expression(command)
        result = get_calculator().calculate(op, *args)

        if result.success:
            return format_result(result.value, precision=6)
        else:
            return f"Error: {result.error_message}"

    except CalculationError as e:
        return f"Calculation error: {e}"
    except Exception as e:
        return f"Error: {e}"


def get_help_text() -> str:
    """Return help text using multiline string."""
    return """
Calculator Commands:
  Basic:     2 + 3, 10 - 5, 4 * 3, 20 / 4
  Power:     2 ** 8, 3 ** 2
  Modulo:    10 % 3
  Functions: sqrt(16), log(100), sin(0.5), cos(0.5), tan(0.5)

  Memory:
    mc / memory clear  - Clear memory
    mr / memory recall - Show memory value
    m+ <value>        - Add to memory
    m- <value>        - Subtract from memory

  Other:
    history - Show calculation history
    help    - Show this help
    quit    - Exit calculator
""".strip()


def interactive_mode() -> None:
    """Run calculator in interactive mode."""
    print(f"Python Calculator v{VERSION}")
    print("Type 'help' for commands, 'quit' to exit")
    print()

    while True:
        try:
            # Get input
            user_input = input("calc> ")

            # Process command
            result = process_command(user_input)

            # Check for exit
            if result == "__EXIT__":
                print("Goodbye!")
                break

            # Print result
            if result:
                print(result)

        except KeyboardInterrupt:
            print("\nInterrupted. Type 'quit' to exit.")
        except EOFError:
            print("\nGoodbye!")
            break


async def async_calculate(expressions: list[str]) -> list[str]:
    """
    Process multiple expressions concurrently.

    Demonstrates async/await patterns.
    """
    async def process_one(expr: str) -> str:
        # Simulate async work
        await asyncio.sleep(0.01)
        return process_command(expr)

    # Create tasks for all expressions
    tasks = [process_one(expr) for expr in expressions]

    # Gather results
    results = await asyncio.gather(*tasks, return_exceptions=True)

    # Process results
    processed = []
    for expr, result in zip(expressions, results):
        if isinstance(result, Exception):
            processed.append(f"{expr} = Error: {result}")
        else:
            processed.append(f"{expr} = {result}")

    return processed


def batch_mode(expressions: list[str]) -> None:
    """Process multiple expressions in batch mode."""
    # Use list comprehension to filter valid expressions
    valid_exprs = [e for e in expressions if e.strip() and not e.startswith('#')]

    if not valid_exprs:
        print("No valid expressions to process")
        return

    # Run async processing
    results = asyncio.run(async_calculate(valid_exprs))

    # Print results
    for result in results:
        print(result)


def demo_comprehensions() -> None:
    """Demonstrate various comprehension patterns."""
    calc = get_calculator()

    # List comprehension: squares
    squares = [i ** 2 for i in range(1, 11)]
    print(f"Squares: {squares}")

    # Dict comprehension: number -> factorial
    factorials = {n: calc.factorial(n) for n in range(1, 8)}
    print(f"Factorials: {factorials}")

    # Set comprehension: unique last digits of squares
    last_digits = {(i ** 2) % 10 for i in range(100)}
    print(f"Possible last digits of squares: {sorted(last_digits)}")

    # Generator expression: sum of cubes
    cube_sum = sum(i ** 3 for i in range(1, 11))
    print(f"Sum of cubes 1-10: {cube_sum}")

    # Nested comprehension: multiplication table
    mult_table = [[i * j for j in range(1, 6)] for i in range(1, 6)]
    print("Multiplication table (5x5):")
    for row in mult_table:
        print(f"  {row}")

    # Conditional comprehension: even Fibonacci numbers
    even_fibs = [fibonacci_cached(n) for n in range(20) if fibonacci_cached(n) % 2 == 0]
    print(f"Even Fibonacci numbers: {even_fibs}")


def main() -> int:
    """Main entry point."""
    # Parse command line arguments manually (no argparse for simplicity)
    args = sys.argv[1:]

    # Handle flags
    if '--help' in args or '-h' in args:
        print(get_help_text())
        return 0

    if '--version' in args or '-v' in args:
        print(f"Python Calculator v{VERSION}")
        return 0

    if '--demo' in args:
        demo_comprehensions()
        return 0

    # Batch mode with expressions
    if '--batch' in args:
        idx = args.index('--batch')
        expressions = args[idx + 1:]
        batch_mode(expressions)
        return 0

    # Single expression evaluation
    if args:
        expr = ' '.join(args)
        result = process_command(expr)
        if result and result != "__EXIT__":
            print(result)
        return 0

    # Default: interactive mode
    interactive_mode()
    return 0


if __name__ == '__main__':
    sys.exit(main())
