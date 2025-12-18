"""Advanced calculator operations with class-based design."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Callable, Any, Generic, TypeVar
from enum import Enum, auto
import math
import statistics
from collections import deque

# Relative imports
from .basic import add, subtract, multiply, divide, Number
from .utils import validate_number, CalculationError


T = TypeVar('T')


class OperationType(Enum):
    """Enum for operation types."""
    UNARY = auto()
    BINARY = auto()
    REDUCTION = auto()


@dataclass
class OperationResult:
    """Dataclass to hold operation results with metadata."""
    value: Number
    operation: str
    inputs: list[Number]
    success: bool = True
    error_message: str = ""


@dataclass
class HistoryEntry:
    """Entry in calculation history."""
    expression: str
    result: Number
    timestamp: float = field(default_factory=lambda: __import__('time').time())


class BaseCalculator(ABC):
    """Abstract base class for calculators."""

    @abstractmethod
    def calculate(self, *args, **kwargs) -> Number:
        """Perform calculation."""
        pass

    @abstractmethod
    def reset(self) -> None:
        """Reset calculator state."""
        pass


class AdvancedCalculator(BaseCalculator):
    """
    Advanced calculator with history, memory, and extended operations.

    Demonstrates:
    - Inheritance from ABC
    - Property decorators
    - Class and instance methods
    - Static methods
    - Context manager protocol
    - Iterator protocol
    """

    # Class variable
    instances_created: int = 0

    def __init__(self, precision: int = 10, memory_size: int = 100):
        self._precision = precision
        self._memory: float = 0.0
        self._history: deque[HistoryEntry] = deque(maxlen=memory_size)
        self._last_result: Number = 0
        AdvancedCalculator.instances_created += 1

    @property
    def precision(self) -> int:
        """Get current precision."""
        return self._precision

    @precision.setter
    def precision(self, value: int) -> None:
        """Set precision with validation."""
        if not isinstance(value, int) or value < 0:
            raise ValueError("Precision must be a non-negative integer")
        self._precision = value

    @property
    def memory(self) -> float:
        """Get memory value."""
        return self._memory

    @property
    def last_result(self) -> Number:
        """Get last calculation result."""
        return self._last_result

    @classmethod
    def get_instance_count(cls) -> int:
        """Return number of instances created."""
        return cls.instances_created

    @staticmethod
    def is_valid_operation(op: str) -> bool:
        """Check if operation string is valid."""
        valid_ops = {'+', '-', '*', '/', '**', '%', 'sqrt', 'log', 'sin', 'cos', 'tan'}
        return op in valid_ops

    def calculate(self, operation: str, *args: Number) -> OperationResult:
        """
        Perform calculation based on operation string.

        Uses nested conditionals and various control flow patterns.
        """
        try:
            # Validate inputs
            for arg in args:
                if not validate_number(arg):
                    return OperationResult(
                        value=0, operation=operation, inputs=list(args),
                        success=False, error_message="Invalid input"
                    )

            result: Number

            # Binary operations
            if operation in ('+', '-', '*', '/') and len(args) == 2:
                a, b = args
                if operation == '+':
                    result = add(a, b)
                elif operation == '-':
                    result = subtract(a, b)
                elif operation == '*':
                    result = multiply(a, b)
                else:  # division
                    if b == 0:
                        raise CalculationError("Division by zero")
                    result = divide(a, b)

            # Power and modulo
            elif operation == '**' and len(args) == 2:
                result = args[0] ** args[1]
            elif operation == '%' and len(args) == 2:
                if args[1] == 0:
                    raise CalculationError("Modulo by zero")
                result = args[0] % args[1]

            # Unary operations
            elif operation == 'sqrt' and len(args) == 1:
                if args[0] < 0:
                    raise CalculationError("Cannot take sqrt of negative number")
                result = math.sqrt(args[0])
            elif operation == 'log' and len(args) >= 1:
                if args[0] <= 0:
                    raise CalculationError("Log of non-positive number")
                base = args[1] if len(args) > 1 else math.e
                result = math.log(args[0], base)
            elif operation in ('sin', 'cos', 'tan') and len(args) == 1:
                trig_funcs = {'sin': math.sin, 'cos': math.cos, 'tan': math.tan}
                result = trig_funcs[operation](args[0])

            else:
                raise CalculationError(f"Unknown operation: {operation}")

            # Round result
            result = round(result, self._precision)
            self._last_result = result

            # Record in history
            expr = f"{operation}({', '.join(str(a) for a in args)})"
            self._history.append(HistoryEntry(expression=expr, result=result))

            return OperationResult(
                value=result, operation=operation, inputs=list(args)
            )

        except CalculationError as e:
            return OperationResult(
                value=0, operation=operation, inputs=list(args),
                success=False, error_message=str(e)
            )
        except Exception as e:
            return OperationResult(
                value=0, operation=operation, inputs=list(args),
                success=False, error_message=f"Unexpected error: {e}"
            )

    def reset(self) -> None:
        """Reset calculator state."""
        self._memory = 0.0
        self._last_result = 0
        self._history.clear()

    def memory_add(self, value: Number) -> None:
        """Add value to memory."""
        self._memory += value

    def memory_subtract(self, value: Number) -> None:
        """Subtract value from memory."""
        self._memory -= value

    def memory_clear(self) -> None:
        """Clear memory."""
        self._memory = 0.0

    def memory_recall(self) -> float:
        """Recall memory value."""
        return self._memory

    def get_history(self, limit: int = 10) -> list[HistoryEntry]:
        """Get recent history entries."""
        return list(self._history)[-limit:]

    def statistics_operation(self, operation: str, data: list[Number]) -> Number:
        """Perform statistical operations on data."""
        if not data:
            raise CalculationError("Empty data list")

        stat_funcs: dict[str, Callable[[list], Number]] = {
            'mean': statistics.mean,
            'median': statistics.median,
            'mode': lambda d: statistics.mode(d),
            'stdev': statistics.stdev,
            'variance': statistics.variance,
        }

        if operation not in stat_funcs:
            raise CalculationError(f"Unknown statistical operation: {operation}")

        return stat_funcs[operation](data)

    # Context manager protocol
    def __enter__(self) -> 'AdvancedCalculator':
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> bool:
        self.reset()
        return False

    # Iterator protocol for history
    def __iter__(self):
        return iter(self._history)

    def __len__(self) -> int:
        return len(self._history)


class ScientificCalculator(AdvancedCalculator):
    """Extended calculator with scientific functions."""

    def __init__(self, angle_mode: str = 'radians', **kwargs):
        super().__init__(**kwargs)
        self._angle_mode = angle_mode

    @property
    def angle_mode(self) -> str:
        return self._angle_mode

    @angle_mode.setter
    def angle_mode(self, mode: str) -> None:
        if mode not in ('radians', 'degrees'):
            raise ValueError("Angle mode must be 'radians' or 'degrees'")
        self._angle_mode = mode

    def _to_radians(self, angle: Number) -> float:
        """Convert angle to radians if in degrees mode."""
        if self._angle_mode == 'degrees':
            return math.radians(angle)
        return float(angle)

    def factorial(self, n: int) -> int:
        """Calculate factorial using recursion."""
        if n < 0:
            raise CalculationError("Factorial of negative number")
        if n <= 1:
            return 1
        return n * self.factorial(n - 1)

    def fibonacci(self, n: int) -> int:
        """Calculate nth Fibonacci number using iteration."""
        if n < 0:
            raise CalculationError("Negative Fibonacci index")

        a, b = 0, 1
        for _ in range(n):
            a, b = b, a + b
        return a

    def is_prime(self, n: int) -> bool:
        """Check if number is prime using optimized trial division."""
        if n < 2:
            return False
        if n == 2:
            return True
        if n % 2 == 0:
            return False

        for i in range(3, int(math.sqrt(n)) + 1, 2):
            if n % i == 0:
                return False
        return True

    def prime_factors(self, n: int) -> list[int]:
        """Get prime factors of a number."""
        factors = []
        d = 2
        while d * d <= n:
            while n % d == 0:
                factors.append(d)
                n //= d
            d += 1
        if n > 1:
            factors.append(n)
        return factors


# Generic wrapper class
class ResultWrapper(Generic[T]):
    """Generic wrapper for calculation results."""

    def __init__(self, value: T, metadata: dict[str, Any] | None = None):
        self._value = value
        self._metadata = metadata or {}

    @property
    def value(self) -> T:
        return self._value

    @property
    def metadata(self) -> dict[str, Any]:
        return self._metadata

    def map(self, func: Callable[[T], T]) -> 'ResultWrapper[T]':
        """Apply function to wrapped value."""
        return ResultWrapper(func(self._value), self._metadata)

    def __repr__(self) -> str:
        return f"ResultWrapper({self._value!r})"
