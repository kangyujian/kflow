# KFlow API 参考（更新版）

本文档与当前代码实现保持一致，概述核心类型、接口与使用方式。示例与字段名称均以引擎源码为准。

## 核心模块与文件
- 配置与解析：<mcfile name="config.go" path="/Users/kangyujian/goProject/kflow/engine/config.go"></mcfile>
- 组件接口与注册表：<mcfile name="component.go" path="/Users/kangyujian/goProject/kflow/engine/component.go"></mcfile>
- 层执行逻辑：<mcfile name="layer.go" path="/Users/kangyujian/goProject/kflow/engine/layer.go"></mcfile>
- 引擎与执行统计：<mcfile name="engine.go" path="/Users/kangyujian/goProject/kflow/engine/engine.go"></mcfile>

## 配置结构体

```go
// 根配置
type Config struct {
    Name        string                 `json:"name"`
    Version     string                 `json:"version"`
    Description string                 `json:"description"`
    Layers      []LayerConfig          `json:"layers"`
    Global      map[string]interface{} `json:"global,omitempty"`
    Timeout     time.Duration          `json:"timeout,omitempty"`
    Metadata    map[string]string      `json:"metadata,omitempty"`
}

// 层配置
type LayerConfig struct {
    Name         string            `json:"name"`
    Mode         ExecutionMode     `json:"mode"`           // serial/parallel/async
    Components   []ComponentConfig `json:"components"`
    Timeout      time.Duration     `json:"timeout"`
    Dependencies []string          `json:"dependencies"`
    Enabled      bool              `json:"enabled"`
    Parallel     int               `json:"parallel,omitempty"` // 并行度上限
}

// 组件配置
type ComponentConfig struct {
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Config       map[string]interface{} `json:"config"`
    Dependencies []string               `json:"dependencies"`
    Timeout      time.Duration          `json:"timeout"`
    Retry        *RetryConfig           `json:"retry,omitempty"`
    Critical     bool                   `json:"critical"`
    Enabled      bool                   `json:"enabled"`
}

// 重试配置
type RetryConfig struct {
    MaxRetries int           `json:"max_retries"`
    Delay      time.Duration `json:"delay"`
    Backoff    float64       `json:"backoff"`
}
```

- 执行模式：`serial` / `parallel` / `async`。
- 默认值：未显式设置时，层与组件的 `enabled` 默认 true；组件 `timeout` 默认 30s；层 `mode` 默认 serial。
- 环境变量替换：支持 `${VAR}` 与 `${VAR:default}`（详见 <mcfile name="config.go" path="/Users/kangyujian/goProject/kflow/engine/config.go"></mcfile>）。

## 组件接口

```go
// 基础组件接口
type Component interface {
    Name() string
    Execute(ctx context.Context, data DataContext) error
}

// 可选扩展接口
type InitializableComponent interface { Initialize(ctx context.Context) error }
type CleanupComponent      interface { Cleanup(ctx context.Context) error }

type RetryableComponent interface {
    Component
    ShouldRetry(err error) bool
    GetRetryConfig() RetryConfig
}

type ValidatableComponent interface {
    Component
    Validate() error
}
```

共享数据通过 `DataContext` 传递，提供并发安全的 `Set/Get/GetString/Delete/Has/Snapshot` 接口，各组件通过键读写（例如 `file_data`、`transformed_data`）。

## 组件工厂与注册表

```go
// 工厂接口
type ComponentFactory interface {
    Create(config ComponentConfig) (Component, error)
    GetType() string
}

// 注册表
type ComponentRegistry struct {
    factories map[string]ComponentFactory
}

func NewComponentRegistry() *ComponentRegistry
func (r *ComponentRegistry) Register(factory ComponentFactory)
func (r *ComponentRegistry) Create(config ComponentConfig) (Component, error)
func (r *ComponentRegistry) GetRegisteredTypes() []string
```

## 引擎 API

```go
// 创建与执行
func NewEngine(cfg *Config, registry *ComponentRegistry, options ...EngineOption) (*Engine, error)
func (e *Engine) Execute(ctx context.Context, data DataContext) (*ExecutionStats, error)

// 查询与校验
func (e *Engine) GetConfig() *Config
func (e *Engine) Validate() error
func (e *Engine) GetLayers() []*Layer
func (e *Engine) GetLayer(name string) (*Layer, bool)
```

- EngineOption：支持 WithLogger、WithErrorHandler、WithMiddleware。
- 执行统计 `ExecutionStats`：含总时长、层统计、成功/失败标识与错误。

## 层执行与关键组件
- Serial：按顺序执行；遇到错误立即返回。
- Parallel：使用信号量限制并发度（`parallel`）；收集错误，若遇到关键组件错误（`Critical`）则中止并返回。
- Async：启动后不等待完成直接进入下一层。

关键组件由组件配置的 `critical` 字段指定；执行失败会产生 `CriticalComponentError`，终止当前层或整体流程。

## 错误类型（示例）
- ConfigError：配置解析/校验错误
- ComponentError：组件创建/初始化/清理错误
- ExecutionError：组件执行失败的统一封装
- CriticalComponentError：关键组件失败错误
- RetryExhaustedError：重试耗尽错误

## 使用示例（简化）

```go
parser := engine.NewConfigParser()
cfg, _ := parser.ParseFile("workflow.json")
registry := engine.NewComponentRegistry()
registry.Register(&yourFactory{})
eng, _ := engine.NewEngine(cfg, registry)
data := NewDataContext()
stats, err := eng.Execute(context.Background(), data)
```

更多示例请参考 <mcfile name="README.md" path="/Users/kangyujian/goProject/kflow/README.md"></mcfile> 与 <mcfile name="example/basic/README.md" path="/Users/kangyujian/goProject/kflow/example/basic/README.md"></mcfile>。