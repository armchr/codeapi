"""Basic arithmetic operations module."""

from typing import Union, Optional
from functools import reduce
from decimal import Decimal, ROUND_HALF_UP
import math

# Type alias
Number = Union[int, float, Decimal]


def add(a: Number, b: Number) -> Number:
    """Add two numbers."""
    return a + b


def subtract(a: Number, b: Number) -> Number:
    """Subtract b from a."""
    return a - b


def multiply(a: Number, b: Number) -> Number:
    """Multiply two numbers."""
    return a * b


def divide(a: Number, b: Number) -> Optional[Number]:
    """
    Divide a by b.

    Returns None if division by zero is attempted.
    """
    if b == 0:
        return None
    return a / b


def sum_all(*args: Number) -> Number:
    """Sum any number of arguments using reduce."""
    return reduce(lambda x, y: x + y, args, 0)


def product_all(*args: Number) -> Number:
    """Multiply any number of arguments."""
    return reduce(multiply, args, 1)


def safe_divide(a: Number, b: Number, default: Number = 0) -> Number:
    """Safe division with default value on error."""
    result = divide(a, b)
    return default if result is None else result


def round_result(value: Number, precision: int = 2) -> Decimal:
    """Round a number to specified decimal places."""
    decimal_value = Decimal(str(value))
    quantize_str = '0.' + '0' * precision
    return decimal_value.quantize(Decimal(quantize_str), rounding=ROUND_HALF_UP)


def batch_operation(
    operation: str,
    numbers: list[Number]
) -> Optional[Number]:
    """
    Apply an operation to a list of numbers.

    Uses match statement (Python 3.10+) for operation dispatch.
    """
    if not numbers:
        return None

    match operation:
        case 'add' | 'sum':
            return sum_all(*numbers)
        case 'multiply' | 'product':
            return product_all(*numbers)
        case 'max':
            return max(numbers)
        case 'min':
            return min(numbers)
        case 'average' | 'avg':
            return sum_all(*numbers) / len(numbers)
        case _:
            return None


# Lambda functions for single operations
negate = lambda x: -x
absolute = lambda x: abs(x)
square = lambda x: x ** 2
sqrt = lambda x: math.sqrt(x) if x >= 0 else None


# Higher-order function
def create_operation(operator: str):
    """Factory function returning operation lambdas."""
    operations = {
        '+': lambda a, b: a + b,
        '-': lambda a, b: a - b,
        '*': lambda a, b: a * b,
        '/': lambda a, b: a / b if b != 0 else None,
        '**': lambda a, b: a ** b,
        '%': lambda a, b: a % b if b != 0 else None,
    }
    return operations.get(operator, lambda a, b: None)
