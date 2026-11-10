package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// timeArgs 是工具的入参结构体：字段与 tag 会被 InferTool 反射成 JSON Schema，
// 模型据此知道有哪些参数、每个参数是什么含义。
type timeArgs struct {
	TZ string `json:"tz" jsonschema_description:"IANA 时区名，例如 Asia/Shanghai、UTC"`
}

// newTimeTool 用 utils.InferTool 定义一个本地工具：
// 参数 schema 从 timeArgs 结构体推断，无需手写 JSON Schema；
// 模型返回的参数 JSON 会自动解码成 timeArgs 再调用函数，返回值自动编码回字符串。
func newTimeTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_current_time",
		"获取指定时区的当前时间。当用户询问现在几点、日期等时间问题时使用。",
		func(ctx context.Context, args timeArgs) (string, error) {
			tz, err := time.LoadLocation(args.TZ)
			if err != nil {
				return "", fmt.Errorf("未知时区 %q，请使用 IANA 名称如 Asia/Shanghai", args.TZ)
			}
			return time.Now().In(tz).Format("2006-01-02 15:04:05 MST"), nil
		},
	)
}

// ToolCalling 工具调用闭环：手写"模型 <-> 工具"循环。
//
// 与 adk 示例对照：ADK 的 ChatModelAgent 内置了下面这个循环，本示例把它摊开
// 手写一遍，看清每一轮发生什么：
//
//	用户提问 --> Generate（带工具列表）--> 模型返回 ToolCalls（想调工具）
//	          --> 本地执行工具 --> 结果包成 ToolMessage 回传
//	          --> 再次 Generate --> …… 直到模型不再要工具，给出最终答复
//
// 本示例学什么：
// - utils.InferTool：结构体反射生成参数 schema，一个 Go 函数就是一个工具
// - model.WithTools：调用时把工具列表交给模型（是每次调用的 Option，不是构造时绑定）
// - reply.ToolCalls 非空表示模型想调工具，Function.Arguments 是模型生成的参数 JSON 字符串
// - 工具结果用 schema.ToolMessage(content, callID) 回传，ID 必须对上，
//   模型才知道这份结果对应哪次调用
// - 消息历史必须完整：模型回复（含 ToolCalls）和工具结果都要 append 进去
func (c *Client) ToolCalling(ctx context.Context) error {
	timeTool, err := newTimeTool()
	if err != nil {
		return err
	}
	toolInfo, err := timeTool.Info(ctx)
	if err != nil {
		return err
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一个简洁的助手，回答控制在两句话以内。需要时间信息时必须使用工具，不要猜测。"),
		schema.UserMessage("现在上海和 UTC 分别是几点？"),
	}

	for round := 1; ; round++ {
		reply, err := c.chatModel.Generate(ctx, messages, model.WithTools([]*schema.ToolInfo{toolInfo}))
		if err != nil {
			return err
		}
		messages = append(messages, reply) // 模型回复（含工具调用请求）也要进历史

		if len(reply.ToolCalls) == 0 { // 不再要工具：这就是最终答复
			fmt.Printf("[第 %d 轮] 模型最终答复:\n%s\n", round, reply.Content)
			return nil
		}

		for _, call := range reply.ToolCalls {
			fmt.Printf("[第 %d 轮] 模型请求调用: %s(%s)\n", round, call.Function.Name, call.Function.Arguments)

			// 本地执行：InvokableRun 内部把 Arguments JSON 解码成 timeArgs 再调函数
			result, err := timeTool.InvokableRun(ctx, call.Function.Arguments)
			if err != nil {
				// 工具出错也回传给模型：它可以向用户解释，或换个方式重试
				result = fmt.Sprintf("工具执行出错: %v", err)
			}
			fmt.Printf("[第 %d 轮] 工具返回: %s\n\n", round, result)

			messages = append(messages, schema.ToolMessage(result, call.ID))
		}
	}
}
