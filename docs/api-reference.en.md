# KFlow API Reference (Updated)

This document aligns with the current codebase, summarizing core types, interfaces, and usage. Examples and field names reflect the engine source.

## Core Modules and Files
- Config & Parsing: engine/config.go
- Component Interface & Registry: engine/component.go
- Layer Execution Logic: engine/layer.go
- Engine & Execution Stats: engine/engine.go

## Configuration Structs

```go
// Root config
type Config struct {
    Name        string                 `json:"name"`
    Version     string                 `json:"version"`
    Description string                 `json:"description"`
    Layers      []LayerConfig          `json:"layers"`
    Global      map[string]interface{} `json:"global,omitempty"`
    Timeout     time.Duration          `json:"timeout,omitempty"`
    Metadata    map[string]string      `json:"metadata,omitempty"`
    Extends     string                 `json:"extends,omitempty"`
}

// Layer config
type LayerConfig struct {
    Name         string            `json:"name"`
    Mode         ExecutionMode     `json:"mode"`           // serial/parallel/async
    Components   []ComponentConfig `json:"components"`
    Timeout      time.Duration     `json:"timeout"`
    Dependencies []string          `json:"dependencies"`
    Enabled      bool              `json:"enabled"`
    Parallel     int               `json:"parallel,omitempty"` // parallelism limit
    Remove       bool              `json:"remove,omitempty"`
}

// Component config
type ComponentConfig struct {
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Config       map[string]interface{} `json:"config"`
    Dependencies []string               `json:"dependencies"`
    Timeout      time.Duration          `json:"timeout"`
    Retry        *RetryConfig           `json:"retry,omitempty"`
    Critical     bool                   `json:"critical"`
    Enabled      bool                   `json:"enabled"`
    Remove       bool                   `json:"remove,omitempty"`
}

// Retry config
type RetryConfig struct {
    MaxRetries int           `json:"max_retries"`
    Delay      time.Duration `json:"delay"`
    Backoff    float64       `json:"backoff"`
}
```

- Execution modes: `serial` / `parallel` / `async`.
- Defaults: layer and component `enabled` default to true when not set; component `timeout` defaults to 30s; layer `mode` defaults to serial.
- Environment variable substitution supports `${VAR}` and `${VAR:default}`.

## Inheritance & Merge
- Root-level `extends`: a child workflow can inherit from a parent workflow (file path or identifier).
- Layer/Component `remove: true`: delete the corresponding layer or component during inheritance merge.
- Field overrides: child overrides parent fields with the same name; `enabled/critical` only override to true when explicitly set to true; unset values do not override parent values.
- Detailed rules: see Config Spec [ZH](docs/config-spec.md) and [EN](docs/config-spec.en.md).

## Component Interfaces

```go
// Base component interface
type Component interface {
    Name() string
    Execute(ctx context.Context, data DataContext) error
}

// Optional extension interfaces
type InitializableComponent interface { Initialize(ctx context.Context) error }
type CleanupComponent      interface { Cleanup(ctx context.Context) error }

type RetryableComponent interface {
    Component
    ShouldRetry(err error) bool
    GetRetryConfig() RetryConfig
}

type ValidatableComponent interface {
    Component
    Validate() error
}
```

Shared data is passed via `DataContext` with concurrency-safe `Set/Get/GetString/Delete/Has/Snapshot` methods; components read/write by keys (e.g., `file_data`, `transformed_data`).

### Data Passing with DataContext
- Init: `NewDataContext()` or `NewDataContextWith(map[string]interface{})`.
- Common keys: `input_path`, `file_data`, `transformed_data`, `output_path`, `errors`.
- Snapshot: `Snapshot()` returns a shallow copy for debugging/output.
- Concurrency: safe for parallel layers; deep copy reference types when necessary.
- Details & examples: see <mcfile name="data-context.en.md" path="/Users/kangyujian/goProject/kflow/docs/data-context.en.md"></mcfile>.

## Component Factory & Registry

```go
// Factory interface
type ComponentFactory interface {
    Create(config ComponentConfig) (Component, error)
    GetType() string
}

// Registry
type ComponentRegistry struct {
    factories map[string]ComponentFactory
}

func NewComponentRegistry() *ComponentRegistry
func (r *ComponentRegistry) Register(factory ComponentFactory)
func (r *ComponentRegistry) Create(config ComponentConfig) (Component, error)
func (r *ComponentRegistry) GetRegisteredTypes() []string
```

## Engine API

```go
// Creation & execution
func NewEngine(cfg *Config, registry *ComponentRegistry, options ...EngineOption) (*Engine, error)
func (e *Engine) Execute(ctx context.Context, data DataContext) (*ExecutionStats, error)

// Query & validation
func (e *Engine) GetConfig() *Config
func (e *Engine) Validate() error
func (e) GetLayers() []*Layer
func (e) GetLayer(name string) (*Layer, bool)
```

- EngineOption: supports WithLogger, WithErrorHandler, WithMiddleware.
- ExecutionStats: includes total duration, per-layer stats, success/failure flags, and error info.

## Layer Execution & Critical Components
- Serial: execute in order; return immediately on error.
- Parallel: use semaphore to limit concurrency (`parallel`); collect errors; abort on critical component failure (`Critical`).
- Async: start and proceed to next layer without waiting.

Critical components are marked by the `critical` field in component config; failures produce `CriticalComponentError`, terminating the layer or entire workflow.

## Error Types (Examples)
- ConfigError: config parsing/validation errors
- ComponentError: component creation/initialization/cleanup errors
- ExecutionError: unified wrapper for component execution failures
- CriticalComponentError: critical component failure
- RetryExhaustedError: retries exhausted

## Usage Example (Simplified)

```go
parser := engine.NewConfigParser()
cfg, _ := parser.ParseFile("workflow.json")
registry := engine.NewComponentRegistry()
registry.Register(&yourFactory{})
eng, _ := engine.NewEngine(cfg, registry)
data := NewDataContext()
stats, err := eng.Execute(context.Background(), data)
```

See also README.md (EN), README.zh.md (ZH), and examples in example/basic.