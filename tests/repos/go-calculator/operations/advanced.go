package operations

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// OperationType represents the type of operation.
type OperationType int

const (
	// OpUnary represents a unary operation.
	OpUnary OperationType = iota
	// OpBinary represents a binary operation.
	OpBinary
	// OpReduction represents a reduction operation.
	OpReduction
)

// CalculationError represents an error during calculation.
type CalculationError struct {
	Operation string
	Message   string
	Err       error
}

func (e *CalculationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

func (e *CalculationError) Unwrap() error {
	return e.Err
}

// OperationResult holds the result of a calculation.
type OperationResult struct {
	Value     float64
	Operation string
	Inputs    []float64
	Success   bool
	Error     error
	Duration  time.Duration
}

// HistoryEntry represents an entry in calculation history.
type HistoryEntry struct {
	Expression string
	Result     float64
	Timestamp  time.Time
}

// Calculator defines the interface for calculators.
type Calculator interface {
	Calculate(op string, args ...float64) (*OperationResult, error)
	Reset()
	GetHistory() []HistoryEntry
}

// AdvancedCalculator provides advanced calculation capabilities.
type AdvancedCalculator struct {
	precision    int
	memory       float64
	history      []HistoryEntry
	historyLimit int
	mu           sync.RWMutex
}

// NewAdvancedCalculator creates a new AdvancedCalculator instance.
func NewAdvancedCalculator(precision, historyLimit int) *AdvancedCalculator {
	return &AdvancedCalculator{
		precision:    precision,
		memory:       0,
		history:      make([]HistoryEntry, 0, historyLimit),
		historyLimit: historyLimit,
	}
}

// Calculate performs the specified operation.
func (c *AdvancedCalculator) Calculate(op string, args ...float64) (*OperationResult, error) {
	start := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	result := &OperationResult{
		Operation: op,
		Inputs:    args,
	}

	var value float64
	var err error

	// Binary operations
	switch op {
	case "+":
		if len(args) != 2 {
			err = errors.New("addition requires 2 arguments")
		} else {
			value = Add(args[0], args[1])
		}
	case "-":
		if len(args) != 2 {
			err = errors.New("subtraction requires 2 arguments")
		} else {
			value = Subtract(args[0], args[1])
		}
	case "*":
		if len(args) != 2 {
			err = errors.New("multiplication requires 2 arguments")
		} else {
			value = Multiply(args[0], args[1])
		}
	case "/":
		if len(args) != 2 {
			err = errors.New("division requires 2 arguments")
		} else {
			value, err = Divide(args[0], args[1])
		}
	case "**", "pow":
		if len(args) != 2 {
			err = errors.New("power requires 2 arguments")
		} else {
			value = Power(args[0], args[1])
		}
	case "%", "mod":
		if len(args) != 2 {
			err = errors.New("modulo requires 2 arguments")
		} else {
			intResult, modErr := Modulo(int(args[0]), int(args[1]))
			if modErr != nil {
				err = modErr
			} else {
				value = float64(intResult)
			}
		}

	// Unary operations
	case "sqrt":
		if len(args) != 1 {
			err = errors.New("sqrt requires 1 argument")
		} else if args[0] < 0 {
			err = &CalculationError{
				Operation: "sqrt",
				Message:   "cannot take square root of negative number",
			}
		} else {
			value = math.Sqrt(args[0])
		}
	case "log":
		if len(args) < 1 || len(args) > 2 {
			err = errors.New("log requires 1 or 2 arguments")
		} else if args[0] <= 0 {
			err = &CalculationError{
				Operation: "log",
				Message:   "cannot take log of non-positive number",
			}
		} else {
			if len(args) == 2 {
				value = math.Log(args[0]) / math.Log(args[1])
			} else {
				value = math.Log(args[0])
			}
		}
	case "sin":
		if len(args) != 1 {
			err = errors.New("sin requires 1 argument")
		} else {
			value = math.Sin(args[0])
		}
	case "cos":
		if len(args) != 1 {
			err = errors.New("cos requires 1 argument")
		} else {
			value = math.Cos(args[0])
		}
	case "tan":
		if len(args) != 1 {
			err = errors.New("tan requires 1 argument")
		} else {
			value = math.Tan(args[0])
		}
	case "abs":
		if len(args) != 1 {
			err = errors.New("abs requires 1 argument")
		} else {
			value = math.Abs(args[0])
		}
	case "floor":
		if len(args) != 1 {
			err = errors.New("floor requires 1 argument")
		} else {
			value = math.Floor(args[0])
		}
	case "ceil":
		if len(args) != 1 {
			err = errors.New("ceil requires 1 argument")
		} else {
			value = math.Ceil(args[0])
		}

	// Reduction operations
	case "sum":
		value = Sum(args...)
	case "product":
		value = Product(args...)
	case "max":
		if len(args) == 0 {
			err = errors.New("max requires at least 1 argument")
		} else {
			value = args[0]
			for _, a := range args[1:] {
				if a > value {
					value = a
				}
			}
		}
	case "min":
		if len(args) == 0 {
			err = errors.New("min requires at least 1 argument")
		} else {
			value = args[0]
			for _, a := range args[1:] {
				if a < value {
					value = a
				}
			}
		}
	case "avg", "mean":
		if len(args) == 0 {
			err = errors.New("average requires at least 1 argument")
		} else {
			value = Sum(args...) / float64(len(args))
		}

	default:
		err = &CalculationError{
			Operation: op,
			Message:   "unknown operation",
		}
	}

	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err
		return result, err
	}

	// Round to precision
	multiplier := math.Pow(10, float64(c.precision))
	value = math.Round(value*multiplier) / multiplier

	result.Value = value
	result.Success = true

	// Add to history
	c.addHistory(op, args, value)

	return result, nil
}

// addHistory adds an entry to the calculation history.
func (c *AdvancedCalculator) addHistory(op string, args []float64, result float64) {
	expr := fmt.Sprintf("%s(%v)", op, args)
	entry := HistoryEntry{
		Expression: expr,
		Result:     result,
		Timestamp:  time.Now(),
	}

	if len(c.history) >= c.historyLimit {
		c.history = c.history[1:]
	}
	c.history = append(c.history, entry)
}

// Reset clears the calculator state.
func (c *AdvancedCalculator) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memory = 0
	c.history = c.history[:0]
}

// GetHistory returns the calculation history.
func (c *AdvancedCalculator) GetHistory() []HistoryEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]HistoryEntry, len(c.history))
	copy(result, c.history)
	return result
}

// MemoryAdd adds a value to memory.
func (c *AdvancedCalculator) MemoryAdd(value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memory += value
}

// MemorySubtract subtracts a value from memory.
func (c *AdvancedCalculator) MemorySubtract(value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memory -= value
}

// MemoryClear clears the memory.
func (c *AdvancedCalculator) MemoryClear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memory = 0
}

// MemoryRecall returns the memory value.
func (c *AdvancedCalculator) MemoryRecall() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.memory
}

// ScientificCalculator extends AdvancedCalculator with scientific functions.
type ScientificCalculator struct {
	*AdvancedCalculator
	angleMode string // "radians" or "degrees"
}

// NewScientificCalculator creates a new ScientificCalculator.
func NewScientificCalculator(precision, historyLimit int) *ScientificCalculator {
	return &ScientificCalculator{
		AdvancedCalculator: NewAdvancedCalculator(precision, historyLimit),
		angleMode:          "radians",
	}
}

// SetAngleMode sets the angle mode for trigonometric functions.
func (c *ScientificCalculator) SetAngleMode(mode string) error {
	if mode != "radians" && mode != "degrees" {
		return errors.New("angle mode must be 'radians' or 'degrees'")
	}
	c.angleMode = mode
	return nil
}

// ToRadians converts an angle to radians if in degrees mode.
func (c *ScientificCalculator) ToRadians(angle float64) float64 {
	if c.angleMode == "degrees" {
		return angle * math.Pi / 180
	}
	return angle
}

// Factorial calculates n! iteratively.
func (c *ScientificCalculator) Factorial(n int) (int64, error) {
	if n < 0 {
		return 0, &CalculationError{
			Operation: "factorial",
			Message:   "negative input not allowed",
		}
	}
	if n > 20 {
		return 0, &CalculationError{
			Operation: "factorial",
			Message:   "input too large (max 20)",
		}
	}

	var result int64 = 1
	for i := 2; i <= n; i++ {
		result *= int64(i)
	}
	return result, nil
}

// Fibonacci calculates the nth Fibonacci number.
func (c *ScientificCalculator) Fibonacci(n int) (int64, error) {
	if n < 0 {
		return 0, &CalculationError{
			Operation: "fibonacci",
			Message:   "negative index not allowed",
		}
	}

	if n <= 1 {
		return int64(n), nil
	}

	var a, b int64 = 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b, nil
}

// IsPrime checks if a number is prime.
func (c *ScientificCalculator) IsPrime(n int) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}

	sqrt := int(math.Sqrt(float64(n)))
	for i := 3; i <= sqrt; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// PrimeFactors returns the prime factors of n.
func (c *ScientificCalculator) PrimeFactors(n int) []int {
	var factors []int
	d := 2
	for d*d <= n {
		for n%d == 0 {
			factors = append(factors, d)
			n /= d
		}
		d++
	}
	if n > 1 {
		factors = append(factors, n)
	}
	return factors
}

// GCD calculates the greatest common divisor using Euclidean algorithm.
func (c *ScientificCalculator) GCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return Abs(a)
}

// LCM calculates the least common multiple.
func (c *ScientificCalculator) LCM(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return Abs(a*b) / c.GCD(a, b)
}

// CalculateWithContext performs calculation with context support for cancellation.
func (c *ScientificCalculator) CalculateWithContext(
	ctx context.Context,
	op string,
	args ...float64,
) (*OperationResult, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Perform calculation
	result, err := c.Calculate(op, args...)

	// Check again after calculation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return result, err
}

// BatchCalculate performs multiple calculations concurrently.
func (c *ScientificCalculator) BatchCalculate(
	ctx context.Context,
	operations []struct {
		Op   string
		Args []float64
	},
) []*OperationResult {
	results := make([]*OperationResult, len(operations))
	var wg sync.WaitGroup

	for i, op := range operations {
		wg.Add(1)
		go func(idx int, operation struct {
			Op   string
			Args []float64
		}) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				results[idx] = &OperationResult{
					Operation: operation.Op,
					Success:   false,
					Error:     ctx.Err(),
				}
				return
			default:
			}

			result, _ := c.Calculate(operation.Op, operation.Args...)
			results[idx] = result
		}(i, op)
	}

	wg.Wait()
	return results
}
