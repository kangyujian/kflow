# KFlow Architecture (Updated)

This document describes the overall architecture and design concepts consistent with the current implementation.

## Design Principles
- Simplicity: Keep interfaces minimal and avoid unnecessary coupling.
- Flexibility: Support three layer execution modes — serial, parallel, and async. Components can optionally implement extended interfaces.
- Reliability: Critical component failures will stop the workflow; support retries and timeouts; strict config validation.

## Core Components

1) Engine
- Responsible for loading config, building layers and components, executing the workflow, and collecting execution stats.
- Supports pluggable options: logger, error handler, middleware.
- Key file: engine/engine.go

2) Layer
- Represents an ordered set of components with execution mode and parallelism.
- Validation rules: must have name and mode; non-empty component list; dependencies must exist.
- Key file: engine/layer.go

3) Component
- Base interface includes `Name()` and `Execute(ctx, data)`.
- Optional interfaces: initialization, cleanup, retry strategy, validation.
- Components are created via `ComponentRegistry` using factories and registered by `type`.
- Key file: engine/component.go

4) Config & Parsing
- JSON config maps to `Config`, `LayerConfig`, and `ComponentConfig`.
- Default values: `version`, `layer.mode=serial`, `enabled=true`, `component.timeout=30s`.
- Supports environment variable substitution and dependency checking.
- Key file: engine/config.go

## Execution Flow

1. Parse and validate configuration
- Use ConfigParser to parse the file, fill defaults, and validate fields.
- Filter layers and components using the `enabled` field.

2. Build components and layers
- Create component instances via the registry by `type`.
- Build executors for each layer (Serial/Parallel/Async).

3. Execute DAG
- Shared data is passed via concurrency-safe `DataContext`; components read/write via keys.
- Serial: execute one by one in order, return on error.
- Parallel: execute concurrently limited by `parallel`; collect errors and make decisions based on `critical`.
- Async: start without waiting for completion, immediately proceed to the next layer.

4. Retry & Critical Components
- Components may declare `retry` strategies or implement `RetryableComponent`; layer's `executeWithRetry` handles retries uniformly.
- Components with `critical=true` failing will cause layer or global failure.
 - Retry parameters: `max_retries` (excluding the initial attempt; total attempts = 1 + max_retries), `delay` (nanoseconds), `backoff` (factor). Layer delay formula: for the n-th retry, delay = `delay` × (`backoff` × (n-1)) — linear scaling, not exponential.
 - Early stop when `ShouldRetry(err)` returns false; retries exhausted return `RetryExhaustedError`.

5. Stats & Logging
- Engine collects `ExecutionStats`; each layer records start/end, success/failure, and duration.
- Custom Logger and Error Handler can be injected via options.

6. Timeout Control
- Global: `Config.timeout` sets a workflow-wide `context.WithTimeout` at the start.
- Layer: `LayerConfig.timeout` sets a new `context.WithTimeout` when entering the layer, affecting serial/parallel/async execution.
- Component: `ComponentConfig.timeout` (default 30s) is provided for component implementations; the engine does not automatically create a per-component timeout context. Components should honor cancellation in `Execute(ctx, data)` and may implement finer-grained control using their own `timeout`.

## Config Example (Excerpt)

```json
{
  "name": "basic_workflow",
  "version": "1.0.0",
  "layers": [
    {
      "name": "data_preparation",
      "mode": "parallel",
      "enabled": true,
      "components": [
        {
          "name": "file_reader",
          "type": "FileReader",
          "enabled": true,
          "timeout": 30000000000,
          "config": {"path": "data.txt"}
        }
      ]
    }
  ]
}
```

- `timeout` uses integer nanoseconds (Go native time.Duration JSON representation).
- `enabled` supports boolean or 0/1 (normalized to boolean during parsing).

## Differences from previous version
- Removed legacy fields: `is_core`, `execution_mode` (replaced by `critical` and `mode`).
- Component interface unified from `GetName()/IsCore()/Execute(ctx)` to `Name()/Execute(ctx, data)`.
- Engine API changed from global singleton to explicit creation passing `Config` and `ComponentRegistry`.

## Extensions & Middleware
- Middleware can inject logic before/after execution (e.g., monitoring, tracing, auditing).
- Error handler allows custom degradation, alerting, and recovery strategies.

## Performance & Reliability Tips
- Set reasonable `parallel` value for parallel layers to avoid resource contention.
- Handle context cancellation and timeouts in component implementations.
- Enable retry strategies for critical components; use exponential backoff to control load.

For more details, see:
- README.md (EN) and README.zh.md (ZH)
- docs/api-reference.en.md (EN) and docs/api-reference.md (ZH)
- docs/config-spec.en.md (EN) and docs/config-spec.md (ZH)