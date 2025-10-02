# KFlow 基础示例

这个示例展示了 KFlow 框架的基本用法，包括配置文件定义和组件实现，并演示真实的文件读取与写入。

## 示例结构

- `workflow.json`: 工作流配置文件，定义了三个层级的执行流程
- `components.go`: 组件实现代码，包含文件读取、配置读取、转换、验证、写入、日志
- `main.go`: 主程序，展示如何初始化和执行工作流
- `data.txt`: 示例输入文件，供 FileReaderComponent 读取
- `output.txt`: 示例输出文件，由 FileWriterComponent 写入

## 工作流说明

这个示例工作流包含三个层级：

1. **数据准备层 (data_preparation)**
   - 并行执行模式
   - 包含文件读取和配置读取两个组件

2. **数据处理层 (data_processing)**
   - 串行执行模式
   - 包含数据转换和数据验证两个组件
   - 依赖于数据准备层

3. **数据输出层 (data_output)**
   - 串行执行模式
   - 包含文件写入和日志通知两个组件
   - 依赖于数据处理层

## 运行示例

在项目根目录运行：

```bash
go run ./example/basic
```

或在示例目录运行：

```bash
cd example/basic
go run .
```

> 组件实现已支持在根目录或示例目录运行时自动解析相对路径。

## 文件I/O说明

- FileReaderComponent：使用 `os.ReadFile` 读取 `data.txt` 内容，并将结果写入共享数据 `file_data`
- FileWriterComponent：将 `transformed_data` 写入 `output.txt`，根据 `append` 决定覆盖或追加

## 注意事项

- 验证规则（validator）默认为 `not_empty` 和 `max_length:500`，如需更严格可调整规则或精简 `data.txt`
- 所有组件的 `Execute` 方法签名为 `Execute(ctx context.Context, data map[string]interface{})`

## 核心概念

- **组件 (Component)**: 工作流中的最小执行单元
- **层级 (Layer)**: 组件的逻辑分组，可以设置执行模式和依赖关系
- **执行模式 (Mode)**: 支持串行、并行和异步三种执行模式
- **核心组件 (Core Component)**: 标记为核心的组件失败会导致整个工作流失败