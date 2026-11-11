# example-eino

[eino](https://github.com/cloudwego/eino) 学习示例集，通过 OpenAI 兼容端点接入 [DeepSeek](https://api.deepseek.com)。每个示例对应 `agent/` 下的一个文件，跑一个命令学一个知识点。

## 准备

```bash
export DEEPSEEK_API_KEY=sk-xxx   # 必须
```

## 示例清单

| 子命令 | 文件 | 学什么 |
| --- | --- | --- |
| `chatmodel` | `agent/chatmodel.go` | ChatModel 组件直调：`Generate` 阻塞拿全量 vs `Stream` 增量读分片 |
| `chain` | `agent/chain.go` | Chain 编排：ChatTemplate -> ChatModel -> MessageParser 类型串联，`Compile` 成 Runnable |
| `tool` | `agent/tool.go` | Tool Calling 闭环：`InferTool` 结构体推断 schema，手写"调用 -> 执行 -> 回传"循环 |
| `graph` | `agent/graph.go` | Graph 图编排：`Branch` 意图分流，双模板汇合到同一模型，Runnable 复用 |
| `structured` | `agent/structured.go` | 结构化输出：伪工具 + `ToolChoiceForced`，`MessageJSONParser` 解析成结构体 |
| `adk` | `agent/adk_agent.go` | ADK Agent：注册工具后由框架自动跑调用循环 + Runner 事件流消费 |

## 运行

```bash
go run . chatmodel
go run . chain
go run . adk
```

不带参数运行可查看完整用法帮助。

## 目录结构

```
main.go              # 子命令入口
agent/               # 每个示例一个文件
  client.go          #   Client 封装与构造（OpenAI 兼容协议接入 DeepSeek）
```
