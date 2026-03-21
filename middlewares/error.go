package middlewares

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
)

type SafeToolMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *SafeToolMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			// 将错误转换为字符串，而不是返回错误
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		return result, nil
	}, nil
}
