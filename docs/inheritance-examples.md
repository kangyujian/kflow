# 继承案例指南（KFlow）

本文通过清晰示例展示工作流继承能力：如何在子工作流中对父工作流进行增、删、改与字段覆盖，并解释关键合并规则与执行效果。

## 目标
- 从工作流 A 继承，删除指定层/组件。
- 覆盖字段（如 `mode`、`timeout`、`enabled`）。
- 在已有层中修改或新增组件；新增整层。

## 父工作流 A（workflowA.json）

```json
{
  "name": "workflowA",
  "version": "1.0.0",
  "description": "父工作流 A",
  "global": { "env": "prod" },
  "metadata": { "owner": "team-A" },
  "layers": [
    {
      "name": "prepare",
      "mode": "serial",
      "timeout": 10,
      "enabled": true,
      "components": [
        { "name": "read", "type": "file", "config": {"path": "input.txt"}, "enabled": true },
        { "name": "validate", "type": "validator", "config": {}, "critical": true }
      ]
    },
    {
      "name": "process",
      "mode": "parallel",
      "timeout": 20,
      "components": [
        { "name": "transform", "type": "transform", "config": {} },
        { "name": "analyze", "type": "analyzer", "config": {} }
      ]
    },
    {
      "name": "cleanup",
      "mode": "serial",
      "timeout": 5,
      "components": [ { "name": "clean", "type": "cleanup", "config": {} } ]
    }
  ]
}
```

## 子工作流 B（workflowB.json）— 继承与增删改

```json
{
  "name": "workflowB",
  "extends": "workflowA.json",
  "description": "子工作流 B：继承 A 并修改",
  "metadata": { "owner": "team-B" },
  "layers": [
    { "name": "cleanup", "remove": true },
    {
      "name": "prepare",
      "mode": "parallel",
      "timeout": 15,
      "components": [
        { "name": "read", "type": "file", "config": {"path": "${INPUT_PATH:default.txt}"} },
        { "name": "validate", "type": "validator", "critical": true },
        { "name": "normalize", "type": "normalizer", "config": {}, "enabled": true }
      ]
    },
    {
      "name": "export",
      "mode": "serial",
      "timeout": 30,
      "components": [ { "name": "write", "type": "file", "config": {"path": "out.txt"} } ]
    }
  ]
}
```

## 合并结果（关键点）
- 删除：`cleanup` 层被删除（`remove: true`）。
- 修改：`prepare` 层的 `mode` 从 serial 改为 parallel，`timeout` 从 10 改为 15；组件 `read/validate` 仍在，新增 `normalize`。
- 新增：新增 `export` 层，追加到合并后的层列表。
- 覆盖：根级 `metadata.owner` 变为 `team-B`；`global` 与 `metadata` 进行键级合并（子覆盖父）。
- 布尔字段：`enabled/critical` 仅在子显式为 `true` 时覆盖为 `true`；未设置不影响父值。

## 运行示例（Go）

```go
parser := engine.NewConfigParser()
cfg, err := parser.ParseFile("workflowB.json")
if err != nil { panic(err) }

registry := engine.NewComponentRegistry()
// 注册需要的组件工厂...

eng, err := engine.NewEngine(cfg, registry)
if err != nil { panic(err) }

stats, err := eng.Execute(context.Background(), engine.NewDataContext())
if err != nil { panic(err) }
fmt.Printf("done: %+v\n", stats)
```

## 规则速览
- 根级 `extends` 指定父工作流；解析器自动读取、检测循环依赖并进行合并。
- 层/组件存在即修改；不存在即新增；设置 `remove: true` 即删除。
- `retry`、`timeout`、`dependencies`、`parallel` 等字段采用子覆盖父的策略。
- 更详细的语义与边界详见：
  - 配置规范（ZH）：docs/config-spec.md
  - 配置规范（EN）：docs/config-spec.en.md