package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// CommentAnalysis 是期望模型返回的结构化结果：字段与 tag 描述会进 JSON Schema，
// 模型被约束只能往这些字段里填值。
type CommentAnalysis struct {
	Sentiment string   `json:"sentiment" jsonschema_description:"情感倾向，只能是 positive、negative、neutral 之一"`
	Score     float64  `json:"score" jsonschema_description:"情感强度 0-10，10 最强烈"`
	Keywords  []string `json:"keywords" jsonschema_description:"1-3 个最能代表这条评论的关键词"`
}

// StructuredOutput 结构化输出：让模型稳定返回"能直接解析成 Go 结构体"的 JSON。
//
// 直接让模型"输出 JSON"并不可靠：回复里常夹带解释文字、字段名漂移、漏字段。
// 可靠的做法是借用 Tool Calling 通道——注册一个只有参数 schema 的"伪工具"，
// 再用 ToolChoiceForced 强制模型调用它：模型只能把结果填进参数槽，
// Arguments 必然是符合 schema 的 JSON 字符串，本地再解析成结构体。
//
//	结构体 --GoStruct2ToolInfo--> schema --WithTools+Forced--> 模型
//	      <--NewMessageJSONParser(ParseFrom: tool_call)<-- ToolCalls[0].Arguments
//
// 本示例学什么：
//   - utils.GoStruct2ToolInfo：只要 schema、不绑可执行函数（对照 tool 示例的
//     InferTool = schema + 可执行函数；结构化输出恰好只需要前一半）
//   - model.WithToolChoice(ToolChoiceForced, 工具名)：强制模型必须调用指定工具，
//     这就把"自由发挥的回复"变成了"按 schema 填槽"
//   - schema.NewMessageJSONParser[T]：指定 ParseFrom: tool_call，
//     直接从 ToolCalls 的 Arguments JSON 解析出结构体（chain 示例注释里的伏笔）
//   - openai.WithExtraFields：厂商特有参数的逃生口。DeepSeek 的 thinking 模式
//     与 forced tool_choice 互斥（直接用会报 400 "Thinking mode does not support
//     this tool_choice"），这里通过请求体加 thinking.type=disabled 单次关闭
//   - 同一套代码跑两条评论：不同输入稳定产出同构结果，这就是可编程的输出
func (c *Client) StructuredOutput(ctx context.Context) error {
	// 1. 结构体 → 参数 schema：这一步不需要提供任何函数实现
	toolInfo, err := utils.GoStruct2ToolInfo[CommentAnalysis](
		"analyze_comment",
		"分析一条用户评论的情感倾向",
	)
	if err != nil {
		return err
	}

	parser := schema.NewMessageJSONParser[CommentAnalysis](
		&schema.MessageJSONParseConfig{ParseFrom: schema.MessageParseFromToolCall},
	)

	comments := []string{
		"物流快得离谱，包装也扎实，第二次回购了。",
		"用了一周就坏了，客服踢皮球，气死我了。",
	}
	for _, comment := range comments {
		messages := []*schema.Message{
			schema.SystemMessage("你是评论分析器，必须且只能调用 analyze_comment 工具输出结果。"),
			schema.UserMessage(comment),
		}

		// 2. 强制调用伪工具：WithTools 给 schema，WithToolChoice 锁死必须调它。
		//    DeepSeek 的 thinking 模式不接受 forced tool_choice（400），
		//    用 ExtraFields 在这一请求上关闭 thinking
		reply, err := c.chatModel.Generate(ctx, messages,
			model.WithTools([]*schema.ToolInfo{toolInfo}),
			model.WithToolChoice(schema.ToolChoiceForced, "analyze_comment"),
			openai.WithExtraFields(map[string]any{
				"thinking": map[string]any{"type": "disabled"},
			}),
		)
		if err != nil {
			return err
		}

		// 3. 从 ToolCalls 的 Arguments 解析成结构体（普通回复没有这个通道）
		analysis, err := parser.Parse(ctx, reply)
		if err != nil {
			return err
		}

		fmt.Printf("评论: %s\n→ 情感=%s 强度=%.1f 关键词=%v\n\n",
			comment, analysis.Sentiment, analysis.Score, analysis.Keywords)
	}
	return nil
}
