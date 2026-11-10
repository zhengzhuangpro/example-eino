package agent

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

// ChatModelBasic 基础调用：不经编排，直接使用 ChatModel 组件。
//
// 本示例学什么：
// - eino 里一切 LLM 交互都经过 model.BaseChatModel 接口：
//     Generate：阻塞直到返回完整回复
//     Stream：  返回 StreamReader，增量读取分片
// - 输入是 []*schema.Message，一段带角色的对话历史；
//   Role 取值：system（设定人设）/ user（用户）/ assistant（模型）/ tool（工具结果）
// - Stream 的 reader 必须 Close（defer），流结束的标志是 io.EOF 而不是 nil
func (c *Client) ChatModelBasic(ctx context.Context) error {
	messages := []*schema.Message{
		schema.SystemMessage("你是一个简洁的 Go 语言助教，回答控制在两句话以内。"),
		schema.UserMessage("用一句话解释什么是 interface"),
	}

	// ---- 方式一：Generate，一次性拿到完整回复 ----
	// 适合需要完整结果再继续的场景：结构化抽取、分类等。
	reply, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return err
	}
	fmt.Printf("[Generate] Role=%s\n[Generate] 回复: %s\n\n", reply.Role, reply.Content)

	// ---- 方式二：Stream，边生成边返回分片 ----
	// 适合聊天界面、长文本生成等"打字机效果"场景。
	fmt.Println("[Stream] 回复: ")
	reader, err := c.chatModel.Stream(ctx, messages)
	if err != nil {
		return err
	}
	defer reader.Close() // 习惯写法：拿到 reader 立刻 defer Close，否则泄漏底层连接

	for {
		chunk, err := reader.Recv()
		if errors.Is(err, io.EOF) {
			break // 流结束
		}
		if err != nil {
			return err
		}
		// 每个分片也是一个 *schema.Message，Content 是本次增量的文本
		fmt.Print(chunk.Content)
	}
	fmt.Println()
	return nil
}
