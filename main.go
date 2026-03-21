package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"einoclaw/agent"
	"einoclaw/middlewares"
	"einoclaw/tools"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"

	// "github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	fmt.Println("=== Eino ADK 火山引擎 Coding Plain 模型示例 ===")

	ctx := context.Background()

	fmt.Println("1. 初始化火山引擎模型...")
	// apiKey := "3fe6ebfe-02ec-4608-86df-605399c7a658"
	apiKey := os.Getenv("APIKEY")
	if apiKey == "" {
		fmt.Println("   警告: 未设置 ARK_API_KEY 环境变量")
		fmt.Println("   请设置环境变量: export ARK_API_KEY=your_api_key")
		fmt.Println("   或在代码中直接设置 API Key")
		apiKey = "your_api_key_here"
	}

	arkChatModel, err := arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
		APIKey:  apiKey,
		Model:   "ark-code-latest",
		BaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3",
	})
	if err != nil {
		fmt.Printf("   创建火山引擎模型失败: %v\n", err)
		return
	}
	fmt.Println("   火山引擎模型创建成功!")

	fmt.Println("2. 配置工作空间...")
	// os.Setenv("WORKSPACE", "F:\\golangdemo\\einoclaw\\workspace")
	workspace := os.Getenv("WORKSPACE")
	if workspace == "" {
		workspace = filepath.Join(".", "workspace")
	}
	// os.Setenv("RESTRICT_ACCESS", "false")
	restrict := os.Getenv("RESTRICT_ACCESS") != "false"
	fmt.Printf("   工作空间: %s\n", workspace)
	fmt.Printf("   访问限制: %v\n", restrict)

	if err := os.MkdirAll(workspace, 0755); err != nil {
		fmt.Printf("   创建工作空间失败: %v\n", err)
		return
	}

	fmt.Println("3. 加载工具...")
	loadedTools := tools.LoadTools(workspace, restrict)
	fmt.Printf("   成功加载 %d 个工具: ", len(loadedTools))
	for i, tool := range loadedTools {
		info, _ := tool.Info(ctx)
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(info.Name)
	}
	fmt.Println()

	// 加载skill
	pwd, _ := os.Getwd()
	// workDir := filepath.Join(pwd, "adk", "middlewares", "skill", "workdir")
	skillsDir := filepath.Join(pwd, "skills")
	if err != nil {
		log.Fatal(err)
	}

	// 使用 Sandbox Backend 替代 Local Backend（Windows 兼容）
	// 注意：需要配置火山引擎 AgentKit 的访问密钥
	be, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		log.Fatal(err)
	}

	skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: be,
		BaseDir: skillsDir,
	})
	if err != nil {
		log.Fatalf("Failed to create skill backend: %v", err)
	}

	// 使用自定义 Backend 包装原始 Backend
	// customSkillBackend := skills.NewCustomBackend(skillBackend)

	sm, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		log.Fatalf("Failed to create skill middleware: %v", err)
	}

	fmt.Println("4. 创建 ChatModelAgent...")

	// 创建系统提示词构建器
	contextBuilder := agent.NewContextBuilder(workspace)
	agent, err := deep.New(ctx, &deep.Config{
		Name:        "ark-coding-agent",
		Description: "使用火山引擎 Coding Plain 模型的编程助手",
		ChatModel:   arkChatModel,
		Instruction: contextBuilder.BuildInstruction(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: loadedTools,
			},
		},
		Handlers: []adk.ChatModelAgentMiddleware{
			sm,
			&middlewares.SafeToolMiddleware{},
		},
	})
	if err != nil {
		fmt.Printf("   创建 Agent 失败: %v\n", err)
		return
	}
	fmt.Println("   Agent 创建成功!")

	fmt.Println("5. 创建 Runner...")
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	fmt.Println("\n=== 交互式对话模式 ===")
	fmt.Println("输入您的问题，输入 'quit' 或 'exit' 退出")
	fmt.Println("=====================================")

	reader := bufio.NewReader(os.Stdin)

	var conversationHistory []adk.Message

	for {
		fmt.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
			continue
		}

		query := strings.TrimSpace(input)
		if query == "" {
			continue
		}

		lowerQuery := strings.ToLower(query)
		if lowerQuery == "quit" || lowerQuery == "exit" {
			fmt.Println("再见！")
			break
		}

		userMessage := schema.UserMessage(query)
		conversationHistory = append(conversationHistory, userMessage)

		fmt.Printf("查询: %s\n", query)
		iter := runner.Run(ctx, conversationHistory)

		fmt.Println("处理中...")
		eventCount := 0
		var aiResponse string
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			eventCount++

			if event.Output != nil && event.Output.MessageOutput != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err == nil && msg != nil {
					switch event.Output.MessageOutput.Role {
					case schema.Tool:
						fmt.Printf("%s 工具结果: %s\n", event.Output.MessageOutput.ToolName, msg.Content)
						toolMessage := schema.ToolMessage(msg.Content, event.Output.MessageOutput.ToolName)
						conversationHistory = append(conversationHistory, toolMessage)
					case schema.Assistant:
						fmt.Printf("AI: %s\n", msg.Content)
						aiResponse = msg.Content
					}
				}
			}
			if event.Err != nil {
				fmt.Printf("错误: %v\n", event.Err)
			}
		}

		if aiResponse != "" {
			assistantMessage := schema.AssistantMessage(aiResponse, nil)
			conversationHistory = append(conversationHistory, assistantMessage)
		}
	}

	fmt.Println("=== 程序结束 ===")
}
