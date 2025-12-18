// Package main provides the entry point for the Go calculator application.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// Internal package import
	"github.com/example/calculator/operations"

	// External library import
	"golang.org/x/exp/constraints"
)

// Version information
const (
	Version = "1.0.0"
	AppName = "Go Calculator"
)

// Numeric is a constraint for numeric types.
type Numeric interface {
	constraints.Integer | constraints.Float
}

// Global calculator instance
var calculator *operations.ScientificCalculator

// init initializes the calculator.
func init() {
	calculator = operations.NewScientificCalculator(10, 100)
}

// Command represents a calculator command.
type Command struct {
	Name        string
	Description string
	Handler     func(args []string) (string, error)
}

// commands holds all available commands.
var commands = map[string]Command{
	"help": {
		Name:        "help",
		Description: "Show help information",
		Handler:     handleHelp,
	},
	"history": {
		Name:        "history",
		Description: "Show calculation history",
		Handler:     handleHistory,
	},
	"mc": {
		Name:        "mc",
		Description: "Clear memory",
		Handler:     handleMemoryClear,
	},
	"mr": {
		Name:        "mr",
		Description: "Recall memory",
		Handler:     handleMemoryRecall,
	},
	"m+": {
		Name:        "m+",
		Description: "Add to memory",
		Handler:     handleMemoryAdd,
	},
	"m-": {
		Name:        "m-",
		Description: "Subtract from memory",
		Handler:     handleMemorySubtract,
	},
	"prime": {
		Name:        "prime",
		Description: "Check if a number is prime",
		Handler:     handlePrime,
	},
	"factors": {
		Name:        "factors",
		Description: "Get prime factors",
		Handler:     handleFactors,
	},
}

func handleHelp(args []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Go Calculator Commands:\n")
	sb.WriteString("  Basic:     2 + 3, 10 - 5, 4 * 3, 20 / 4\n")
	sb.WriteString("  Power:     2 ** 8, pow(2, 8)\n")
	sb.WriteString("  Functions: sqrt(16), log(100), sin(0.5), cos(0.5), tan(0.5)\n")
	sb.WriteString("  Stats:     sum(1,2,3), avg(1,2,3), max(1,2,3), min(1,2,3)\n")
	sb.WriteString("\nMemory:\n")
	sb.WriteString("  mc - Clear memory\n")
	sb.WriteString("  mr - Recall memory\n")
	sb.WriteString("  m+ <value> - Add to memory\n")
	sb.WriteString("  m- <value> - Subtract from memory\n")
	sb.WriteString("\nOther:\n")
	sb.WriteString("  history - Show calculation history\n")
	sb.WriteString("  prime <n> - Check if n is prime\n")
	sb.WriteString("  factors <n> - Get prime factors of n\n")
	sb.WriteString("  help - Show this help\n")
	sb.WriteString("  quit - Exit calculator\n")
	return sb.String(), nil
}

func handleHistory(args []string) (string, error) {
	history := calculator.GetHistory()
	if len(history) == 0 {
		return "No history", nil
	}

	var sb strings.Builder
	for _, entry := range history {
		sb.WriteString(fmt.Sprintf("  %s = %v\n", entry.Expression, entry.Result))
	}
	return sb.String(), nil
}

func handleMemoryClear(args []string) (string, error) {
	calculator.MemoryClear()
	return "Memory cleared", nil
}

func handleMemoryRecall(args []string) (string, error) {
	return fmt.Sprintf("Memory: %v", calculator.MemoryRecall()), nil
}

func handleMemoryAdd(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("m+ requires a value")
	}
	expr, err := operations.ParseExpression(args[0])
	if err != nil {
		return "", err
	}
	if len(expr.Operands) < 1 {
		return "", fmt.Errorf("invalid value")
	}
	calculator.MemoryAdd(expr.Operands[0])
	return fmt.Sprintf("Added to memory: %v", calculator.MemoryRecall()), nil
}

func handleMemorySubtract(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("m- requires a value")
	}
	expr, err := operations.ParseExpression(args[0])
	if err != nil {
		return "", err
	}
	if len(expr.Operands) < 1 {
		return "", fmt.Errorf("invalid value")
	}
	calculator.MemorySubtract(expr.Operands[0])
	return fmt.Sprintf("Subtracted from memory: %v", calculator.MemoryRecall()), nil
}

func handlePrime(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("prime requires a number")
	}
	expr, err := operations.ParseExpression(args[0])
	if err != nil {
		return "", err
	}
	n := int(expr.Operands[0])
	if calculator.IsPrime(n) {
		return fmt.Sprintf("%d is prime", n), nil
	}
	return fmt.Sprintf("%d is not prime", n), nil
}

func handleFactors(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("factors requires a number")
	}
	expr, err := operations.ParseExpression(args[0])
	if err != nil {
		return "", err
	}
	n := int(expr.Operands[0])
	factors := calculator.PrimeFactors(n)
	return fmt.Sprintf("Prime factors of %d: %v", n, factors), nil
}

// processCommand processes a single command input.
func processCommand(input string) (string, bool) {
	input = strings.TrimSpace(input)

	// Empty input
	if input == "" {
		return "", false
	}

	// Exit commands
	switch strings.ToLower(input) {
	case "exit", "quit", "q":
		return "Goodbye!", true
	}

	// Check for built-in commands
	parts := strings.Fields(input)
	cmdName := strings.ToLower(parts[0])

	if cmd, ok := commands[cmdName]; ok {
		result, err := cmd.Handler(parts[1:])
		if err != nil {
			return fmt.Sprintf("Error: %v", err), false
		}
		return result, false
	}

	// Try to parse as expression
	expr, err := operations.ParseExpression(input)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), false
	}

	// Execute calculation
	result, err := calculator.Calculate(expr.Operator, expr.Operands...)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), false
	}

	return operations.FormatResult(result.Value, 6, true), false
}

// runInteractive runs the calculator in interactive mode.
func runInteractive(ctx context.Context) {
	fmt.Printf("%s v%s\n", AppName, Version)
	fmt.Println("Type 'help' for commands, 'quit' to exit")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return
		default:
		}

		fmt.Print("calc> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		result, shouldExit := processCommand(input)

		if result != "" {
			fmt.Println(result)
		}

		if shouldExit {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

// runBatch processes multiple expressions from arguments.
func runBatch(ctx context.Context, expressions []string) {
	// Filter comments and empty lines
	filtered := operations.FilterSlice(expressions, func(s string) bool {
		s = strings.TrimSpace(s)
		return s != "" && !strings.HasPrefix(s, "#")
	})

	// Process each expression
	for _, expr := range filtered {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, _ := processCommand(expr)
		fmt.Printf("%s = %s\n", expr, result)
	}
}

// runDemo demonstrates various calculator features.
func runDemo() {
	fmt.Println("Calculator Demo")
	fmt.Println("===============")
	fmt.Println()

	// Demonstrate generics with different types
	fmt.Println("Generic operations:")
	fmt.Printf("  Add[int](5, 3) = %d\n", operations.Add(5, 3))
	fmt.Printf("  Add[float64](2.5, 1.5) = %f\n", operations.Add(2.5, 1.5))
	fmt.Println()

	// Demonstrate higher-order functions
	fmt.Println("Higher-order functions:")
	numbers := []int{1, 2, 3, 4, 5}
	squares := operations.MapSlice(numbers, func(n int) int { return n * n })
	fmt.Printf("  MapSlice(%v, square) = %v\n", numbers, squares)

	evens := operations.FilterSlice(numbers, func(n int) bool { return n%2 == 0 })
	fmt.Printf("  FilterSlice(%v, even) = %v\n", numbers, evens)

	sum := operations.ReduceSlice(numbers, 0, func(acc, n int) int { return acc + n })
	fmt.Printf("  ReduceSlice(%v, sum) = %d\n", numbers, sum)
	fmt.Println()

	// Demonstrate functional options
	fmt.Println("Functional options:")
	config := operations.ApplyOptions(
		operations.WithPrecision(5),
		operations.WithHistoryLimit(50),
		operations.WithAngleMode("degrees"),
	)
	fmt.Printf("  Config: precision=%d, historyLimit=%d, angleMode=%s\n",
		config.Precision, config.HistoryLimit, config.AngleMode)
	fmt.Println()

	// Demonstrate Result type
	fmt.Println("Result type:")
	okResult := operations.Ok(42.0)
	fmt.Printf("  Ok(42).IsOk() = %v\n", okResult.IsOk())
	fmt.Printf("  Ok(42).Unwrap() = %v\n", okResult.Unwrap())

	errResult := operations.Err[float64](fmt.Errorf("something went wrong"))
	fmt.Printf("  Err.IsErr() = %v\n", errResult.IsErr())
	fmt.Printf("  Err.UnwrapOr(0) = %v\n", errResult.UnwrapOr(0))
	fmt.Println()

	// Demonstrate scientific calculator
	fmt.Println("Scientific calculator:")
	fact, _ := calculator.Factorial(10)
	fmt.Printf("  Factorial(10) = %d\n", fact)
	fib, _ := calculator.Fibonacci(20)
	fmt.Printf("  Fibonacci(20) = %d\n", fib)
	fmt.Printf("  IsPrime(17) = %v\n", calculator.IsPrime(17))
	fmt.Printf("  PrimeFactors(84) = %v\n", calculator.PrimeFactors(84))
	fmt.Printf("  GCD(48, 18) = %d\n", calculator.GCD(48, 18))
	fmt.Printf("  LCM(4, 6) = %d\n", calculator.LCM(4, 6))
	fmt.Println()

	// Demonstrate concurrent batch calculation
	fmt.Println("Concurrent batch calculation:")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ops := []struct {
		Op   string
		Args []float64
	}{
		{Op: "+", Args: []float64{100, 200}},
		{Op: "sqrt", Args: []float64{144}},
		{Op: "sin", Args: []float64{0}},
		{Op: "*", Args: []float64{7, 8}},
	}

	results := calculator.BatchCalculate(ctx, ops)
	for i, result := range results {
		if result.Success {
			fmt.Printf("  %s(%v) = %v\n", ops[i].Op, ops[i].Args, result.Value)
		} else {
			fmt.Printf("  %s(%v) = Error: %v\n", ops[i].Op, ops[i].Args, result.Error)
		}
	}
}

func main() {
	// Parse command line flags
	var (
		showHelp    = flag.Bool("help", false, "Show help")
		showVersion = flag.Bool("version", false, "Show version")
		runDemoFlag = flag.Bool("demo", false, "Run demo")
		batchMode   = flag.Bool("batch", false, "Run in batch mode")
	)
	flag.Parse()

	// Handle flags
	switch {
	case *showHelp:
		flag.Usage()
		result, _ := handleHelp(nil)
		fmt.Println(result)
		return
	case *showVersion:
		fmt.Printf("%s v%s\n", AppName, Version)
		return
	case *runDemoFlag:
		runDemo()
		return
	}

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal...")
		cancel()
	}()

	// Run appropriate mode
	if *batchMode || len(flag.Args()) > 0 {
		runBatch(ctx, flag.Args())
	} else {
		runInteractive(ctx)
	}
}
