"""Utility functions and classes for calculator operations."""

from typing import Any, Callable, TypeVar
from functools import wraps
import re
import logging

# External library imports
from decimal import Decimal, InvalidOperation

# Configure logger
logger = logging.getLogger(__name__)

T = TypeVar('T')
R = TypeVar('R')


class CalculationError(Exception):
    """Custom exception for calculation errors."""

    def __init__(self, message: str, code: int = 0):
        super().__init__(message)
        self.code = code
        self.message = message

    def __str__(self) -> str:
        return f"CalculationError({self.code}): {self.message}"


class ValidationError(CalculationError):
    """Exception for validation failures."""
    pass


def validate_number(value: Any) -> bool:
    """
    Validate that a value is a valid number.

    Handles various edge cases with nested conditionals.
    """
    if value is None:
        return False

    if isinstance(value, bool):
        # Booleans are technically ints in Python, but we don't want them
        return False

    if isinstance(value, (int, float)):
        # Check for special float values
        if isinstance(value, float):
            import math
            if math.isnan(value) or math.isinf(value):
                return False
        return True

    if isinstance(value, Decimal):
        try:
            # Check if it's a valid decimal
            float(value)
            return True
        except (InvalidOperation, ValueError):
            return False

    if isinstance(value, str):
        # Try to parse as number
        try:
            float(value)
            return True
        except ValueError:
            return False

    return False


def format_result(
    value: Any,
    precision: int = 2,
    thousands_separator: bool = True,
    prefix: str = "",
    suffix: str = ""
) -> str:
    """
    Format a number for display.

    Uses string formatting with various options.
    """
    if not validate_number(value):
        return "Invalid"

    num = float(value)

    # Format with precision
    if thousands_separator:
        formatted = f"{num:,.{precision}f}"
    else:
        formatted = f"{num:.{precision}f}"

    return f"{prefix}{formatted}{suffix}"


def parse_expression(expression: str) -> tuple[str, list[float]]:
    """
    Parse a simple mathematical expression.

    Returns tuple of (operator, operands).
    """
    # Match patterns like "2 + 3", "sqrt(4)", "2 * 3 + 4"
    expression = expression.strip()

    # Check for function-style calls
    func_match = re.match(r'(\w+)\s*\(\s*(.+)\s*\)', expression)
    if func_match:
        func_name = func_match.group(1)
        args_str = func_match.group(2)
        args = [float(a.strip()) for a in args_str.split(',')]
        return func_name, args

    # Check for binary operations
    binary_ops = ['+', '-', '*', '/', '**', '%']
    for op in binary_ops:
        if op in expression:
            parts = expression.split(op, 1)
            if len(parts) == 2:
                try:
                    a = float(parts[0].strip())
                    b = float(parts[1].strip())
                    return op, [a, b]
                except ValueError:
                    continue

    raise ValidationError(f"Cannot parse expression: {expression}")


# Decorator for logging function calls
def log_call(func: Callable[..., R]) -> Callable[..., R]:
    """Decorator to log function calls."""

    @wraps(func)
    def wrapper(*args, **kwargs) -> R:
        logger.debug(f"Calling {func.__name__} with args={args}, kwargs={kwargs}")
        try:
            result = func(*args, **kwargs)
            logger.debug(f"{func.__name__} returned {result}")
            return result
        except Exception as e:
            logger.error(f"{func.__name__} raised {type(e).__name__}: {e}")
            raise

    return wrapper


# Decorator for validating numeric arguments
def validate_args(func: Callable[..., R]) -> Callable[..., R]:
    """Decorator to validate all positional arguments are numbers."""

    @wraps(func)
    def wrapper(*args, **kwargs) -> R:
        for i, arg in enumerate(args):
            if not validate_number(arg):
                raise ValidationError(f"Argument {i} is not a valid number: {arg}")
        return func(*args, **kwargs)

    return wrapper


# Decorator factory with parameters
def retry(max_attempts: int = 3, exceptions: tuple = (Exception,)):
    """
    Decorator factory for retrying failed operations.

    Demonstrates closure and decorator with parameters.
    """
    def decorator(func: Callable[..., R]) -> Callable[..., R]:
        @wraps(func)
        def wrapper(*args, **kwargs) -> R:
            last_exception = None
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    logger.warning(
                        f"Attempt {attempt + 1}/{max_attempts} failed: {e}"
                    )
            raise last_exception

        return wrapper
    return decorator


# Memoization decorator using closure
def memoize(func: Callable[..., R]) -> Callable[..., R]:
    """Memoization decorator with closure-based cache."""
    cache: dict[tuple, R] = {}

    @wraps(func)
    def wrapper(*args) -> R:
        if args in cache:
            return cache[args]
        result = func(*args)
        cache[args] = result
        return result

    wrapper.cache = cache  # type: ignore
    wrapper.clear_cache = lambda: cache.clear()  # type: ignore
    return wrapper


class Singleton:
    """
    Singleton metaclass implementation.

    Ensures only one instance of a class exists.
    """
    _instances: dict[type, Any] = {}

    def __new__(cls, *args, **kwargs):
        if cls not in cls._instances:
            cls._instances[cls] = super().__new__(cls)
        return cls._instances[cls]


class Observable:
    """
    Observable pattern implementation.

    Demonstrates callback registration and invocation.
    """

    def __init__(self):
        self._observers: list[Callable[[Any], None]] = []

    def subscribe(self, callback: Callable[[Any], None]) -> Callable[[], None]:
        """Subscribe to updates. Returns unsubscribe function."""
        self._observers.append(callback)

        def unsubscribe():
            self._observers.remove(callback)

        return unsubscribe

    def notify(self, data: Any) -> None:
        """Notify all observers."""
        for observer in self._observers:
            try:
                observer(data)
            except Exception as e:
                logger.error(f"Observer error: {e}")


# Context manager using generator
from contextlib import contextmanager

@contextmanager
def calculation_context(name: str):
    """Context manager for tracking calculation context."""
    logger.info(f"Starting calculation: {name}")
    try:
        yield
        logger.info(f"Completed calculation: {name}")
    except Exception as e:
        logger.error(f"Calculation {name} failed: {e}")
        raise


# Async utility (for async calculator operations)
async def async_validate(value: Any) -> bool:
    """Async version of validate_number."""
    import asyncio
    await asyncio.sleep(0)  # Yield control
    return validate_number(value)
