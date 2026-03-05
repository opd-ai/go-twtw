// Package simple provides a simple example package used as a test fixture
// for the go-twtw analyzer. It covers every construct kind the analyzer must
// handle: constants, variables, type aliases, structs, interfaces, functions,
// and methods, including channel and goroutine usage.
package simple

import "errors"

// MaxItems is the maximum number of items that can be processed at once.
const MaxItems = 100

// defaultTimeout is the default processing timeout in milliseconds.
var defaultTimeout = 5000

// Status represents the current processing status of an operation.
type Status string

// Config holds configuration for the processor.
type Config struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int
	// Timeout is the timeout in milliseconds.
	Timeout int
	// ResultChan is the channel results are written to.
	ResultChan chan Result
}

// Processor defines the interface for all data processors.
type Processor interface {
	// Process processes the given input and returns a result.
	Process(input string) (Result, error)
	// Reset resets the processor to its initial state.
	Reset()
}

// Result holds the output of a single processing operation.
type Result struct {
	// Value is the processed output value.
	Value string
	// Success indicates whether processing succeeded.
	Success bool
}

// NewConfig creates a new Config with sensible default values.
func NewConfig() *Config {
	return &Config{
		MaxRetries: 3,
		Timeout:    defaultTimeout,
		ResultChan: make(chan Result, MaxItems),
	}
}

// Process processes input according to the configuration settings.
func (c *Config) Process(input string) (Result, error) {
	if input == "" {
		return Result{}, errors.New("empty input")
	}
	return Result{Value: input, Success: true}, nil
}

// Reset resets the Config to its default state.
func (c *Config) Reset() {
	c.MaxRetries = 3
	c.Timeout = defaultTimeout
}

// RunProcessor runs p in a new goroutine, writing results to the results channel.
func RunProcessor(p Processor, input string, results chan<- Result) {
	go func() {
		result, err := p.Process(input)
		if err == nil {
			results <- result
		}
	}()
}
