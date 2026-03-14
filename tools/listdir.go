package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ListDirTool 列出目录工具
type ListDirTool struct {
	BaseTool
	workspace string
	restrict  bool
}

// NewListDirTool 创建列出目录工具
func NewListDirTool(workspace string, restrict bool) *ListDirTool {
	params := map[string]*schema.ParameterInfo{
		"path": {
			Type:     "string",
			Desc:     "Path to list (default: current directory)",
			Required: false,
		},
	}

	return &ListDirTool{
		BaseTool: BaseTool{
			info: &schema.ToolInfo{
				Name:       "list_dir",
				Desc:       "List files and directories in a path",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
		workspace: workspace,
		restrict:  restrict,
	}
}

// InvokableRun 执行目录列表
func (t *ListDirTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	// 解析 JSON 参数
	var params map[string]any
	if err := parseJSON(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// path (optional, default ".")
	path, ok := params["path"].(string)
	if !ok {
		path = "."
	}

	// 验证路径
	validPath, err := t.validatePath(path)
	if err != nil {
		return "", err
	}

	// 读取目录
	entries, err := os.ReadDir(validPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	// 格式化输出
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("DIR:  %s\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("FILE: %s\n", entry.Name()))
		}
	}

	return result.String(), nil
}

// StreamableRun 流式执行（不支持）
func (t *ListDirTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented for list_dir")
}

// validatePath 验证路径
func (t *ListDirTool) validatePath(path string) (string, error) {
	if t.workspace == "" {
		return path, nil
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	absPath := filepath.Join(t.workspace, path)
	return filepath.Clean(absPath), nil
}
