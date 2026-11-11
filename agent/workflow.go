package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// weeklyInput / weeklyOutput 是工作流的输入输出结构体：
// 字段级装配的起点与终点——输出结构体的每个字段来自哪个节点，在连线上显式声明。
type weeklyInput struct {
	Topic string
}

type weeklyOutput struct {
	Title string
	Brief string
}

// WorkflowCompose Workflow 编排：字段级装配 + 天然并行。
//
// 场景：技术周刊生成器——输入一个主题，并行生成标题和摘要，装配成结构体返回。
//
//	weeklyInput.Topic ─MapFields─→ title_tpl → title_model → title_out ─ToField─→ weeklyOutput.Title
//	       └──────────MapFields──→ brief_tpl → brief_model → brief_out ─ToField─→ weeklyOutput.Brief
//
// 与 graph 示例对照：
//   - Graph 的边上流动的是"整个值"，相邻节点类型要人肉对齐；
//     Workflow 改成"字段级装配"：AddInput 声明我从上游哪个字段拿什么、填进哪个入参
//   - Graph 的并行产出要在下游手写 merge 节点；Workflow 用 AddEnd 把多个节点的
//     输出各自装配进输出结构体的字段
//   - Workflow 底层是 AllPredecessor 触发的无环图：互相没有依赖的节点自动并发，
//     title/brief 两条支路不需要（也不能）显式声明并行
func (c *Client) WorkflowCompose(ctx context.Context) error {
	// 1. 两条支路各自的模板（模板写法与 chain 示例相同，重点看装配方式）
	titleTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个起名大师，只输出名字本身，不要任何解释。"),
		schema.UserMessage("给一个讲{topic}的技术专栏起一个名字"))
	briefTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是技术编辑，摘要严格控制在两句话以内。"),
		schema.UserMessage("用一段话概括{topic}最核心的内容"))

	// 2. 取正文的 Lambda：model 输出的 *schema.Message → 纯文本 string。
	//    顺带打印支路完成时间，让"并行"直接可见：两条支路的完成时刻
	//    都落在总耗时以内、互不等待
	start := time.Now()
	newMsgText := func(branch string) *compose.Lambda {
		return compose.InvokableLambda(func(ctx context.Context, m *schema.Message) (string, error) {
			fmt.Printf("[%s 支路完成，第 %v]\n", branch, time.Since(start).Round(time.Millisecond))
			return m.Content, nil
		})
	}

	// 3. 组装：AddInput(上游key, 映射...) 声明数据从哪来、到哪去。
	//    首节点从 START 拿工作流输入；Topic(输入结构体字段) → topic(模板变量)
	wf := compose.NewWorkflow[weeklyInput, weeklyOutput]()

	wf.AddChatTemplateNode("title_tpl", titleTpl).
		AddInput(compose.START, compose.MapFields("Topic", "topic"))
	wf.AddChatModelNode("title_model", c.chatModel).
		AddInput("title_tpl") // 同类型直传：模板产出的 messages 整个交给模型
	wf.AddLambdaNode("title_out", newMsgText("title")).
		AddInput("title_model")

	wf.AddChatTemplateNode("brief_tpl", briefTpl).
		AddInput(compose.START, compose.MapFields("Topic", "topic"))
	wf.AddChatModelNode("brief_model", c.chatModel).
		AddInput("brief_tpl")
	wf.AddLambdaNode("brief_out", newMsgText("brief")).
		AddInput("brief_model")

	// 4. 输出装配：各节点的结果填进 weeklyOutput 的哪个字段
	wf.AddEnd("title_out", compose.ToField("Title"))
	wf.AddEnd("brief_out", compose.ToField("Brief"))

	// 5. Compile：构建方法都不返回 error，映射/类型问题在 Compile 统一暴露
	runnable, err := wf.Compile(ctx)
	if err != nil {
		return err
	}

	// 6. Invoke：两条支路无依赖，自动并发跑
	out, err := runnable.Invoke(ctx, weeklyInput{Topic: "Go 泛型"})
	if err != nil {
		return err
	}

	fmt.Printf("标题: %s\n摘要: %s\n", out.Title, out.Brief)
	fmt.Printf("(两次 LLM 调用并行完成，总耗时约等于单次: %v)\n", time.Since(start).Round(time.Millisecond))
	return nil
}
