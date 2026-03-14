package tools

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// BaseTool 基础工具实现
type BaseTool struct {
	info *schema.ToolInfo
}

func (t *BaseTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}
