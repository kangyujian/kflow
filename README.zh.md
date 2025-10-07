# KFlow - 轻量级 Go DAG 框架

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

KFlow 是一个轻量级的 Go 语言 DAG（有向无环图）执行框架，支持通过 JSON 配置文件定义复杂的工作流程，提供灵活的执行模式和强大的错误处理能力。

## ✨ 特性

- 🚀 **轻量级设计** - 简洁的 API，最小化的依赖
- 📋 **JSON 配置** - 通过 JSON 文件定义 DAG 结构和执行策略
- 🔄 **多种执行模式** - 支持串行、并行、异步执行
- 🛡️ **错误恢复** - 内置 recover 机制，提供兜底保障
- 📊 **层级执行** - 层与层之间顺序执行，层内支持多种执行模式
- 🔧 **可扩展** - 易于扩展的组件接口设计
 - 🧬 **工作流继承** - 通过 `extends/remove` 实现继承、增删改

## 📦 安装

```bash
go get github.com/kangyujian/kflow
```

## 🚀 快速开始

### 1. 定义组件

```go
package main

import (
    "context"
)

// 实现 engine.Component 接口
// 注意：Execute 需要接受共享数据 data
// Name 返回组件名称

type HelloComponent struct{ name string }

func (h *HelloComponent) Name() string { return h.name }

func (h *HelloComponent) Execute(ctx context.Context, data DataContext) error {
    data.Set("greeting", "Hello, "+h.name)
    return nil
}
```

### 2. 注册组件工厂

```go
// 组件工厂需要实现 Create 和 GetType
// Create 接受 engine.ComponentConfig 并返回组件实例

type helloFactory struct{}

func (f *helloFactory) GetType() string { return "hello" }

func (f *helloFactory) Create(cfg engine.ComponentConfig) (engine.Component, error) {
    return &HelloComponent{name: cfg.Name}, nil
}
```

### 3. 创建 JSON 配置

```json
{
  "name": "hello_workflow",
  "version": "1.0.0",
  "description": "示例工作流",
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

### 4. 执行工作流

```go
package main

import (
    "context"
    "fmt"
    "github.com/kangyujian/kflow/engine"
)

func main() {
    // 解析配置
    parser := engine.NewConfigParser()
    cfg, err := parser.ParseFile("workflow.json")
    if err != nil { panic(err) }

    // 注册组件工厂
    registry := engine.NewComponentRegistry()
    registry.Register(&helloFactory{})

    // 创建引擎
    eng, err := engine.NewEngine(cfg, registry)
    if err != nil { panic(err) }

    // 共享数据存储（并发安全）
    data := engine.NewDataContext()

    // 执行
    if _, err := eng.Execute(context.Background(), data); err != nil {
        fmt.Printf("执行失败: %v\n", err)
        return
    }

    fmt.Printf("执行完成, 数据: %+v\n", data.Snapshot())
}
```

## 📖 执行模式

- 串行执行 (Serial): 组件按定义顺序依次执行
- 并行执行 (Parallel): 层内组件并发执行，等待全部完成
- 异步执行 (Async): 组件异步执行，不阻塞进入下一层

## 📁 项目结构

```
kflow/
├── README.md
├── go.mod
├── engine/
│   ├── component.go       # 组件接口与注册表
│   ├── config.go          # 配置解析与校验
│   ├── engine.go          # 引擎与执行统计
│   └── layer.go           # 层执行逻辑
├── example/
│   └── basic/             # 基础示例
│       ├── components.go
│       ├── workflow.json
│       ├── main.go
│       └── data.txt / output.txt
└── docs/
    ├── architecture.md
    ├── config-spec.md
    ├── api-reference.md
    ├── inheritance-examples.md
    └── inheritance-examples.en.md
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。