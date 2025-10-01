# KFlow 架构设计文档

## 概述

KFlow 是一个轻量级的 Go 语言 DAG（有向无环图）执行框架，旨在提供简洁、高效、可扩展的工作流执行能力。本文档详细介绍了 KFlow 的架构设计、核心组件以及设计理念。

## 设计理念

### 1. 简洁性 (Simplicity)
- **最小化 API**: 提供简洁直观的 API 接口
- **零依赖**: 除 Go 标准库外，不依赖第三方库
- **配置驱动**: 通过 JSON 配置文件定义复杂的工作流

### 2. 灵活性 (Flexibility)
- **多种执行模式**: 支持串行、并行、异步执行
- **可插拔组件**: 通过接口设计实现组件的可插拔
- **层级结构**: 支持复杂的层级依赖关系

### 3. 可靠性 (Reliability)
- **错误恢复**: 内置 panic 恢复机制
- **核心组件控制**: 支持关键组件失败时的流程控制
- **状态追踪**: 完整的执行状态追踪和日志记录

## 核心架构

```
┌─────────────────────────────────────────────────────────────┐
│                        KFlow Engine                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Config      │  │ Component   │  │ Execution           │  │
│  │ Parser      │  │ Registry    │  │ Coordinator         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Layer       │  │ Serial      │  │ Parallel            │  │
│  │ Manager     │  │ Executor    │  │ Executor            │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Error       │  │ Context     │  │ Async               │  │
│  │ Handler     │  │ Manager     │  │ Executor            │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 核心组件详解

### 1. Engine (执行引擎)

执行引擎是 KFlow 的核心，负责协调整个 DAG 的执行过程。

**主要职责:**
- DAG 配置解析和验证
- 组件注册和管理
- 执行流程协调
- 错误处理和恢复

**接口设计:**
```go
type Engine interface {
    // 注册组件工厂
    RegisterComponentFactory(componentType string, factory ComponentFactory)
    
    // 从配置文件加载 DAG
    LoadFromFile(configPath string) (*DAG, error)
    
    // 从配置对象加载 DAG
    LoadFromConfig(config *Config) (*DAG, error)
    
    // 执行 DAG
    Execute(ctx context.Context, dag *DAG) error
    
    // 验证 DAG 配置
    Validate(dag *DAG) error
}
```

### 2. Component (组件接口)

组件是 DAG 中的最小执行单元，所有业务逻辑都通过组件实现。

**接口定义:**
```go
type Component interface {
    // 执行组件逻辑
    Execute(ctx context.Context) error
    
    // 获取组件名称
    GetName() string
    
    // 是否为核心组件
    IsCore() bool
}
```

**扩展接口:**
```go
// 支持初始化的组件
type InitializableComponent interface {
    Component
    Initialize(config map[string]interface{}) error
}

// 支持清理的组件
type CleanupableComponent interface {
    Component
    Cleanup() error
}

// 支持健康检查的组件
type HealthCheckableComponent interface {
    Component
    HealthCheck() error
}
```

### 3. Layer (层管理器)

层管理器负责管理 DAG 中的层级结构和执行顺序。

**核心特性:**
- 层间顺序执行
- 层内支持多种执行模式
- 依赖关系验证
- 执行状态管理

**执行模式:**

#### 串行执行 (Serial)
```go
func (l *Layer) executeSerial(ctx context.Context) error {
    for _, component := range l.Components {
        if err := l.executeComponent(ctx, component); err != nil {
            if component.IsCore() {
                return fmt.Errorf("核心组件 %s 执行失败: %w", component.GetName(), err)
            }
            // 非核心组件失败，记录错误但继续执行
            l.logger.Errorf("组件 %s 执行失败: %v", component.GetName(), err)
        }
    }
    return nil
}
```

#### 并行执行 (Parallel)
```go
func (l *Layer) executeParallel(ctx context.Context) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(l.Components))
    
    for _, component := range l.Components {
        wg.Add(1)
        go func(comp Component) {
            defer wg.Done()
            if err := l.executeComponent(ctx, comp); err != nil {
                errChan <- err
            }
        }(component)
    }
    
    wg.Wait()
    close(errChan)
    
    // 处理错误
    return l.handleErrors(errChan)
}
```

#### 异步执行 (Async)
```go
func (l *Layer) executeAsync(ctx context.Context) error {
    for _, component := range l.Components {
        go func(comp Component) {
            if err := l.executeComponent(ctx, comp); err != nil {
                l.logger.Errorf("异步组件 %s 执行失败: %v", comp.GetName(), err)
            }
        }(component)
    }
    return nil
}
```

### 4. Config Parser (配置解析器)

配置解析器负责解析 JSON 配置文件，构建 DAG 结构。

**配置结构:**
```go
type Config struct {
    Name        string        `json:"name"`
    Description string        `json:"description,omitempty"`
    Timeout     time.Duration `json:"timeout,omitempty"`
    Layers      []LayerConfig `json:"layers"`
}

type LayerConfig struct {
    Name          string            `json:"name"`
    ExecutionMode ExecutionMode     `json:"execution_mode"`
    Timeout       time.Duration     `json:"timeout,omitempty"`
    Components    []ComponentConfig `json:"components"`
}

type ComponentConfig struct {
    Name   string                 `json:"name"`
    Type   string                 `json:"type"`
    IsCore bool                   `json:"is_core"`
    Config map[string]interface{} `json:"config,omitempty"`
}
```

### 5. Error Handler (错误处理器)

错误处理器提供统一的错误处理和恢复机制。

**核心功能:**
- Panic 恢复
- 错误分类和处理
- 核心组件失败处理
- 错误日志记录

```go
type ErrorHandler struct {
    logger Logger
}

func (eh *ErrorHandler) HandleComponentError(component Component, err error) error {
    // 记录错误
    eh.logger.Errorf("组件 %s 执行失败: %v", component.GetName(), err)
    
    // 如果是核心组件，返回错误终止流程
    if component.IsCore() {
        return fmt.Errorf("核心组件失败，终止执行: %w", err)
    }
    
    // 非核心组件，记录但不终止
    return nil
}

func (eh *ErrorHandler) RecoverFromPanic(component Component) {
    if r := recover(); r != nil {
        err := fmt.Errorf("组件 %s 发生 panic: %v", component.GetName(), r)
        eh.HandleComponentError(component, err)
    }
}
```

## 执行流程

### 1. 初始化阶段
```
加载配置 → 解析 DAG → 验证配置 → 注册组件 → 构建执行计划
```

### 2. 执行阶段
```
遍历层级 → 选择执行模式 → 执行组件 → 错误处理 → 状态更新
```

### 3. 清理阶段
```
执行清理 → 资源释放 → 状态报告 → 日志记录
```

## 扩展性设计

### 1. 组件扩展
通过实现 `Component` 接口，可以轻松添加新的组件类型：

```go
type CustomComponent struct {
    name   string
    config map[string]interface{}
}

func (c *CustomComponent) Execute(ctx context.Context) error {
    // 自定义执行逻辑
    return nil
}

func (c *CustomComponent) GetName() string {
    return c.name
}

func (c *CustomComponent) IsCore() bool {
    return false
}
```

### 2. 执行器扩展
可以通过实现新的执行器来支持更多执行模式：

```go
type CustomExecutor struct{}

func (e *CustomExecutor) Execute(ctx context.Context, components []Component) error {
    // 自定义执行逻辑
    return nil
}
```

### 3. 中间件支持
支持在组件执行前后添加中间件：

```go
type Middleware interface {
    Before(ctx context.Context, component Component) error
    After(ctx context.Context, component Component, err error) error
}
```

## 性能考虑

### 1. 内存管理
- 使用对象池减少内存分配
- 及时释放不再使用的资源
- 避免内存泄漏

### 2. 并发控制
- 合理控制并发度
- 使用 context 进行超时控制
- 避免 goroutine 泄漏

### 3. 错误处理
- 快速失败机制
- 错误聚合和批处理
- 避免错误传播导致的性能问题

## 安全考虑

### 1. 输入验证
- 严格验证配置文件格式
- 防止恶意配置导致的安全问题
- 限制资源使用

### 2. 权限控制
- 组件执行权限控制
- 资源访问权限验证
- 敏感信息保护

### 3. 错误信息
- 避免敏感信息泄露
- 错误信息脱敏处理
- 安全的错误日志记录

## 总结

KFlow 的架构设计遵循简洁、灵活、可靠的原则，通过模块化的设计实现了高度的可扩展性和可维护性。框架的核心在于提供一个稳定、高效的 DAG 执行环境，同时保持 API 的简洁性和易用性。

通过合理的抽象和接口设计，KFlow 能够满足各种复杂的工作流执行需求，同时为未来的功能扩展预留了充分的空间。