# Inheritance Examples Guide (KFlow)

This guide demonstrates workflow inheritance with clear examples: how a child workflow performs add/delete/modify and field overrides on a parent workflow, and how merge rules affect execution.

## Goals
- Inherit from Workflow A and delete specific layers/components.
- Override fields (e.g., `mode`, `timeout`, `enabled`).
- Modify or add components in an existing layer; add an entire new layer.

## Parent Workflow A (workflowA.json)

```json
{
  "name": "workflowA",
  "version": "1.0.0",
  "description": "Parent workflow A",
  "global": { "env": "prod" },
  "metadata": { "owner": "team-A" },
  "layers": [
    {
      "name": "prepare",
      "mode": "serial",
      "timeout": 10,
      "enabled": true,
      "components": [
        { "name": "read", "type": "file", "config": {"path": "input.txt"}, "enabled": true },
        { "name": "validate", "type": "validator", "config": {}, "critical": true }
      ]
    },
    {
      "name": "process",
      "mode": "parallel",
      "timeout": 20,
      "components": [
        { "name": "transform", "type": "transform", "config": {} },
        { "name": "analyze", "type": "analyzer", "config": {} }
      ]
    },
    {
      "name": "cleanup",
      "mode": "serial",
      "timeout": 5,
      "components": [ { "name": "clean", "type": "cleanup", "config": {} } ]
    }
  ]
}
```

## Child Workflow B (workflowB.json) — Inheritance & Edits

```json
{
  "name": "workflowB",
  "extends": "workflowA.json",
  "description": "Child workflow B: inherit A and modify",
  "metadata": { "owner": "team-B" },
  "layers": [
    { "name": "cleanup", "remove": true },
    {
      "name": "prepare",
      "mode": "parallel",
      "timeout": 15,
      "components": [
        { "name": "read", "type": "file", "config": {"path": "${INPUT_PATH:default.txt}"} },
        { "name": "validate", "type": "validator", "critical": true },
        { "name": "normalize", "type": "normalizer", "config": {}, "enabled": true }
      ]
    },
    {
      "name": "export",
      "mode": "serial",
      "timeout": 30,
      "components": [ { "name": "write", "type": "file", "config": {"path": "out.txt"} } ]
    }
  ]
}
```

## Merge Result (Key Points)
- Delete: `cleanup` layer is removed (`remove: true`).
- Modify: `prepare` layer `mode` changes from serial to parallel, `timeout` from 10 to 15; components `read/validate` remain, new component `normalize` is added.
- Add: New `export` layer appended to the merged layers.
- Override: root `metadata.owner` becomes `team-B`; `global` and `metadata` are merged at key level (child overrides parent).
- Boolean fields: `enabled/critical` only override to `true` when explicitly set to `true`; unset values do not affect parent values.

## Run Example (Go)

```go
parser := engine.NewConfigParser()
cfg, err := parser.ParseFile("workflowB.json")
if err != nil { panic(err) }

registry := engine.NewComponentRegistry()
// Register required component factories...

eng, err := engine.NewEngine(cfg, registry)
if err != nil { panic(err) }

stats, err := eng.Execute(context.Background(), engine.NewDataContext())
if err != nil { panic(err) }
fmt.Printf("done: %+v\n", stats)
```

## Rule Overview
- Root-level `extends` specifies the parent workflow; the parser handles loading, cycle detection, and merging.
- Layer/Component: existing → modify; non-existing → add; `remove: true` → delete.
- `retry`, `timeout`, `dependencies`, `parallel` follow child-overrides-parent strategy.
- See detailed semantics:
  - Config Spec (ZH): docs/config-spec.md
  - Config Spec (EN): docs/config-spec.en.md