package tools

import (
	"log"
	"os"

	"github.com/cloudwego/eino/components/tool"
)

// LoadTools 加载所有工具
func LoadTools(workspace string, restrict bool) []tool.BaseTool {
	var tools []tool.BaseTool

	// 加载计算器工具
	calculator := NewCalculatorTool()
	tools = append(tools, calculator)

	// 加载文件系统工具
	readFileTool := NewReadFileTool(workspace, restrict, 0)
	tools = append(tools, readFileTool)

	writeFileTool := NewWriteFileTool(workspace, restrict)
	tools = append(tools, writeFileTool)

	listDirTool := NewListDirTool(workspace, restrict)
	tools = append(tools, listDirTool)

	// 加载命令执行工具
	shellTool := NewShellTool(workspace, restrict)
	tools = append(tools, shellTool)

	// 加载网页搜索工具（支持 SearXNG 或 DuckDuckGo）
	searxngURL := os.Getenv("SEARXNG_URL")
	log.Printf("SEARXNG_URL: %s", searxngURL)
	providerType := "duckduckgo"
	if searxngURL != "" {
		providerType = "searxng"
	}
	webSearchTool := NewWebSearchTool(providerType, searxngURL, 5)
	tools = append(tools, webSearchTool)

	// 加载网页获取工具
	webFetchTool, err := NewWebFetchTool(0, 0)
	if err == nil {
		tools = append(tools, webFetchTool)
	}

	return tools
}
