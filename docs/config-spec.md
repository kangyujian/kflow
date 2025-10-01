# KFlow JSON 配置文件规范

## 概述

KFlow 使用 JSON 格式的配置文件来定义 DAG（有向无环图）的结构和执行策略。本文档详细说明了配置文件的格式、字段含义以及使用示例。

## 配置文件结构

### 根配置对象

```json
{
  "name": "workflow_name",
  "description": "工作流描述",
  "version": "1.0.0",
  "timeout": "30m",
  "max_retries": 3,
  "retry_delay": "5s",
  "layers": [
    // 层配置数组
  ],
  "global_config": {
    // 全局配置
  }
}
```

#### 字段说明

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `name` | string | ✅ | - | 工作流名称，必须唯一 |
| `description` | string | ❌ | "" | 工作流描述信息 |
| `version` | string | ❌ | "1.0.0" | 配置文件版本 |
| `timeout` | string | ❌ | "0" | 整个工作流的超时时间，格式如 "30m", "1h" |
| `max_retries` | int | ❌ | 0 | 全局最大重试次数 |
| `retry_delay` | string | ❌ | "1s" | 重试间隔时间 |
| `layers` | array | ✅ | - | 层配置数组 |
| `global_config` | object | ❌ | {} | 全局配置，会传递给所有组件 |

### 层配置对象

```json
{
  "name": "layer_name",
  "description": "层描述",
  "execution_mode": "parallel",
  "timeout": "10m",
  "max_retries": 2,
  "retry_delay": "3s",
  "depends_on": ["layer1", "layer2"],
  "components": [
    // 组件配置数组
  ],
  "layer_config": {
    // 层级配置
  }
}
```

#### 字段说明

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `name` | string | ✅ | - | 层名称，在工作流中必须唯一 |
| `description` | string | ❌ | "" | 层描述信息 |
| `execution_mode` | string | ✅ | - | 执行模式：serial/parallel/async |
| `timeout` | string | ❌ | 继承全局 | 层执行超时时间 |
| `max_retries` | int | ❌ | 继承全局 | 层最大重试次数 |
| `retry_delay` | string | ❌ | 继承全局 | 层重试间隔时间 |
| `depends_on` | array | ❌ | [] | 依赖的层名称数组 |
| `components` | array | ✅ | - | 组件配置数组 |
| `layer_config` | object | ❌ | {} | 层级配置，会传递给层内所有组件 |

### 组件配置对象

```json
{
  "name": "component_name",
  "type": "component_type",
  "description": "组件描述",
  "is_core": true,
  "enabled": true,
  "timeout": "5m",
  "max_retries": 1,
  "retry_delay": "2s",
  "depends_on": ["comp1", "comp2"],
  "config": {
    // 组件特定配置
  },
  "env": {
    // 环境变量
  }
}
```

#### 字段说明

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `name` | string | ✅ | - | 组件名称，在层内必须唯一 |
| `type` | string | ✅ | - | 组件类型，用于组件工厂创建 |
| `description` | string | ❌ | "" | 组件描述信息 |
| `is_core` | bool | ❌ | false | 是否为核心组件，核心组件失败会终止流程 |
| `enabled` | bool | ❌ | true | 是否启用该组件 |
| `timeout` | string | ❌ | 继承层级 | 组件执行超时时间 |
| `max_retries` | int | ❌ | 继承层级 | 组件最大重试次数 |
| `retry_delay` | string | ❌ | 继承层级 | 组件重试间隔时间 |
| `depends_on` | array | ❌ | [] | 依赖的组件名称数组（仅在 serial 模式下有效） |
| `config` | object | ❌ | {} | 组件特定配置 |
| `env` | object | ❌ | {} | 组件环境变量 |

## 执行模式详解

### 1. Serial (串行执行)

组件按照定义顺序或依赖关系依次执行。

```json
{
  "name": "serial_layer",
  "execution_mode": "serial",
  "components": [
    {
      "name": "step1",
      "type": "http_request",
      "config": {
        "url": "https://api.example.com/step1"
      }
    },
    {
      "name": "step2",
      "type": "data_process",
      "depends_on": ["step1"],
      "config": {
        "input_from": "step1"
      }
    },
    {
      "name": "step3",
      "type": "notification",
      "depends_on": ["step2"],
      "config": {
        "message": "处理完成"
      }
    }
  ]
}
```

**特点:**
- 组件按顺序执行
- 支持显式依赖关系
- 前一个组件完成后才开始下一个
- 任何组件失败都会影响后续执行

### 2. Parallel (并行执行)

层内所有组件同时启动，等待所有组件完成。

```json
{
  "name": "parallel_layer",
  "execution_mode": "parallel",
  "components": [
    {
      "name": "task1",
      "type": "data_fetch",
      "config": {
        "source": "database1"
      }
    },
    {
      "name": "task2",
      "type": "data_fetch",
      "config": {
        "source": "database2"
      }
    },
    {
      "name": "task3",
      "type": "api_call",
      "config": {
        "endpoint": "/api/v1/data"
      }
    }
  ]
}
```

**特点:**
- 所有组件同时启动
- 等待所有组件完成才进入下一层
- 提高执行效率
- 核心组件失败会终止整个层的执行

### 3. Async (异步执行)

组件异步执行，不等待完成即继续下一层。

```json
{
  "name": "async_layer",
  "execution_mode": "async",
  "components": [
    {
      "name": "log_writer",
      "type": "log_component",
      "is_core": false,
      "config": {
        "log_level": "info"
      }
    },
    {
      "name": "metrics_collector",
      "type": "metrics_component",
      "is_core": false,
      "config": {
        "interval": "30s"
      }
    }
  ]
}
```

**特点:**
- 组件异步执行
- 不等待完成即进入下一层
- 适用于日志、监控等非关键任务
- 通常设置为非核心组件

## 配置示例

### 完整示例

```json
{
  "name": "data_processing_workflow",
  "description": "数据处理工作流示例",
  "version": "1.0.0",
  "timeout": "1h",
  "max_retries": 2,
  "retry_delay": "10s",
  "global_config": {
    "log_level": "info",
    "environment": "production"
  },
  "layers": [
    {
      "name": "initialization",
      "description": "初始化层",
      "execution_mode": "serial",
      "timeout": "5m",
      "components": [
        {
          "name": "config_loader",
          "type": "config_component",
          "is_core": true,
          "config": {
            "config_path": "/etc/app/config.yaml"
          }
        },
        {
          "name": "db_connection",
          "type": "database_component",
          "is_core": true,
          "depends_on": ["config_loader"],
          "config": {
            "connection_string": "${DB_CONNECTION_STRING}",
            "max_connections": 10
          }
        }
      ]
    },
    {
      "name": "data_collection",
      "description": "数据收集层",
      "execution_mode": "parallel",
      "timeout": "15m",
      "depends_on": ["initialization"],
      "components": [
        {
          "name": "api_collector",
          "type": "http_collector",
          "is_core": true,
          "config": {
            "endpoints": [
              "https://api1.example.com/data",
              "https://api2.example.com/data"
            ],
            "timeout": "30s"
          }
        },
        {
          "name": "file_collector",
          "type": "file_collector",
          "is_core": false,
          "config": {
            "input_dir": "/data/input",
            "file_pattern": "*.json"
          }
        },
        {
          "name": "queue_collector",
          "type": "queue_collector",
          "is_core": true,
          "config": {
            "queue_name": "data_queue",
            "batch_size": 100
          }
        }
      ]
    },
    {
      "name": "data_processing",
      "description": "数据处理层",
      "execution_mode": "serial",
      "timeout": "30m",
      "depends_on": ["data_collection"],
      "components": [
        {
          "name": "data_validator",
          "type": "validator_component",
          "is_core": true,
          "config": {
            "schema_path": "/schemas/data.json",
            "strict_mode": true
          }
        },
        {
          "name": "data_transformer",
          "type": "transformer_component",
          "is_core": true,
          "depends_on": ["data_validator"],
          "config": {
            "transformation_rules": "/rules/transform.yaml"
          }
        },
        {
          "name": "data_enricher",
          "type": "enricher_component",
          "is_core": false,
          "depends_on": ["data_transformer"],
          "config": {
            "enrichment_api": "https://enrich.example.com/api"
          }
        }
      ]
    },
    {
      "name": "data_output",
      "description": "数据输出层",
      "execution_mode": "parallel",
      "timeout": "10m",
      "depends_on": ["data_processing"],
      "components": [
        {
          "name": "database_writer",
          "type": "db_writer",
          "is_core": true,
          "config": {
            "table_name": "processed_data",
            "batch_size": 1000
          }
        },
        {
          "name": "file_writer",
          "type": "file_writer",
          "is_core": false,
          "config": {
            "output_dir": "/data/output",
            "file_format": "parquet"
          }
        },
        {
          "name": "api_publisher",
          "type": "http_publisher",
          "is_core": false,
          "config": {
            "webhook_url": "https://webhook.example.com/data"
          }
        }
      ]
    },
    {
      "name": "monitoring",
      "description": "监控层",
      "execution_mode": "async",
      "components": [
        {
          "name": "metrics_reporter",
          "type": "metrics_component",
          "is_core": false,
          "config": {
            "metrics_endpoint": "https://metrics.example.com/api"
          }
        },
        {
          "name": "log_aggregator",
          "type": "log_component",
          "is_core": false,
          "config": {
            "log_endpoint": "https://logs.example.com/api"
          }
        }
      ]
    }
  ]
}
```

## 配置验证规则

### 1. 基本验证
- 所有必需字段必须存在
- 字段类型必须正确
- 名称必须唯一（在相应作用域内）

### 2. 依赖关系验证
- 依赖的层/组件必须存在
- 不能存在循环依赖
- 依赖关系必须在同一层内（组件依赖）

### 3. 执行模式验证
- execution_mode 必须是有效值
- async 模式下的组件通常应该是非核心组件
- serial 模式下的依赖关系必须形成有效的执行顺序

### 4. 超时时间验证
- 超时时间格式必须正确（如 "30s", "5m", "1h"）
- 子级超时时间不应超过父级
- 超时时间必须为正数

## 最佳实践

### 1. 命名规范
- 使用有意义的名称
- 采用一致的命名风格（如 snake_case）
- 避免使用保留字或特殊字符

### 2. 层级设计
- 合理划分层级，每层职责单一
- 控制层内组件数量，避免过于复杂
- 合理使用依赖关系，避免过度耦合

### 3. 错误处理
- 合理设置核心组件标识
- 为关键组件设置适当的重试策略
- 使用超时机制防止长时间阻塞

### 4. 性能优化
- 合理使用并行执行提高效率
- 避免不必要的依赖关系
- 控制并发度，防止资源耗尽

### 5. 配置管理
- 使用环境变量管理敏感信息
- 为不同环境准备不同的配置文件
- 定期验证和更新配置文件

## 环境变量支持

配置文件支持环境变量替换，格式为 `${VARIABLE_NAME}`：

```json
{
  "config": {
    "database_url": "${DATABASE_URL}",
    "api_key": "${API_KEY}",
    "timeout": "${REQUEST_TIMEOUT:30s}"
  }
}
```

支持默认值语法：`${VARIABLE_NAME:default_value}`

## 配置文件模板

### 简单工作流模板

```json
{
  "name": "simple_workflow",
  "layers": [
    {
      "name": "main_layer",
      "execution_mode": "serial",
      "components": [
        {
          "name": "component1",
          "type": "your_component_type",
          "is_core": true,
          "config": {}
        }
      ]
    }
  ]
}
```

### 复杂工作流模板

```json
{
  "name": "complex_workflow",
  "description": "复杂工作流模板",
  "timeout": "1h",
  "max_retries": 3,
  "layers": [
    {
      "name": "init_layer",
      "execution_mode": "serial",
      "components": []
    },
    {
      "name": "process_layer",
      "execution_mode": "parallel",
      "depends_on": ["init_layer"],
      "components": []
    },
    {
      "name": "cleanup_layer",
      "execution_mode": "async",
      "depends_on": ["process_layer"],
      "components": []
    }
  ]
}
```

通过遵循本规范，您可以创建结构清晰、功能完整的 KFlow 工作流配置文件。