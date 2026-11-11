package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// MultiAgent 多智能体协作：AgentTool 委托模式（官方推荐）。
//
// 拓扑：前台 Agent 把两位专家包成工具，按问题类型委托：
//
//	reception（前台，它的全部工具就是两个"专家工具"）
//	  ├─ go_expert：  Go 技术专家（纯人设子 Agent）
//	  └─ time_expert：时间专家（子 Agent 自己挂 get_current_time 工具）
//
// 本示例学什么：
//   - adk.NewAgentTool：把子 Agent 包成普通工具。子 Agent 的 Name/Description
//     直接成为工具名/描述（父模型靠它判断何时委托），默认入参是
//     {"request": string}——父模型把要委托的问题写进 request
//   - 委托即工具调用：父 Agent 工具循环里"执行工具"= 完整跑一遍子 Agent，
//     子 Agent 的最终答复作为工具结果回传，父 Agent 汇总后答复用户——
//     与 tool 示例的手写循环是同一个机制，只是"工具的实现"换成了另一个 Agent
//   - 子 Agent 也能有自己的工具：time_expert 内部还会调 get_current_time，
//     工具循环可以嵌套
//   - 对照：adk.SetSubAgents 的 transfer 模式官方标注 NOT RECOMMENDED
//     （Agent 间全量共享上下文实证效果不佳），委托模式用摘要式沟通更稳
func (c *Client) MultiAgent(ctx context.Context) error {
	// 1. 子 Agent 一：Go 技术专家（纯人设，展示最小形态）
	goExpert, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "go_expert",
		Description: "Go 语言与技术架构专家。涉及 Go 编程、性能、报错排查等技术问题时委托他。",
		Instruction: "你是资深 Go 工程师，回答控制在三句话以内，给可操作的建议。",
		Model:       c.chatModel,
	})
	if err != nil {
		return err
	}

	// 2. 子 Agent 二：时间专家（自带工具，展示工具循环嵌套）
	timeTool, err := newTimeTool()
	if err != nil {
		return err
	}
	timeExpert, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "time_expert",
		Description: "时间查询专家。涉及当前时间、日期、时区的问题委托他。",
		Instruction: "你负责回答时间问题，必须使用工具获取真实时间，不要猜测。",
		Model:       c.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{timeTool}, // 复用 tool 示例的 get_current_time
			},
		},
	})
	if err != nil {
		return err
	}

	// 3. 父 Agent：前台。注意两点：
	//    a. 它的工具列表里没有普通工具，只有 NewAgentTool 包出来的两个"专家"
	//    b. Instruction 不点名具体专家——模型路由依据是各工具（子 Agent）的
	//       Description，实测不点名也路由正确；Instruction 只写人设与输出规范
	reception, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reception",
		Description: "客服前台，负责把问题分流给对应专家。",
		Instruction: "你是客服前台。收到问题后委托合适的专家，拿到专家答复后用一两句话转述给用户。",
		Model:       c.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					adk.NewAgentTool(ctx, goExpert),
					adk.NewAgentTool(ctx, timeExpert),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// 4. Runner 执行父 Agent（事件流消费与 adk 示例同款）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: reception})

	questions := []string{
		"我的 Go 程序内存占用越来越高，可能是什么原因？", // → 委托 go_expert
		"现在上海几点了？", // → 委托 time_expert（内部再调工具）
	}
	for _, q := range questions {
		fmt.Printf("\nQ: %s\n", q)
		events := runner.Run(ctx, []adk.Message{schema.UserMessage(q)})
		for {
			event, ok := events.Next()
			if !ok {
				break
			}
			// 委托模式下事件都归在父 Agent 名下（event.AgentName 不区分内外），
			// 用消息 Role 区分：tool 消息 = 专家工具的返回原文，assistant = 前台答复
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil || msg.Content == "" {
				continue
			}
			if msg.Role == schema.Tool {
				fmt.Printf("  ↳ %s 的答复: %s\n", msg.ToolName, msg.Content)
			} else {
				fmt.Printf("A: %s\n", msg.Content)
			}
		}
	}
	return nil
}
