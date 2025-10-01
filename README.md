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
- ⚡ **核心组件控制** - 支持核心组件失败时终止整个流程
- 📊 **层级执行** - 层与层之间顺序执行，层内支持多种执行模式
- 🔧 **可扩展** - 易于扩展的组件接口设计

## 🏗️ 架构概览

```
┌─────────────────┐
│   JSON Config   │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐    ┌─────────────────┐
│   DAG Parser    │───▶│   DAG Engine    │
└─────────────────┘    └─────────┬───────┘
                                 │
                                 ▼
                       ┌─────────────────┐
                       │   Layer Exec    │
                       └─────────┬───────┘
                                 │
                                 ▼
                    ┌──────────┬──────────┬──────────┐
                    │  Serial  │ Parallel │  Async   │
                    └──────────┴──────────┴──────────┘
```

## 📦 安装

```bash
go get github.com/yourusername/kflow
```

## 🚀 快速开始

### 1. 定义组件

```go
package main

import (
    "context"
    "fmt"
    "github.com/yourusername/kflow"
)

// 实现 Component 接口
type HelloComponent struct {
    Name string
}

func (h *HelloComponent) Execute(ctx context.Context) error {
    fmt.Printf("Hello from %s\n", h.Name)
    return nil
}

func (h *HelloComponent) GetName() string {
    return h.Name
}

func (h *HelloComponent) IsCore() bool {
    return false
}
```

### 2. 创建 JSON 配置

```json
{
  "name": "hello_workflow",
  "layers": [
    {
      "name": "layer1",
      "execution_mode": "parallel",
      "components": [
        {
          "name": "hello1",
          "type": "hello",
          "is_core": false,
          "config": {
            "message": "Hello World 1"
          }
        },
        {
          "name": "hello2",
          "type": "hello",
          "is_core": true,
          "config": {
            "message": "Hello World 2"
          }
        }
      ]
    },
    {
      "name": "layer2",
      "execution_mode": "serial",
      "components": [
        {
          "name": "hello3",
          "type": "hello",
          "is_core": false,
          "config": {
            "message": "Hello World 3"
          }
        }
      ]
    }
  ]
}
```

### 3. 执行工作流

```go
func main() {
    // 创建 DAG 引擎
    engine := kflow.NewEngine()
    
    // 注册组件工厂
    engine.RegisterComponentFactory("hello", func(config map[string]interface{}) kflow.Component {
        message := config["message"].(string)
        return &HelloComponent{Name: message}
    })
    
    // 从配置文件加载 DAG
    dag, err := engine.LoadFromFile("workflow.json")
    if err != nil {
        panic(err)
    }
    
    // 执行 DAG
    ctx := context.Background()
    if err := engine.Execute(ctx, dag); err != nil {
        fmt.Printf("执行失败: %v\n", err)
    }
}
```

## 📖 执行模式

### 串行执行 (Serial)
组件按顺序依次执行，前一个组件完成后才开始下一个。

### 并行执行 (Parallel)
层内所有组件同时启动执行，等待所有组件完成。

### 异步执行 (Async)
组件异步执行，不等待完成即继续执行下一层。

## 🛡️ 错误处理

- **Recover 机制**: 每个组件都有内置的 panic 恢复机制
- **核心组件**: 标记为核心的组件失败时会终止整个流程
- **非核心组件**: 失败时记录错误但不影响流程继续执行

## 📁 项目结构

```
kflow/
├── README.md
├── go.mod
├── engine.go          # DAG 执行引擎
├── component.go       # 组件接口定义
├── config.go          # 配置文件解析
├── layer.go           # 层执行逻辑
├── examples/          # 使用示例
│   ├── basic/
│   └── advanced/
└── docs/              # 详细文档
    ├── architecture.md
    ├── config-spec.md
    └── api-reference.md
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🔗 相关链接

- [架构设计文档](docs/architecture.md)
- [配置文件规范](docs/config-spec.md)
- [API 参考文档](docs/api-reference.md)
- [使用示例](examples/)

---

如果这个项目对你有帮助，请给个 ⭐️ 支持一下！