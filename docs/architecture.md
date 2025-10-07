# KFlow 架构说明（更新版）

本文档阐述当前实现的整体架构与设计理念，并与源码保持一致。

## 设计理念
- 简洁：接口尽量最小化，避免不必要的耦合。
- 灵活：支持串行、并行与异步三种层执行模式，组件可选扩展接口。
- 可靠：关键组件失败会中止流程；支持重试与超时；严格的配置校验。

## 核心组件

1) 引擎（Engine）
- 负责加载配置、构建层与组件、执行工作流并收集统计信息。
- 支持可插拔选项：日志、错误处理器、中间件。
- 关键文件：<mcfile name="engine.go" path="/Users/kangyujian/goProject/kflow/engine/engine.go"></mcfile>

2) 层（Layer）
- 表示一个有序的组件集合，具备执行模式与并行度。
- 校验规则：必须有 name、mode；组件列表非空；依赖必须存在。
- 关键文件：<mcfile name="layer.go" path="/Users/kangyujian/goProject/kflow/engine/layer.go"></mcfile>

3) 组件（Component）
- 基础接口仅包含 `Name()` 与 `Execute(ctx, data)`。
- 可选接口：初始化、清理、重试策略、校验。
- 通过 `ComponentRegistry` 用工厂创建并按 `type` 注册。
- 关键文件：<mcfile name="component.go" path="/Users/kangyujian/goProject/kflow/engine/component.go"></mcfile>

4) 配置与解析
- JSON 配置映射到 `Config`、`LayerConfig`、`ComponentConfig`。
- 默认值设置：`version`、`layer.mode=serial`、`enabled=true`、`component.timeout=30s`。
- 支持环境变量替换与依赖检查。
- 关键文件：<mcfile name="config.go" path="/Users/kangyujian/goProject/kflow/engine/config.go"></mcfile>

## 执行流程

1. 解析配置并校验
- 使用 ConfigParser 解析文件，执行默认值填充与字段校验。
- 根据 `enabled` 字段筛选层与组件。

2. 构建组件与层
- 通过注册表按 `type` 创建组件实例。
- 为每个层构建执行器（Serial/Parallel/Async）。

3. 执行 DAG
- 数据通过并发安全的 `DataContext` 在组件间传递，组件通过键读写。
- Serial：顺序逐个执行，遇错返回。
- Parallel：并发执行，受 `parallel` 限制；收集错误并按 `critical` 决策。
- Async：启动但不等待完成，立即进入下一层。

4. 重试与关键组件
- 组件可声明 `retry` 策略或实现 `RetryableComponent`，由层的 `executeWithRetry` 统一处理。
- `critical=true` 的组件失败会导致层或全局失败。
 - 重试参数：`max_retries`（不含首次尝试，总尝试=1+max_retries）、`delay`（纳秒）、`backoff`（退避系数）。层内重试延迟计算：第 n 次重试延迟 = `delay` × (`backoff` × (n-1))，为线性乘系数。
 - 当 `ShouldRetry(err)` 返回 false 时提前结束重试；耗尽后返回 `RetryExhaustedError`。

5. 统计与日志
- Engine 收集 `ExecutionStats`，每层记录开始/结束、成功/失败与耗时。
- 可通过选项注入自定义 Logger 与错误处理器。

6. 超时控制
- 全局：`Config.timeout` 在引擎开始时设置整体 `context.WithTimeout`。
- 层级：`LayerConfig.timeout` 进入该层时设置新的 `context.WithTimeout`，影响串行/并行/异步执行。
- 组件：`ComponentConfig.timeout`（默认 30s）提供给组件实现参考，当前引擎不为单组件自动创建专属超时；组件应在 `Execute(ctx, data)` 中尊重 `ctx` 取消，并可结合自身的 `timeout` 实现更细粒度控制。

## 配置示例（节选）

```json
{
  "name": "basic_workflow",
  "version": "1.0.0",
  "layers": [
    {
      "name": "data_preparation",
      "mode": "parallel",
      "enabled": true,
      "components": [
        {
          "name": "file_reader",
          "type": "FileReader",
          "enabled": true,
          "timeout": 30000000000,
          "config": {"path": "data.txt"}
        }
      ]
    }
  ]
}
```

- `timeout` 采用纳秒整数（Go 原生 time.Duration 的 JSON 表达）。
- `enabled` 支持布尔或 0/1（解析时统一为布尔）。

## 与旧版本差异
- 移除过时字段：`is_core`、`execution_mode`（改为 `critical` 与 `mode`）。
- 组件接口从 `GetName()/IsCore()/Execute(ctx)` 统一到 `Name()/Execute(ctx, data)`。
- 引擎 API 从全局单例式改为显式创建并传入 `Config` 与 `ComponentRegistry`。

## 扩展与中间件
- 中间件可在执行前后注入逻辑（如监控、追踪、审计）。
- 错误处理器可自定义降级、告警与恢复策略。

## 性能与可靠性建议
- 为并行层设置合理的 `parallel` 值以避免资源争用。
- 组件实现中注意上下文取消与超时处理。
- 对关键组件启用重试策略，使用指数退避控制负载。

更多内容请参考：
- <mcfile name="README.md" path="/Users/kangyujian/goProject/kflow/README.md"></mcfile>
- <mcfile name="api-reference.md" path="/Users/kangyujian/goProject/kflow/docs/api-reference.md"></mcfile>
- <mcfile name="config-spec.md" path="/Users/kangyujian/goProject/kflow/docs/config-spec.md"></mcfile>