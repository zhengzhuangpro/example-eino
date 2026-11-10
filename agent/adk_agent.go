package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ADKAgent ADK Agent：eino 的高层 Agent 框架，自动完成"模型 <-> 工具"循环。
//
// 与前面两个示例的区别：ChatModelBasic / ChainCompose 是"一次性"调用，
// ADK 的 ChatModelAgent 则内置了工具调用循环——模型决定调哪个工具、
// 拿到结果后继续推理，直到给出最终答复。Runner 负责执行并输出事件流。
//
// 本示例学什么：
// - adk.NewChatModelAgent：Name/Description/Instruction + Model + 工具列表
// - adk.NewRunner 执行 Agent，EnableStreaming 开启事件流
// - 事件流的消费方式：Next() 迭代 + MessageOutput.GetMessage() 取消息
func (c *Client) ADKAgent(ctx context.Context) error {
	// 1. 创建 ChatModelAgent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "my-assistant",
		Description: "一个可以使用工具回答问题的助手。",
		Instruction: "你是一个有帮助的助手。请根据可用工具回答用户问题。",
		Model:       c.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					// 注册你的工具，例如 webSearchTool
				},
			},
		},
		// Handlers: []adk.ChatModelAgentMiddleware{...}, // 注册 Middleware
	})
	if err != nil {
		return err
	}

	// 2. 通过 Runner 执行 Agent
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 3. 发送用户请求并消费事件流
	input := []adk.Message{
		schema.UserMessage("Hello, 请做一个自我介绍."),
	}

	events := runner.Run(ctx, input)
	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		fmt.Println(event)

		if msg, err := event.Output.MessageOutput.GetMessage(); err == nil {
			fmt.Printf("Agent: %s\n", msg.Content)
		}
	}
	return nil
}
