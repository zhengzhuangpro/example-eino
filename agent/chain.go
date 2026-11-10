package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// contentParser 实现 schema.MessageParser[string] 接口：
// 把模型的回复 Message 提取成纯文本 string。
// 实际项目中也可以用 schema.NewMessageJSONParser[T] 让模型输出结构化 JSON 并解析成结构体。
type contentParser struct{}

func (p *contentParser) Parse(ctx context.Context, m *schema.Message) (string, error) {
	return m.Content, nil
}

// ChainCompose Chain 编排：eino 的核心特色，组件像积木一样按类型串联。
//
// 单个组件（ChatModel、ChatTemplate…）只做一件事，编排（compose）负责把它们
// 连成流水线。Chain 是最简单的编排形态：线性依次执行。本示例搭建三节点链：
//
//	ChatTemplate          ChatModel           MessageParser
//	map[string]any  -->  []*schema.Message  -->  *schema.Message  -->  string
//	（填充模板变量）      （调用 LLM）          （提取回复正文）
//
// 本示例学什么：
// - prompt.FromMessages 定义消息模板，schema.FString 用 {var} 占位
// - NewChain[输入类型, 输出类型]，AppendXXX 顺序即执行顺序，相邻节点类型自动对齐
// - Compile 把可变的 Chain 编译成不可变的 Runnable（可复用、可并发调用）
// - 模板里的每个变量，调用时都必须出现在入参 map 里，否则运行时报错
func (c *Client) ChainCompose(ctx context.Context) error {
	// 1. ChatTemplate：定义消息模板，运行时用变量填充
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个起名大师，只输出名字本身，不要任何解释。"),
		schema.UserMessage("给一个讲{topic}的技术专栏起一个名字"),
	)

	// 2. 组装 Chain：这里不直接调用 ChatModel，而是作为链上的节点交给编排器调度
	chain := compose.NewChain[map[string]any, string]()
	chain.
		AppendChatTemplate(template).             // map[string]any -> []*schema.Message
		AppendChatModel(c.chatModel).             // []*schema.Message -> *schema.Message
		AppendLambda(compose.MessageParser[string]( // *schema.Message -> string
			&contentParser{},
		))

	// 3. Compile：编译成不可变的 Runnable
	runnable, err := chain.Compile(ctx)
	if err != nil {
		return err
	}

	// 4. Invoke：整条链一次跑完。入参只需模板变量，出参直接是 string
	name, err := runnable.Invoke(ctx, map[string]any{"topic": "Go 泛型"})
	if err != nil {
		return err
	}
	fmt.Printf("专栏名: %s\n", name)
	return nil
}
