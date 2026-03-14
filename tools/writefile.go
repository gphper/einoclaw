package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// WriteFileTool 写入文件工具
type WriteFileTool struct {
	BaseTool
	workspace string
	restrict  bool
}

// NewWriteFileTool 创建写入文件工具
func NewWriteFileTool(workspace string, restrict bool) *WriteFileTool {
	params := map[string]*schema.ParameterInfo{
		"path": {
			Type:     "string",
			Desc:     "Path to the file to write",
			Required: true,
		},
		"content": {
			Type:     "string",
			Desc:     "Content to write to the file",
			Required: true,
		},
	}

	return &WriteFileTool{
		BaseTool: BaseTool{
			info: &schema.ToolInfo{
				Name:       "write_file",
				Desc:       "Write content to a file",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
		workspace: workspace,
		restrict:  restrict,
	}
}

// InvokableRun 执行文件写入
func (t *WriteFileTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	// 解析 JSON 参数
	var params map[string]any
	if err := parseJSON(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}

	// 验证路径
	validPath, err := t.validatePath(path)
	if err != nil {
		return "", err
	}

	// 确保目录存在
	dir := filepath.Dir(validPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// 写入文件
	if err := os.WriteFile(validPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File written successfully: %s", path), nil
}

// StreamableRun 流式执行（不支持）
func (t *WriteFileTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented for write_file")
}

// validatePath 验证路径
func (t *WriteFileTool) validatePath(path string) (string, error) {
	if t.workspace == "" {
		return path, nil
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	absPath := filepath.Join(t.workspace, path)
	return filepath.Clean(absPath), nil
}
