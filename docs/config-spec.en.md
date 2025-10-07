# KFlow JSON Configuration Specification (Updated)

## Overview

KFlow uses JSON configuration files to define DAG structure and execution strategies. This document reflects fields and examples consistent with the current engine implementation.

## Configuration Structure

### Root Configuration Object

```json
{
  "name": "workflow_name",
  "description": "workflow description",
  "version": "1.0.0",
  "timeout": 0,
  "extends": "./base_workflow.json",
  "layers": [
    // array of layer configs
  ],
  "global": {
    // global parameters (optional)
  },
  "metadata": {
    // metadata (optional)
  }
}
```

#### Field Descriptions

| Field | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | ✅ | - | Unique workflow name |
| `description` | string | ❌ | "" | Workflow description |
| `version` | string | ❌ | "1.0.0" | Configuration version |
| `timeout` | number | ❌ | 0 | Overall workflow timeout in nanoseconds (Go `time.Duration` JSON), e.g., 5000000000 for 5s |
| `layers` | array | ✅ | - | Array of layer configurations |
| `global` | object | ❌ | {} | Global parameters passed to all components (usage may depend on custom logic) |
| `metadata` | object | ❌ | {} | Extra metadata |
| `extends` | string | ❌ | - | Path to parent workflow JSON (relative or absolute). If provided, the parser loads the parent and merges it into the child.

### Layer Configuration Object

```json
{
  "name": "layer_name",
  "mode": "parallel",
  "timeout": 0,
  "components": [
    // array of component configs
  ],
  "dependencies": ["layer1", "layer2"],
  "enabled": true,
"parallel": 0
}
```

#### Field Descriptions

| Field | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | ✅ | - | Layer name, unique within the workflow |
| `mode` | string | ❌ | serial | Execution mode: serial/parallel/async |
| `timeout` | number | ❌ | 0 | Layer execution timeout in nanoseconds |
| `components` | array | ✅ | - | Array of component configurations |
| `dependencies` | array | ❌ | [] | Names of dependent layers that must precede the current layer |
| `enabled` | bool | ❌ | true | Whether the layer is enabled |
| `parallel` | number | ❌ | 0 | Concurrency limit for parallel mode (0 means unlimited) |
| `remove` | bool | ❌ | false | When merging inheritance, if true, delete the layer |

### Component Configuration Object

```json
{
  "name": "component_name",
  "type": "component_type",
  "enabled": true,
  "timeout": 0,
  "dependencies": ["comp1", "comp2"],
  "config": {
    // component-specific configuration
  },
  "retry": {
    "max_retries": 0,
    "delay": 0,
    "backoff": 1.0
  }
}
```

#### Field Descriptions

| Field | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | ✅ | - | Component name, unique within the layer |
| `type` | string | ✅ | - | Component type used by factory creation |
| `enabled` | bool | ❌ | true | Whether the component is enabled |
| `timeout` | number | ❌ | 0 | Component execution timeout in nanoseconds |
| `dependencies` | array | ❌ | [] | Dependent component names (component-level dependency not strictly enforced in current implementation) |
| `config` | object | ❌ | {} | Component-specific configuration |
| `retry` | object | ❌ | null | Retry configuration including max retries, delay (nanoseconds), and backoff factor |
| `remove` | bool | ❌ | false | When merging inheritance, if true, delete the component |

## Execution Modes

- Serial: components execute sequentially in the defined order
- Parallel: all components within a layer execute concurrently and wait for completion
- Async: components execute asynchronously and proceed to the next layer immediately

## Retry Configuration Details

- Two usage patterns:
  - Component implements `RetryableComponent`, returns `GetRetryConfig()`, and uses `ShouldRetry(err)` to decide whether to continue.
  - Provide a `retry` object in the component config (`max_retries`, `delay`, `backoff`); the component factory/implementation reads and applies it.
- Field semantics:
  - `max_retries`: maximum retries excluding the initial attempt; total attempts = 1 + `max_retries`.
  - `delay`: initial retry delay (nanoseconds).
  - `backoff`: backoff factor. In the current layer implementation, the delay for the n-th retry (n starts at 1) is `delay` × (`backoff` × (n-1)). This is linear scaling with the factor, not exponential power.
- Behavior:
  - The engine only performs unified retry for components that implement `RetryableComponent`.
  - If `ShouldRetry(err)` returns false, retry stops early and the current error is returned.
  - When retries are exhausted, a `RetryExhaustedError` is returned, which includes the last error and the list of all attempt errors.

Example (component-level retry config):

```json
{
  "name": "http_fetcher",
  "type": "http_client",
  "timeout": 30000000000,
  "retry": { "max_retries": 3, "delay": 1000000000, "backoff": 2.0 },
  "config": { "endpoint": "https://api.example.com" }
}
```

## Timeout Control

- Global timeout: `Config.timeout` sets the overall workflow timeout. The engine creates a `context.WithTimeout` at the start, shared by all layers and components.
- Layer timeout: `LayerConfig.timeout` creates a new `context.WithTimeout` when entering the layer, affecting serial/parallel/async execution within that layer.
- Component timeout: `ComponentConfig.timeout` defaults to 30s during parsing when not explicitly set. This is provided for component implementations; the engine does not automatically create a dedicated timeout context per component. Components should respect cancellation from the passed `ctx` and can implement finer-grained control internally using their own `timeout`.
- Timeout errors: When the context is cancelled (timeout or manual cancel), components should return `ctx.Err()`. Errors may appear as `ExecutionError` or custom wrappers (e.g., `TimeoutError`) in logs and stats.

## Inheritance (extends) and Merge Semantics

Workflow B can inherit workflow A by setting `extends` in the root config. The parser loads the parent and merges using the following rules:

- Root field override: child `name`, `version`, `description`, `timeout`, `global`, and `metadata` override the parent when provided (for `global`/`metadata`, keys in the child override keys in the parent).
- Layer merge:
  - `remove: true` deletes the layer with the same name in the parent.
  - Same-name layer field overrides: `mode`, `timeout`, `enabled`, `parallel`, `dependencies`; unspecified fields remain from the parent.
  - Components are merged by name:
    - `remove: true` deletes the component.
    - Same-name component overrides `type`, `timeout`, `enabled`, `dependencies`; `config` uses key-level merge (child keys override parent keys); `retry` overrides entirely when provided.
    - Nonexistent components are treated as additions.
- New layers: child layers not present in the parent are appended.
- Cycle detection: circular inheritance (e.g., A extends B and B extends A) yields `extends_cycle_detected`.

Example (B extends A and performs add/delete/patch):

```json
{
  "extends": "./workflow-a.json",
  "name": "workflow-b",
  "layers": [
    { "name": "L1", "remove": true },
    { "name": "L2", "mode": "parallel", "parallel": 8 },
    { "name": "L3", "components": [
      { "name": "C1", "remove": true },
      { "name": "C2", "config": { "threshold": 0.9 } },
      { "name": "C_new", "type": "MyComp", "config": {"foo": "bar"} }
    ]}
  ]
}
```

## Example

### Complete Example

```json
{
  "name": "data_processing_workflow",
  "description": "Data processing workflow example",
  "version": "1.0.0",
  "timeout": 0,
  "layers": [
    {
      "name": "data_preparation",
      "mode": "parallel",
      "components": [
        { "name": "data_loader", "type": "file_reader", "config": { "file_path": "data.txt", "encoding": "utf-8" } },
        { "name": "config_loader", "type": "config_reader", "config": { "config_path": "config.yaml" } }
      ],
      "timeout": 5000000000,
      "enabled": true
    },
    {
      "name": "data_processing",
      "mode": "serial",
      "components": [
        { "name": "data_transformer", "type": "transformer", "config": { "operations": ["uppercase", "trim"] } },
        { "name": "data_validator", "type": "validator", "config": { "rules": ["not_empty", "max_length:500"] } }
      ],
      "dependencies": ["data_preparation"],
      "timeout": 10000000000,
      "enabled": true
    },
    {
      "name": "data_output",
      "mode": "serial",
      "components": [
        { "name": "data_writer", "type": "file_writer", "config": { "output_path": "output.txt", "append": false } },
        { "name": "notifier", "type": "logger", "config": { "level": "info", "message": "Data processing completed" } }
      ],
      "dependencies": ["data_processing"],
      "timeout": 5000000000,
      "enabled": true
    }
  ]
}
```

## Notes

- All `timeout`/`delay` fields use nanoseconds to align with Go `time.Duration` JSON deserialization.
- The engine sets some defaults, e.g., `mode` defaults to `serial` when omitted, and `enabled` defaults to `true` when not explicitly set.
- Environment variable substitution supports `${VAR}` or `${VAR:default}` syntax in JSON.