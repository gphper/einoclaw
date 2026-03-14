package tools

import (
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

	// 加载网络搜索工具
	// 优先使用 SearXNG，如果没有配置则使用 DuckDuckGo
	searxngURL := os.Getenv("SEARXNG_URL")
	providerType := "duckduckgo"
	if searxngURL != "" {
		providerType = "searxng"
	}
	webSearchTool := NewWebSearchTool(providerType, searxngURL, 5)
	tools = append(tools, webSearchTool)

	return tools
}
