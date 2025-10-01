package engine

import (
	"fmt"
	"time"
)

// ComponentError 组件错误
type ComponentError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Component string `json:"component,omitempty"`
	Layer     string `json:"layer,omitempty"`
	Cause     error  `json:"cause,omitempty"`
}

func (e *ComponentError) Error() string {
	if e.Component != "" {
		return fmt.Sprintf("component error in %s: %s", e.Component, e.Message)
	}
	return fmt.Sprintf("component error: %s", e.Message)
}

func (e *ComponentError) Unwrap() error {
	return e.Cause
}

// ConfigError 配置错误
type ConfigError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Cause   error  `json:"cause,omitempty"`
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config error in field %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// ExecutionError 执行错误
type ExecutionError struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
	Layer     string    `json:"layer,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Cause     error     `json:"cause,omitempty"`
}

func (e *ExecutionError) Error() string {
	if e.Component != "" && e.Layer != "" {
		return fmt.Sprintf("execution error in layer %s, component %s: %s", e.Layer, e.Component, e.Message)
	} else if e.Component != "" {
		return fmt.Sprintf("execution error in component %s: %s", e.Component, e.Message)
	}
	return fmt.Sprintf("execution error: %s", e.Message)
}

func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// TimeoutError 超时错误
type TimeoutError struct {
	Component string        `json:"component"`
	Layer     string        `json:"layer"`
	Timeout   time.Duration `json:"timeout"`
}

func (e *TimeoutError) Error() string {
	if e.Layer != "" {
		return fmt.Sprintf("timeout in layer %s, component %s after %v", e.Layer, e.Component, e.Timeout)
	}
	return fmt.Sprintf("timeout in component %s after %v", e.Component, e.Timeout)
}

// RetryExhaustedError 重试耗尽错误
type RetryExhaustedError struct {
	Component   string  `json:"component"`
	MaxRetries  int     `json:"max_retries"`
	LastError   error   `json:"last_error"`
	RetryErrors []error `json:"retry_errors"`
}

func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("retry exhausted for component %s after %d attempts: %v", e.Component, e.MaxRetries, e.LastError)
}

func (e *RetryExhaustedError) Unwrap() error {
	return e.LastError
}

// CriticalComponentError 关键组件错误
type CriticalComponentError struct {
	Component string `json:"component"`
	Layer     string `json:"layer"`
	Cause     error  `json:"cause"`
}

func (e *CriticalComponentError) Error() string {
	return fmt.Sprintf("critical component %s failed in layer %s: %v", e.Component, e.Layer, e.Cause)
}

func (e *CriticalComponentError) Unwrap() error {
	return e.Cause
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}
