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
| `adk` | `agent/adk_agent.go` | ADK Agent：工具调用循环 + Runner 事件流消费 |

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
