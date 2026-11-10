package agent

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// Client 是所有示例共享的入口：持有 ChatModel 实例，
// 每个示例是 Client 上的一个方法（一个文件一个示例）。
type Client struct {
	chatModel model.BaseChatModel // eino 的 ChatModel 组件接口，各厂商实现（OpenAI/Ark/Ollama…）都满足它
	model     string              // 模型名，仅用于展示
}

// NewClient 初始化共享 Client：通过 OpenAI 兼容协议接入 DeepSeek。
// API Key 从环境变量 DEEPSEEK_API_KEY 读取。
func NewClient(ctx context.Context) (*Client, error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-v4-pro",
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
	})
	if err != nil {
		return nil, err
	}

	return &Client{chatModel: chatModel, model: "deepseek-v4-pro"}, nil
}
