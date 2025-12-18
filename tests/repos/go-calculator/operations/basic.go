// Package operations provides basic and advanced calculator operations.
package operations

import (
	"errors"
	"math"

	"github.com/shopspring/decimal"
)

// ErrDivisionByZero is returned when attempting to divide by zero.
var ErrDivisionByZero = errors.New("division by zero")

// ErrInvalidOperation is returned for unknown operations.
var ErrInvalidOperation = errors.New("invalid operation")

// ErrNegativeInput is returned when a negative input is not allowed.
var ErrNegativeInput = errors.New("negative input not allowed")

// Number represents a numeric type constraint for generics.
type Number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

// Add returns the sum of two numbers.
func Add[T Number](a, b T) T {
	return a + b
}

// Subtract returns the difference of two numbers.
func Subtract[T Number](a, b T) T {
	return a - b
}

// Multiply returns the product of two numbers.
func Multiply[T Number](a, b T) T {
	return a * b
}

// Divide returns the quotient of two numbers.
// Returns an error if b is zero.
func Divide[T Number](a, b T) (T, error) {
	var zero T
	if b == zero {
		return zero, ErrDivisionByZero
	}
	return a / b, nil
}

// Sum returns the sum of all provided numbers using variadic parameters.
func Sum[T Number](numbers ...T) T {
	var result T
	for _, n := range numbers {
		result += n
	}
	return result
}

// Product returns the product of all provided numbers.
func Product[T Number](numbers ...T) T {
	if len(numbers) == 0 {
		var zero T
		return zero
	}
	result := numbers[0]
	for i := 1; i < len(numbers); i++ {
		result *= numbers[i]
	}
	return result
}

// OperationFunc is a function type for binary operations.
type OperationFunc[T Number] func(a, b T) T

// CreateOperation returns an operation function based on the operator string.
// Demonstrates function as return type.
func CreateOperation[T Number](operator string) (OperationFunc[T], error) {
	switch operator {
	case "+":
		return func(a, b T) T { return a + b }, nil
	case "-":
		return func(a, b T) T { return a - b }, nil
	case "*":
		return func(a, b T) T { return a * b }, nil
	case "/":
		return func(a, b T) T { return a / b }, nil
	default:
		return nil, ErrInvalidOperation
	}
}

// BatchOperation applies an operation to a slice of number pairs.
// Returns results and any errors encountered.
func BatchOperation[T Number](
	pairs [][2]T,
	op OperationFunc[T],
) []T {
	results := make([]T, 0, len(pairs))
	for _, pair := range pairs {
		results = append(results, op(pair[0], pair[1]))
	}
	return results
}

// DecimalAdd performs precise decimal addition.
func DecimalAdd(a, b string) (string, error) {
	da, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}
	db, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}
	return da.Add(db).String(), nil
}

// Power calculates a raised to the power of b.
func Power(a, b float64) float64 {
	return math.Pow(a, b)
}

// Modulo returns the remainder of a divided by b.
func Modulo(a, b int) (int, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	return a % b, nil
}

// Abs returns the absolute value of a number.
func Abs[T Number](n T) T {
	if n < 0 {
		return -n
	}
	return n
}

// Max returns the maximum of two numbers.
func Max[T Number](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min returns the minimum of two numbers.
func Min[T Number](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Clamp restricts a value to be within [min, max].
func Clamp[T Number](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// MapSlice applies a function to each element of a slice.
// Demonstrates higher-order functions with generics.
func MapSlice[T, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// FilterSlice returns elements that satisfy the predicate.
func FilterSlice[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// ReduceSlice reduces a slice to a single value.
func ReduceSlice[T, R any](slice []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, v := range slice {
		result = reducer(result, v)
	}
	return result
}
