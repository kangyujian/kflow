# KFlow API 参考文档

## 概述

本文档详细介绍了 KFlow 框架提供的所有公共 API 接口、类型定义和使用方法。KFlow 采用简洁的接口设计，提供了强大而灵活的 DAG 执行能力。

## 核心接口

### Engine 接口

Engine 是 KFlow 的核心执行引擎，负责 DAG 的加载、验证和执行。

```go
type Engine interface {
    // 注册组件工厂函数
    RegisterComponentFactory(componentType string, factory ComponentFactory)
    
    // 从配置文件加载 DAG
    LoadFromFile(configPath string) (*DAG, error)
    
    // 从配置对象加载 DAG
    LoadFromConfig(config *Config) (*DAG, error)
    
    // 执行 DAG
    Execute(ctx context.Context, dag *DAG) error
    
    // 验证 DAG 配置
    Validate(dag *DAG) error
    
    // 获取执行统计信息
    GetStats() *ExecutionStats
    
    // 设置日志记录器
    SetLogger(logger Logger)
    
    // 设置错误处理器
    SetErrorHandler(handler ErrorHandler)
}
```

#### 方法详解

##### RegisterComponentFactory

注册组件工厂函数，用于创建特定类型的组件。

```go
func (e *Engine) RegisterComponentFactory(componentType string, factory ComponentFactory)
```

**参数:**
- `componentType`: 组件类型名称
- `factory`: 组件工厂函数

**示例:**
```go
engine := kflow.NewEngine()
engine.RegisterComponentFactory("http_client", func(config map[string]interface{}) kflow.Component {
    return &HttpClientComponent{
        URL:     config["url"].(string),
        Method:  config["method"].(string),
        Timeout: time.Duration(config["timeout"].(float64)) * time.Second,
    }
})
```

##### LoadFromFile

从 JSON 配置文件加载 DAG 定义。

```go
func (e *Engine) LoadFromFile(configPath string) (*DAG, error)
```

**参数:**
- `configPath`: 配置文件路径

**返回值:**
- `*DAG`: 加载的 DAG 对象
- `error`: 错误信息

**示例:**
```go
dag, err := engine.LoadFromFile("workflow.json")
if err != nil {
    log.Fatalf("加载配置失败: %v", err)
}
```

##### Execute

执行指定的 DAG。

```go
func (e *Engine) Execute(ctx context.Context, dag *DAG) error
```

**参数:**
- `ctx`: 上下文对象，用于控制执行超时和取消
- `dag`: 要执行的 DAG 对象

**返回值:**
- `error`: 执行错误信息

**示例:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()

if err := engine.Execute(ctx, dag); err != nil {
    log.Printf("执行失败: %v", err)
}
```

### Component 接口

Component 是所有组件必须实现的基础接口。

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

#### 扩展接口

##### InitializableComponent

支持初始化的组件接口。

```go
type InitializableComponent interface {
    Component
    Initialize(config map[string]interface{}) error
}
```

##### CleanupableComponent

支持清理资源的组件接口。

```go
type CleanupableComponent interface {
    Component
    Cleanup() error
}
```

##### HealthCheckableComponent

支持健康检查的组件接口。

```go
type HealthCheckableComponent interface {
    Component
    HealthCheck() error
}
```

##### RetryableComponent

支持自定义重试策略的组件接口。

```go
type RetryableComponent interface {
    Component
    ShouldRetry(err error, attempt int) bool
    GetRetryDelay(attempt int) time.Duration
}
```

### ComponentFactory 类型

组件工厂函数类型定义。

```go
type ComponentFactory func(config map[string]interface{}) Component
```

**示例实现:**
```go
func NewHttpClientFactory() ComponentFactory {
    return func(config map[string]interface{}) Component {
        return &HttpClientComponent{
            URL:    config["url"].(string),
            Method: config["method"].(string),
        }
    }
}
```

## 数据结构

### Config 结构体

配置文件的 Go 结构体表示。

```go
type Config struct {
    Name         string        `json:"name"`
    Description  string        `json:"description,omitempty"`
    Version      string        `json:"version,omitempty"`
    Timeout      Duration      `json:"timeout,omitempty"`
    MaxRetries   int           `json:"max_retries,omitempty"`
    RetryDelay   Duration      `json:"retry_delay,omitempty"`
    Layers       []LayerConfig `json:"layers"`
    GlobalConfig ConfigMap     `json:"global_config,omitempty"`
}
```

### LayerConfig 结构体

层配置的结构体定义。

```go
type LayerConfig struct {
    Name          string            `json:"name"`
    Description   string            `json:"description,omitempty"`
    ExecutionMode ExecutionMode     `json:"execution_mode"`
    Timeout       Duration          `json:"timeout,omitempty"`
    MaxRetries    int               `json:"max_retries,omitempty"`
    RetryDelay    Duration          `json:"retry_delay,omitempty"`
    DependsOn     []string          `json:"depends_on,omitempty"`
    Components    []ComponentConfig `json:"components"`
    LayerConfig   ConfigMap         `json:"layer_config,omitempty"`
}
```

### ComponentConfig 结构体

组件配置的结构体定义。

```go
type ComponentConfig struct {
    Name        string    `json:"name"`
    Type        string    `json:"type"`
    Description string    `json:"description,omitempty"`
    IsCore      bool      `json:"is_core,omitempty"`
    Enabled     bool      `json:"enabled,omitempty"`
    Timeout     Duration  `json:"timeout,omitempty"`
    MaxRetries  int       `json:"max_retries,omitempty"`
    RetryDelay  Duration  `json:"retry_delay,omitempty"`
    DependsOn   []string  `json:"depends_on,omitempty"`
    Config      ConfigMap `json:"config,omitempty"`
    Env         ConfigMap `json:"env,omitempty"`
}
```

### ExecutionMode 枚举

执行模式的枚举定义。

```go
type ExecutionMode string

const (
    ExecutionModeSerial   ExecutionMode = "serial"
    ExecutionModeParallel ExecutionMode = "parallel"
    ExecutionModeAsync    ExecutionMode = "async"
)
```

### DAG 结构体

DAG 运行时表示。

```go
type DAG struct {
    Config *Config
    Layers []*Layer
    Stats  *ExecutionStats
}
```

### Layer 结构体

层的运行时表示。

```go
type Layer struct {
    Config     *LayerConfig
    Components []Component
    Executor   LayerExecutor
    Stats      *LayerStats
}
```

### ExecutionStats 结构体

执行统计信息。

```go
type ExecutionStats struct {
    StartTime     time.Time
    EndTime       time.Time
    Duration      time.Duration
    TotalLayers   int
    CompletedLayers int
    FailedLayers  int
    TotalComponents int
    CompletedComponents int
    FailedComponents int
    Errors        []error
}
```

## 工具函数

### NewEngine

创建新的 Engine 实例。

```go
func NewEngine(options ...EngineOption) *Engine
```

**参数:**
- `options`: 可选的引擎配置选项

**返回值:**
- `*Engine`: 新的引擎实例

**示例:**
```go
engine := kflow.NewEngine(
    kflow.WithLogger(logger),
    kflow.WithTimeout(30*time.Minute),
    kflow.WithMaxRetries(3),
)
```

### EngineOption 类型

引擎配置选项函数类型。

```go
type EngineOption func(*Engine)
```

#### 预定义选项

##### WithLogger

设置日志记录器。

```go
func WithLogger(logger Logger) EngineOption
```

##### WithTimeout

设置默认超时时间。

```go
func WithTimeout(timeout time.Duration) EngineOption
```

##### WithMaxRetries

设置默认最大重试次数。

```go
func WithMaxRetries(maxRetries int) EngineOption
```

##### WithErrorHandler

设置错误处理器。

```go
func WithErrorHandler(handler ErrorHandler) EngineOption
```

## 接口实现示例

### 基础组件实现

```go
type EchoComponent struct {
    name    string
    message string
    isCore  bool
}

func (e *EchoComponent) Execute(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        fmt.Printf("[%s] %s\n", e.name, e.message)
        return nil
    }
}

func (e *EchoComponent) GetName() string {
    return e.name
}

func (e *EchoComponent) IsCore() bool {
    return e.isCore
}
```

### 支持初始化的组件

```go
type DatabaseComponent struct {
    name   string
    db     *sql.DB
    config map[string]interface{}
}

func (d *DatabaseComponent) Initialize(config map[string]interface{}) error {
    d.config = config
    
    dsn := config["dsn"].(string)
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return fmt.Errorf("数据库连接失败: %w", err)
    }
    
    d.db = db
    return nil
}

func (d *DatabaseComponent) Execute(ctx context.Context) error {
    // 执行数据库操作
    return nil
}

func (d *DatabaseComponent) Cleanup() error {
    if d.db != nil {
        return d.db.Close()
    }
    return nil
}

func (d *DatabaseComponent) GetName() string {
    return d.name
}

func (d *DatabaseComponent) IsCore() bool {
    return true
}
```

### 支持重试的组件

```go
type HttpClientComponent struct {
    name string
    url  string
}

func (h *HttpClientComponent) Execute(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", h.url, nil)
    if err != nil {
        return err
    }
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP 请求失败: %d", resp.StatusCode)
    }
    
    return nil
}

func (h *HttpClientComponent) ShouldRetry(err error, attempt int) bool {
    // 网络错误或 5xx 错误才重试
    if attempt >= 3 {
        return false
    }
    
    if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
        return true
    }
    
    return strings.Contains(err.Error(), "5")
}

func (h *HttpClientComponent) GetRetryDelay(attempt int) time.Duration {
    // 指数退避
    return time.Duration(attempt*attempt) * time.Second
}

func (h *HttpClientComponent) GetName() string {
    return h.name
}

func (h *HttpClientComponent) IsCore() bool {
    return false
}
```

## 错误处理

### ErrorHandler 接口

```go
type ErrorHandler interface {
    HandleError(component Component, err error) error
    HandlePanic(component Component, recovered interface{}) error
}
```

### 默认错误处理器

```go
type DefaultErrorHandler struct {
    logger Logger
}

func (h *DefaultErrorHandler) HandleError(component Component, err error) error {
    h.logger.Errorf("组件 %s 执行失败: %v", component.GetName(), err)
    
    if component.IsCore() {
        return fmt.Errorf("核心组件失败，终止执行: %w", err)
    }
    
    return nil // 非核心组件失败不终止执行
}

func (h *DefaultErrorHandler) HandlePanic(component Component, recovered interface{}) error {
    err := fmt.Errorf("组件 %s 发生 panic: %v", component.GetName(), recovered)
    return h.HandleError(component, err)
}
```

## 日志接口

### Logger 接口

```go
type Logger interface {
    Debug(args ...interface{})
    Debugf(format string, args ...interface{})
    Info(args ...interface{})
    Infof(format string, args ...interface{})
    Warn(args ...interface{})
    Warnf(format string, args ...interface{})
    Error(args ...interface{})
    Errorf(format string, args ...interface{})
}
```

### 使用标准库日志

```go
type StdLogger struct {
    *log.Logger
}

func (l *StdLogger) Debug(args ...interface{}) {
    l.Logger.Println(append([]interface{}{"[DEBUG]"}, args...)...)
}

func (l *StdLogger) Debugf(format string, args ...interface{}) {
    l.Logger.Printf("[DEBUG] "+format, args...)
}

// ... 其他方法实现
```

## 完整使用示例

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/yourusername/kflow"
)

func main() {
    // 创建引擎
    engine := kflow.NewEngine(
        kflow.WithTimeout(30*time.Minute),
        kflow.WithMaxRetries(3),
    )
    
    // 注册组件工厂
    engine.RegisterComponentFactory("echo", func(config map[string]interface{}) kflow.Component {
        return &EchoComponent{
            name:    config["name"].(string),
            message: config["message"].(string),
            isCore:  config["is_core"].(bool),
        }
    })
    
    // 加载配置
    dag, err := engine.LoadFromFile("workflow.json")
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }
    
    // 验证配置
    if err := engine.Validate(dag); err != nil {
        log.Fatalf("配置验证失败: %v", err)
    }
    
    // 执行 DAG
    ctx := context.Background()
    if err := engine.Execute(ctx, dag); err != nil {
        log.Printf("执行失败: %v", err)
    } else {
        log.Println("执行成功")
    }
    
    // 获取统计信息
    stats := engine.GetStats()
    log.Printf("执行统计: 总耗时 %v, 成功组件 %d, 失败组件 %d", 
        stats.Duration, stats.CompletedComponents, stats.FailedComponents)
}
```

## 最佳实践

### 1. 组件设计
- 保持组件职责单一
- 实现适当的接口（如 InitializableComponent）
- 合理设置核心组件标识
- 支持上下文取消

### 2. 错误处理
- 区分可重试和不可重试的错误
- 提供有意义的错误信息
- 合理使用 panic 恢复机制

### 3. 性能优化
- 避免在组件中执行长时间阻塞操作
- 合理使用并发控制
- 及时释放资源

### 4. 测试
- 为组件编写单元测试
- 使用模拟对象测试复杂依赖
- 测试错误处理逻辑

通过遵循本 API 文档，您可以充分利用 KFlow 框架的强大功能，构建高效、可靠的工作流应用程序。