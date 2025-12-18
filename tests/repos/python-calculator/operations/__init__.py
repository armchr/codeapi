# Package initialization with re-exports
from .basic import add, subtract, multiply, divide
from .advanced import AdvancedCalculator
from .utils import validate_number, format_result

__all__ = [
    'add', 'subtract', 'multiply', 'divide',
    'AdvancedCalculator',
    'validate_number', 'format_result'
]

# Package-level constant
VERSION = "1.0.0"
