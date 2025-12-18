package operations

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// ValidateNumber checks if a value is a valid finite number.
func ValidateNumber(value float64) error {
	if value != value { // NaN check
		return &ValidationError{Field: "value", Message: "is NaN"}
	}
	if value > 1e308 || value < -1e308 {
		return &ValidationError{Field: "value", Message: "is infinite"}
	}
	return nil
}

// FormatResult formats a number for display.
func FormatResult(value float64, precision int, thousandsSep bool) string {
	formatted := strconv.FormatFloat(value, 'f', precision, 64)

	if !thousandsSep {
		return formatted
	}

	// Add thousands separator
	parts := strings.Split(formatted, ".")
	intPart := parts[0]

	var result strings.Builder
	negative := false
	if intPart[0] == '-' {
		negative = true
		intPart = intPart[1:]
	}

	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}

	if negative {
		return "-" + result.String() + "." + parts[1]
	}
	if len(parts) > 1 {
		return result.String() + "." + parts[1]
	}
	return result.String()
}

// ParsedExpression represents a parsed mathematical expression.
type ParsedExpression struct {
	Operator string
	Operands []float64
}

// ParseExpression parses a simple mathematical expression.
func ParseExpression(expr string) (*ParsedExpression, error) {
	expr = strings.TrimSpace(expr)

	// Try function-style: func(args)
	funcRegex := regexp.MustCompile(`^(\w+)\s*\(\s*(.+)\s*\)$`)
	if matches := funcRegex.FindStringSubmatch(expr); matches != nil {
		funcName := matches[1]
		argsStr := matches[2]

		args, err := parseArgs(argsStr)
		if err != nil {
			return nil, err
		}

		return &ParsedExpression{
			Operator: funcName,
			Operands: args,
		}, nil
	}

	// Try binary operation: a op b
	operators := []string{"**", "+", "-", "*", "/", "%"}
	for _, op := range operators {
		if idx := strings.Index(expr, op); idx > 0 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(op):])

			a, err := strconv.ParseFloat(left, 64)
			if err != nil {
				continue
			}
			b, err := strconv.ParseFloat(right, 64)
			if err != nil {
				continue
			}

			return &ParsedExpression{
				Operator: op,
				Operands: []float64{a, b},
			}, nil
		}
	}

	return nil, &ValidationError{
		Field:   "expression",
		Message: "cannot parse expression: " + expr,
	}
}

// parseArgs parses comma-separated arguments.
func parseArgs(argsStr string) ([]float64, error) {
	parts := strings.Split(argsStr, ",")
	args := make([]float64, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		val, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, &ValidationError{
				Field:   "argument",
				Message: "invalid number: " + part,
			}
		}
		args = append(args, val)
	}

	return args, nil
}

// Option represents a functional option for configuration.
type Option func(*Config)

// Config holds calculator configuration.
type Config struct {
	Precision    int
	HistoryLimit int
	AngleMode    string
	EnableCache  bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Precision:    10,
		HistoryLimit: 100,
		AngleMode:    "radians",
		EnableCache:  true,
	}
}

// WithPrecision sets the precision option.
func WithPrecision(p int) Option {
	return func(c *Config) {
		c.Precision = p
	}
}

// WithHistoryLimit sets the history limit option.
func WithHistoryLimit(limit int) Option {
	return func(c *Config) {
		c.HistoryLimit = limit
	}
}

// WithAngleMode sets the angle mode option.
func WithAngleMode(mode string) Option {
	return func(c *Config) {
		c.AngleMode = mode
	}
}

// WithCache enables or disables caching.
func WithCache(enabled bool) Option {
	return func(c *Config) {
		c.EnableCache = enabled
	}
}

// ApplyOptions applies functional options to a config.
func ApplyOptions(opts ...Option) *Config {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// Result represents a generic result type (similar to Rust's Result).
type Result[T any] struct {
	value T
	err   error
}

// Ok creates a successful result.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Err creates an error result.
func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{value: zero, err: err}
}

// IsOk returns true if the result is successful.
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the result is an error.
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Unwrap returns the value or panics if error.
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// UnwrapOr returns the value or a default.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.err != nil {
		return defaultValue
	}
	return r.value
}

// UnwrapOrElse returns the value or calls a function to get default.
func (r Result[T]) UnwrapOrElse(fn func(error) T) T {
	if r.err != nil {
		return fn(r.err)
	}
	return r.value
}

// Map transforms the value if successful.
func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(fn(r.value))
}

// FlatMap chains result operations.
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return fn(r.value)
}

// Observable provides a simple observer pattern implementation.
type Observable[T any] struct {
	observers []func(T)
}

// NewObservable creates a new Observable.
func NewObservable[T any]() *Observable[T] {
	return &Observable[T]{
		observers: make([]func(T), 0),
	}
}

// Subscribe adds an observer and returns an unsubscribe function.
func (o *Observable[T]) Subscribe(fn func(T)) func() {
	o.observers = append(o.observers, fn)
	idx := len(o.observers) - 1

	return func() {
		// Remove observer
		o.observers = append(o.observers[:idx], o.observers[idx+1:]...)
	}
}

// Notify calls all observers with the given value.
func (o *Observable[T]) Notify(value T) {
	for _, fn := range o.observers {
		fn(value)
	}
}

// Cache provides a simple memoization cache.
type Cache[K comparable, V any] struct {
	data map[K]V
}

// NewCache creates a new cache.
func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		data: make(map[K]V),
	}
}

// Get retrieves a value from the cache.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	val, ok := c.data[key]
	return val, ok
}

// Set stores a value in the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.data[key] = value
}

// GetOrCompute gets a cached value or computes and caches it.
func (c *Cache[K, V]) GetOrCompute(key K, compute func() V) V {
	if val, ok := c.data[key]; ok {
		return val
	}
	val := compute()
	c.data[key] = val
	return val
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	c.data = make(map[K]V)
}
