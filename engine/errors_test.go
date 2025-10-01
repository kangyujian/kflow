package engine

import (
	"errors"
	"testing"
	"time"
)

func TestComponentError(t *testing.T) {
	t.Run("Error with component name", func(t *testing.T) {
		err := &ComponentError{
			Type:      "test_error",
			Message:   "test message",
			Component: "test_component",
			Layer:     "test_layer",
		}
		
		expected := "component error in test_component: test message"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without component name", func(t *testing.T) {
		err := &ComponentError{
			Type:    "test_error",
			Message: "test message",
		}
		
		expected := "component error: test message"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &ComponentError{
			Type:    "test_error",
			Message: "test message",
			Cause:   cause,
		}
		
		if err.Unwrap() != cause {
			t.Errorf("Expected %v, got %v", cause, err.Unwrap())
		}
	})

	t.Run("Unwrap returns nil when no cause", func(t *testing.T) {
		err := &ComponentError{
			Type:    "test_error",
			Message: "test message",
		}
		
		if err.Unwrap() != nil {
			t.Errorf("Expected nil, got %v", err.Unwrap())
		}
	})
}

func TestConfigError(t *testing.T) {
	t.Run("Error with field name", func(t *testing.T) {
		err := &ConfigError{
			Type:    "validation_error",
			Message: "invalid value",
			Field:   "test_field",
		}
		
		expected := "config error in field test_field: invalid value"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without field name", func(t *testing.T) {
		err := &ConfigError{
			Type:    "validation_error",
			Message: "invalid value",
		}
		
		expected := "config error: invalid value"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &ConfigError{
			Type:    "validation_error",
			Message: "invalid value",
			Cause:   cause,
		}
		
		if err.Unwrap() != cause {
			t.Errorf("Expected %v, got %v", cause, err.Unwrap())
		}
	})
}

func TestExecutionError(t *testing.T) {
	t.Run("Error with component and layer", func(t *testing.T) {
		timestamp := time.Now()
		err := &ExecutionError{
			Type:      "runtime_error",
			Message:   "execution failed",
			Component: "test_component",
			Layer:     "test_layer",
			Timestamp: timestamp,
		}
		
		expected := "execution error in layer test_layer, component test_component: execution failed"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without component", func(t *testing.T) {
		timestamp := time.Now()
		err := &ExecutionError{
			Type:      "runtime_error",
			Message:   "execution failed",
			Timestamp: timestamp,
		}
		
		expected := "execution error: execution failed"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &ExecutionError{
			Type:      "runtime_error",
			Message:   "execution failed",
			Timestamp: time.Now(),
			Cause:     cause,
		}
		
		if err.Unwrap() != cause {
			t.Errorf("Expected %v, got %v", cause, err.Unwrap())
		}
	})
}

func TestTimeoutError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		err := &TimeoutError{
			Component: "test_component",
			Layer:     "test_layer",
			Timeout:   5 * time.Second,
		}
		
		expected := "timeout in layer test_layer, component test_component after 5s"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})
}

func TestRetryExhaustedError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		lastError := errors.New("last error")
		err := &RetryExhaustedError{
			Component:  "test_component",
			MaxRetries: 3,
			LastError:  lastError,
		}
		
		expected := "retry exhausted for component test_component after 3 attempts: last error"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns last error", func(t *testing.T) {
		lastError := errors.New("last error")
		err := &RetryExhaustedError{
			Component:  "test_component",
			MaxRetries: 3,
			LastError:  lastError,
		}
		
		if err.Unwrap() != lastError {
			t.Errorf("Expected %v, got %v", lastError, err.Unwrap())
		}
	})
}

func TestCriticalComponentError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		cause := errors.New("test cause")
		err := &CriticalComponentError{
			Component: "test_component",
			Layer:     "test_layer",
			Cause:     cause,
		}
		
		expected := "critical component test_component failed in layer test_layer: test cause"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("root cause")
		err := &CriticalComponentError{
			Component: "test_component",
			Layer:     "test_layer",
			Cause:     cause,
		}
		
		if err.Unwrap() != cause {
			t.Errorf("Expected %v, got %v", cause, err.Unwrap())
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		err := &ValidationError{
			Field:   "test_field",
			Value:   "invalid_value",
			Message: "must be a valid format",
		}
		
		expected := "validation error for field test_field: must be a valid format"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error with nil value", func(t *testing.T) {
		err := &ValidationError{
			Field:   "test_field",
			Value:   nil,
			Message: "cannot be nil",
		}
		
		expected := "validation error for field test_field: cannot be nil"
		if err.Error() != expected {
			t.Errorf("Expected %q, got %q", expected, err.Error())
		}
	})
}