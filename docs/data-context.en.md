# Data Passing & Shared DataContext (KFlow)

DataContext is KFlow’s concurrency-safe key-value store for passing and sharing data between components. It works reliably across serial, parallel, and async execution modes.

## Capabilities
- Concurrency-safe: protected by `RWMutex`, suitable for parallel layers.
- Convenient API: `Set/Get/GetString/Delete/Has/Snapshot` methods.
- Snapshot: `Snapshot()` returns a shallow copy for debugging/output.
- Initialization: create empty or with initial values.

## Interface Overview

```go
type DataContext interface {
    Set(key string, value interface{})
    Get(key string) (interface{}, bool)
    GetString(key string) (string, bool)
    Delete(key string)
    Has(key string) bool
    Snapshot() map[string]interface{}
}
```

The default implementation lives in `engine/datacontext.go`. Use `NewDataContext()` and `NewDataContextWith(initial)` to construct.

## Initialization & Basic Usage

```go
// Create an empty shared context
data := engine.NewDataContext()

// Or with initial values
data := engine.NewDataContextWith(map[string]interface{}{
    "input_path": "data.txt",
    "retry_count": 0,
})

// Component read/write examples
data.Set("file_data", []byte("hello"))
if v, ok := data.Get("file_data"); ok {
    b := v.([]byte)
    _ = b
}

// String shortcut
if s, ok := data.GetString("user_id"); ok {
    // use s
}

// Check & delete
exists := data.Has("transformed")
data.Delete("file_data")

// Snapshot for debugging/output (shallow copy)
snapshot := data.Snapshot()
fmt.Printf("data: %+v\n", snapshot)
```

## Component Integration

Use `data` inside component `Execute(ctx, data)` to share and read information:

```go
type TransformComponent struct{ name string }

func (t *TransformComponent) Name() string { return t.name }

func (t *TransformComponent) Execute(ctx context.Context, data engine.DataContext) error {
    v, ok := data.Get("file_data")
    if !ok { return fmt.Errorf("missing file_data") }
    b := v.([]byte)
    // ...transform...
    data.Set("transformed_data", b)
    return nil
}
```

Components cooperatively pass values via agreed keys (e.g., `file_data`, `transformed_data`, `output_path`, `errors`).

## Concurrency & Notes
- Concurrency-safe: all reads/writes are lock-protected and safe for parallel execution.
- Shallow copy: `Snapshot()` returns a shallow copy; `map/slice` values still reference originals—deep copy if needed.
- Type assertions: `Get` returns `interface{}`—assert types and handle errors defensively.
- Key naming: use readable, semantic keys like `input_path`, `raw_data`, `transformed_data`.
- Data shape: prefer serializable primitive/JSON-like structures to simplify debugging and output.

## Suggested Key Conventions
- `input_path`: input file path (string)
- `file_data`: read file content ([]byte/string)
- `transformed_data`: transformed data ([]byte/string/map)
- `output_path`: output file path (string)
- `errors`: accumulated errors ([]error / []string)

See `engine/datacontext.go` and `docs/api-reference.en.md` for details.