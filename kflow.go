package kflow

import (
	"github.com/kangyujian/kflow/engine"
)

// Re-export core types from engine package
type (
	// Component interfaces
	Component             = engine.Component
	InitializableComponent = engine.InitializableComponent
	CleanupComponent      = engine.CleanupComponent
	RetryableComponent    = engine.RetryableComponent
	ValidatableComponent  = engine.ValidatableComponent

	// Configuration types
	ComponentConfig = engine.ComponentConfig
	RetryConfig     = engine.RetryConfig
	LayerConfig     = engine.LayerConfig
	Config          = engine.Config

	// Factory types
	ComponentFactory  = engine.ComponentFactory
	ComponentRegistry = engine.ComponentRegistry

	// Engine types
	Engine         = engine.Engine
	ExecutionStats = engine.ExecutionStats
	LayerStats     = engine.LayerStats
	Logger         = engine.Logger
	ErrorHandler   = engine.ErrorHandler
	Middleware     = engine.Middleware

	// Layer types
	Layer         = engine.Layer
	ExecutionMode = engine.ExecutionMode

	// Error types
	ComponentError        = engine.ComponentError
	ConfigError          = engine.ConfigError
	ExecutionError       = engine.ExecutionError
	TimeoutError         = engine.TimeoutError
	RetryExhaustedError  = engine.RetryExhaustedError
	CriticalComponentError = engine.CriticalComponentError
	ValidationError      = engine.ValidationError

	// Parser type
	ConfigParser = engine.ConfigParser
)

// Re-export constants
const (
	SerialMode   = engine.SerialMode
	ParallelMode = engine.ParallelMode
	AsyncMode    = engine.AsyncMode
)

// Re-export constructor functions
var (
	NewComponentRegistry = engine.NewComponentRegistry
	NewEngine           = engine.NewEngine
	NewLayer            = engine.NewLayer
	NewConfigParser     = engine.NewConfigParser
)