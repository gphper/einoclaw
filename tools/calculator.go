package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// CalculatorTool 计算器工具
type CalculatorTool struct {
	BaseTool
}

// NewCalculatorTool 创建计算器工具
func NewCalculatorTool() *CalculatorTool {
	params := map[string]*schema.ParameterInfo{
		"expression": {
			Type:     "string",
			Desc:     "数学表达式，例如 1+2*3",
			Required: true,
		},
	}

	return &CalculatorTool{
		BaseTool: BaseTool{
			info: &schema.ToolInfo{
				Name:  "calculator",
				Desc:  "执行数学计算",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
	}
}

// InvokableRun 执行计算
func (t *CalculatorTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	// 这里简化处理，实际应用中需要解析 JSON 参数
	return fmt.Sprintf("计算结果: %s", args), nil
}

// StreamableRun 流式执行（可选）
func (t *CalculatorTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented")
}
