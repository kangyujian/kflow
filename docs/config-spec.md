# KFlow JSON 配置文件规范（更新版）

## 概述

KFlow 使用 JSON 格式的配置文件来定义 DAG（有向无环图）的结构和执行策略。本文档与当前引擎实现保持一致的字段与示例。

## 配置文件结构

### 根配置对象

```json
{
  "name": "workflow_name",
  "description": "工作流描述",
  "version": "1.0.0",
  "timeout": 0,
  "layers": [
    // 层配置数组
  ],
  "global": {
    // 全局参数（可选）
  },
  "metadata": {
    // 元数据（可选）
  }
}
```

#### 字段说明

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `name` | string | ✅ | - | 工作流名称，必须唯一 |
| `description` | string | ❌ | "" | 工作流描述信息 |
| `version` | string | ❌ | "1.0.0" | 配置文件版本 |
| `timeout` | number | ❌ | 0 | 整个工作流的超时时间（纳秒，使用 Go 的 time.Duration 解析），例如 5000000000 表示 5s |
| `layers` | array | ✅ | - | 层配置数组 |
| `global` | object | ❌ | {} | 全局参数，传递给所有组件（当前示例未自动注入组件，但可通过自定义逻辑使用） |
| `metadata` | object | ❌ | {} | 元数据，可用于额外说明 |

### 层配置对象

```json
{
  "name": "layer_name",
  "mode": "parallel",
  "timeout": 0,
  "components": [
    // 组件配置数组
  ],
  "dependencies": ["layer1", "layer2"],
  "enabled": true,
  "parallel": 0
}
```

#### 字段说明

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `name` | string | ✅ | - | 层名称，在工作流中必须唯一 |
| `mode` | string | ❌ | serial | 执行模式：serial/parallel/async |
| `timeout` | number | ❌ | 0 | 层执行超时时间（纳秒）|
| `components` | array | ✅ | - | 组件配置数组 |
| `dependencies` | array | ❌ | [] | 依赖的层名称数组，必须指向在当前层之前的层 |
| `enabled` | bool | ❌ | true | 是否启用该层 |
| `parallel` | number | ❌ | 0 | 并行模式的并发度（0 表示不限制） |

### 组件配置对象

```json
{
  "name": "component_name",
  "type": "component_type",
  "enabled": true,
  "timeout": 0,
  "dependencies": ["comp1", "comp2"],
  "config": {
    // 组件特定配置
  },
  "retry": {
    "max_retries": 0,
    "delay": 0,
    "backoff": 1.0
  }
}
```

#### 字段说明

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `name` | string | ✅ | - | 组件名称，在层内必须唯一 |
| `type` | string | ✅ | - | 组件类型，用于组件工厂创建 |
| `enabled` | bool | ❌ | true | 是否启用该组件 |
| `timeout` | number | ❌ | 0 | 组件执行超时时间（纳秒）|
| `dependencies` | array | ❌ | [] | 依赖的组件名称数组（当前实现未强制校验组件级依赖） |
| `config` | object | ❌ | {} | 组件特定配置 |
| `retry` | object | ❌ | null | 组件重试配置，包括最大重试次数、延迟（纳秒）、退避系数 |

## 执行模式详解

- Serial (串行执行)：组件按照定义顺序依次执行
- Parallel (并行执行)：层内所有组件并发执行，等待全部完成
- Async (异步执行)：组件异步执行，不等待完成即进入下一层

## 配置示例

### 完整示例

```json
{
  "name": "data_processing_workflow",
  "description": "数据处理工作流示例",
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

## 备注

- 所有 `timeout`/`delay` 字段以纳秒为单位，兼容 Go `time.Duration` 的 JSON 反序列化方式
- 引擎会设置部分默认值，例如当 `mode` 为空时默认为 `serial`，当 `enabled` 未显式设置时默认为 `true`
- 环境变量替换支持 `${VAR}` 或 `${VAR:default}` 语法，可在 JSON 中使用