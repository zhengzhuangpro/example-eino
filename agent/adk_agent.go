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
// 与 tool 示例对照：那里手写了"模型 <-> 工具"循环，这里的 ChatModelAgent
// 内置了这个循环——模型决定调哪个工具、拿到结果后继续推理，直到给出最终答复。
// Runner 负责执行并输出事件流。
//
// 本示例学什么：
// - adk.NewChatModelAgent：Name/Description/Instruction + Model + 工具列表
// - ToolsConfig 注册工具：tool 示例的 get_current_time 直接复用，不用手写循环
// - adk.NewRunner 执行 Agent，EnableStreaming 开启事件流
// - 事件流的消费方式：Next() 迭代 + MessageOutput.GetMessage() 取消息
func (c *Client) ADKAgent(ctx context.Context) error {
	// 1. 创建 ChatModelAgent
	timeTool, err := newTimeTool()
	if err != nil {
		return err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "my-assistant",
		Description: "一个可以使用工具回答问题的助手。",
		Instruction: "你是一个有帮助的助手。请根据可用工具回答用户问题。",
		Model:       c.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				// 注册工具：同一个 get_current_time，调度交给 ADK
				Tools: []tool.BaseTool{timeTool},
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
		schema.UserMessage("现在上海几点了？这个时间适合做什么？"),
	}

	events := runner.Run(ctx, input)
	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		// 事件流里有多种事件，这里只关心模型消息：内容为空的事件（如纯工具调用轮次）跳过
		if msg, err := event.Output.MessageOutput.GetMessage(); err == nil && msg.Content != "" {
			fmt.Printf("Agent: %s\n", msg.Content)
		}
	}
	return nil
}
