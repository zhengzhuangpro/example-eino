// example-eino：eino 学习示例集（通过 OpenAI 兼容端点接入 DeepSeek）。
//
// 用法：go run . <子命令>，不带参数查看全部示例。
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/example-eino/agent"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ctx := context.Background()

	client, err := agent.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var cmdErr error
	switch os.Args[1] {
	case "chatmodel": // 基础调用：直接使用 ChatModel 组件，Generate + Stream
		cmdErr = client.ChatModelBasic(ctx)
	case "chain": // Chain 编排：ChatTemplate -> ChatModel -> MessageParser
		cmdErr = client.ChainCompose(ctx)
	case "tool": // Tool Calling：手写"模型 <-> 工具"调用循环
		cmdErr = client.ToolCalling(ctx)
	case "adk": // ADK Agent：工具调用循环 + 事件流
		cmdErr = client.ADKAgent(ctx)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintln(os.Stderr, "未知子命令:", os.Args[1])
		usage()
		os.Exit(1)
	}

	if cmdErr != nil {
		log.Fatal(cmdErr)
	}
}

func usage() {
	fmt.Println(`eino 学习示例（DeepSeek 兼容端点）

用法: go run . <子命令>

子命令:
  chatmodel  基础调用：直接使用 ChatModel 组件，Generate + Stream 两种方式
  chain      Chain 编排：ChatTemplate -> ChatModel -> MessageParser 类型串联
  tool       Tool Calling：InferTool 定义工具，手写"调用 -> 执行 -> 回传"循环
  adk        ADK Agent：工具调用循环 + Runner 事件流

环境变量:
  DEEPSEEK_API_KEY  DeepSeek API Key（必须）`)
}
