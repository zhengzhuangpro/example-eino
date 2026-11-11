package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// GraphCompose Graph 图编排：Branch 条件分支。
//
// 场景是智能客服分流：先判断问题类型（技术/通用），走不同人设的提示词模板，
// 再汇聚到同一个 ChatModel 回答。拓扑：
//
//	START → classify ──Branch──┬─ tech_prompt ───┐
//	                           │                 ├─→ model → parser → END
//	                           └─ general_prompt ┘
//
// 与 chain 示例对照：Chain 只能一条道走到黑，Graph 的边显式声明，
// 既能分支也能汇聚——Chain 本质上是"无分支的特例 Graph"。
//
// 本示例学什么：
//   - NewGraph[I,O] + AddXXXNode：节点显式命名，比 Chain 的 Append 顺序更自由
//   - AddBranch 挂在"某个节点之后"：条件函数读该节点的输出、返回端点 key；
//     被选中的下游节点收到的是这同一份输出——所以 classify 必须把原始
//     输入透传下去（只返回分类结果的话，下游模板就拿不到 question 了）
//   - 分支后汇合：两个模板节点连到同一个 model 节点
//   - Compile 一次成 Runnable，Invoke 多次复用（两个问题走两条路由）
func (c *Client) GraphCompose(ctx context.Context) error {
	// 1. classify 节点：关键词判定意图。注意是"透传 + 附加 intent 字段"，
	//    因为 Branch 选中的下游节点拿到的就是这份输出
	classify := compose.InvokableLambda(func(ctx context.Context, vars map[string]any) (map[string]any, error) {
		question, _ := vars["question"].(string)
		for _, kw := range []string{"go", "代码", "报错", "bug", "编程", "部署"} {
			if strings.Contains(strings.ToLower(question), kw) {
				vars["intent"] = "tech"
				return vars, nil
			}
		}
		vars["intent"] = "general"
		return vars, nil
	})

	// 2. 两条路由各自的提示词模板
	techPrompt := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是资深 Go 工程师，用三句话以内给出可操作的技术建议。"),
		schema.UserMessage("{question}"))
	generalPrompt := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一位友好的通用助手，回答保持在两句话以内。"),
		schema.UserMessage("{question}"))

	// 3. Branch：条件函数返回端点 key，endNodes 声明所有合法去向
	routeIntent := compose.NewGraphBranch(
		func(ctx context.Context, vars map[string]any) (string, error) {
			if intent, _ := vars["intent"].(string); intent == "tech" {
				return "tech_prompt", nil
			}
			return "general_prompt", nil
		},
		map[string]bool{"tech_prompt": true, "general_prompt": true},
	)

	// 4. 组装 Graph：节点和边都显式命名声明（Add 系列都返回 error，逐个检查）
	graph := compose.NewGraph[map[string]any, string]()
	if err := graph.AddLambdaNode("classify", classify); err != nil {
		return err
	}
	if err := graph.AddChatTemplateNode("tech_prompt", techPrompt); err != nil {
		return err
	}
	if err := graph.AddChatTemplateNode("general_prompt", generalPrompt); err != nil {
		return err
	}
	if err := graph.AddChatModelNode("model", c.chatModel); err != nil {
		return err
	}
	if err := graph.AddLambdaNode("parser", compose.MessageParser[string](&contentParser{})); err != nil {
		return err // 复用 chain 示例的 contentParser：message -> string
	}

	if err := graph.AddEdge(compose.START, "classify"); err != nil {
		return err
	}
	if err := graph.AddBranch("classify", routeIntent); err != nil { // Branch 挂在 classify 之后
		return err
	}
	if err := graph.AddEdge("tech_prompt", "model"); err != nil { // 两条路汇聚到 model
		return err
	}
	if err := graph.AddEdge("general_prompt", "model"); err != nil {
		return err
	}
	if err := graph.AddEdge("model", "parser"); err != nil {
		return err
	}
	if err := graph.AddEdge("parser", compose.END); err != nil {
		return err
	}

	// 5. Compile：图结构固化，编译产物可复用、可并发调用
	runnable, err := graph.Compile(ctx)
	if err != nil {
		return err
	}

	// 6. 同一个 Runnable 跑两个问题：分类结果不同，走不同的路由
	questions := []string{
		"我的 Go 服务 goroutine 泄漏了，怎么排查？", // → tech 路由
		"今天心情不太好，说句话安慰我一下。",        // → general 路由
	}
	for _, q := range questions {
		answer, err := runnable.Invoke(ctx, map[string]any{"question": q})
		if err != nil {
			return err
		}
		fmt.Printf("Q: %s\nA: %s\n\n", q, answer)
	}
	return nil
}
