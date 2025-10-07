# KFlow - Lightweight Go DAG Framework

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

KFlow is a lightweight DAG (Directed Acyclic Graph) execution framework for Go. It supports defining complex workflows via JSON configuration, offering flexible execution modes and robust error handling.

- 中文版 README: see [README.zh.md](README.zh.md)

## ✨ Features

- 🚀 Lightweight design — Simple APIs with minimal dependencies
- 📋 JSON configuration — Define DAG structure and execution strategies via JSON files
- 🔄 Multiple execution modes — Support serial, parallel, and async execution
- 🛡️ Error recovery — Built-in recover mechanism for fail-safe guarantees
- 📊 Layered execution — Sequential across layers; multiple modes within layers
- 🔧 Extensible — Easy-to-extend component interface design
 - 🧬 Workflow inheritance — Inherit/override/add/delete via `extends/remove`

## 📦 Installation

```bash
go get github.com/kangyujian/kflow
```

## 🚀 Quick Start

### 1. Define a Component

```go
package main

import (
    "context"
)

// Implement engine.Component interface
// Execute receives a shared DataContext
// Name returns the component name

type HelloComponent struct{ name string }

func (h *HelloComponent) Name() string { return h.name }

func (h *HelloComponent) Execute(ctx context.Context, data DataContext) error {
    data.Set("greeting", "Hello, "+h.name)
    return nil
}
```

### 2. Register a Component Factory

```go
// The factory implements Create and GetType
// Create receives engine.ComponentConfig and returns a component instance

type helloFactory struct{}

func (f *helloFactory) GetType() string { return "hello" }

func (f *helloFactory) Create(cfg engine.ComponentConfig) (engine.Component, error) {
    return &HelloComponent{name: cfg.Name}, nil
}
```

### 3. Create a JSON Configuration

```json
{
  "name": "hello_workflow",
  "version": "1.0.0",
  "description": "Sample workflow",
  "layers": [
    {
      "name": "layer1",
      "mode": "parallel",
      "components": [
        { "name": "hello1", "type": "hello", "config": {} },
        { "name": "hello2", "type": "hello", "config": {} }
      ],
      "timeout": 1,
      "enabled": true
    },
    {
      "name": "layer2",
      "mode": "serial",
      "components": [
        { "name": "hello3", "type": "hello", "config": {} }
      ],
      "dependencies": ["layer1"],
      "timeout": 1,
      "enabled": true
    }
  ]
}
```

### 4. Execute the Workflow

```go
package main

import (
    "context"
    "fmt"
    "github.com/kangyujian/kflow/engine"
)

func main() {
    // Parse configuration
    parser := engine.NewConfigParser()
    cfg, err := parser.ParseFile("workflow.json")
    if err != nil { panic(err) }

    // Register component factories
    registry := engine.NewComponentRegistry()
    registry.Register(&helloFactory{})

    // Create engine
    eng, err := engine.NewEngine(cfg, registry)
    if err != nil { panic(err) }

    // Shared, concurrency-safe data store
    data := engine.NewDataContext()

    // Execute
    if _, err := eng.Execute(context.Background(), data); err != nil {
        fmt.Printf("execution failed: %v\n", err)
        return
    }

    fmt.Printf("execution completed, data: %+v\n", data.Snapshot())
}
```

## 📖 Execution Modes

- Serial: Components execute in defined order
- Parallel: Components within a layer execute concurrently; waits for all to complete
- Async: Components execute asynchronously; does not block proceeding to next layer

## 📁 Project Structure

```
kflow/
├── README.md
├── go.mod
├── engine/
│   ├── component.go       # Component interface & registry
│   ├── config.go          # Config parsing & validation
│   ├── engine.go          # Engine & execution stats
│   └── layer.go           # Layer execution logic
├── example/
│   └── basic/
│       ├── components.go
│       ├── workflow.json
│       ├── main.go
│       └── data.txt / output.txt
└── docs/
    ├── architecture.md
    ├── config-spec.md
    └── api-reference.md
```

## 📚 Documentation

- English:
  - Docs: [Architecture (EN)](docs/architecture.en.md), [Config Spec (EN)](docs/config-spec.en.md), [API Reference (EN)](docs/api-reference.en.md)
  - Examples: [Inheritance Examples (EN)](docs/inheritance-examples.en.md)
  - Example: [Basic Example (EN)](example/basic/README.en.md)
- Chinese:
  - Docs: [Architecture (ZH)](docs/architecture.md), [Config Spec (ZH)](docs/config-spec.md), [API Reference (ZH)](docs/api-reference.md)
  - 示例： [继承案例 (ZH)](docs/inheritance-examples.md)
  - Example: [基础示例 (ZH)](example/basic/README.md)

## 🤝 Contributing

Issues and Pull Requests are welcome!

## 📄 License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.