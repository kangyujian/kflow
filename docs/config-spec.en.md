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

## Execution Modes

- Serial: components execute sequentially in the defined order
- Parallel: all components within a layer execute concurrently and wait for completion
- Async: components execute asynchronously and proceed to the next layer immediately

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