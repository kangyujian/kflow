# 数据传递与共享 DataContext（KFlow）

DataContext 是 KFlow 组件之间的数据传递与共享机制，提供并发安全的键值存储接口，支持在串行、并行、异步模式下稳定使用。本文介绍其能力、用法与最佳实践。

## 能力概览
- 并发安全：内部使用 `RWMutex` 保护，适用于并行层执行。
- 易用接口：`Set/Get/GetString/Delete/Has/Snapshot` 常用方法齐备。
- 快照能力：`Snapshot()` 返回当前数据的浅拷贝，便于调试与输出。
- 初始化方式：支持空上下文与带初始值上下文。

## 接口一览

```go
type DataContext interface {
    Set(key string, value interface{})
    Get(key string) (interface{}, bool)
    GetString(key string) (string, bool)
    Delete(key string)
    Has(key string) bool
    Snapshot() map[string]interface{}
}
```

默认实现位于 `engine/datacontext.go`，通过 `NewDataContext()` 与 `NewDataContextWith(initial)` 构造。

## 初始化与基本用法

```go
// 创建空的共享数据上下文
data := engine.NewDataContext()

// 或者带初始数据
data := engine.NewDataContextWith(map[string]interface{}{
    "input_path": "data.txt",
    "retry_count": 0,
})

// 组件读写示例
data.Set("file_data", []byte("hello"))
if v, ok := data.Get("file_data"); ok {
    b := v.([]byte)
    _ = b
}

// 字符串快捷获取
if s, ok := data.GetString("user_id"); ok {
    // 使用 s
}

// 判断与删除
exists := data.Has("transformed")
data.Delete("file_data")

// 快照用于调试或输出（浅拷贝）
snapshot := data.Snapshot()
fmt.Printf("data: %+v\n", snapshot)
```

## 与组件集成

在组件的 `Execute(ctx, data)` 方法中，直接通过 `data` 共享与读取信息：

```go
type TransformComponent struct{ name string }

func (t *TransformComponent) Name() string { return t.name }

func (t *TransformComponent) Execute(ctx context.Context, data engine.DataContext) error {
    v, ok := data.Get("file_data")
    if !ok { return fmt.Errorf("missing file_data") }
    b := v.([]byte)
    // ...transform...
    data.Set("transformed_data", b)
    return nil
}
```

组件之间通过约定的键传递内容（例如：`file_data`、`transformed_data`、`output_path`、`errors`）。

## 并发与注意事项
- 并发安全：所有读写操作都使用锁保护，可在并行层中安全使用。
- 浅拷贝：`Snapshot()` 返回浅拷贝；其中的 `map/slice` 等引用类型仍指向原始对象，必要时需自行深拷贝。
- 类型断言：`Get` 返回 `interface{}`，使用前请进行类型断言并做好容错处理。
- 键命名：建议采用可读的、语义明确的键名，如 `input_path`、`raw_data`、`transformed_data`。
- 数据结构：优先使用可序列化的基本类型与结构，降低调试与输出复杂度。

## 示例键约定（建议）
- `input_path`：输入文件路径（string）
- `file_data`：读取的文件内容（[]byte/string）
- `transformed_data`：转换后的数据（[]byte/string/map）
- `output_path`：输出文件路径（string）
- `errors`：累计错误信息（[]error / []string）

更多实现细节请参考：`engine/datacontext.go` 与 `docs/api-reference.md` 的相关章节。