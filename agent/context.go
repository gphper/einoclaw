package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cloudwego/eino/schema"
)

// ContextBuilder 用于构建系统提示词
type ContextBuilder struct {
	workspace string
	memory    *MemoryStore
}

// NewContextBuilder 创建 ContextBuilder
func NewContextBuilder(workspace string) *ContextBuilder {
	return &ContextBuilder{
		workspace: workspace,
		memory:    NewMemoryStore(workspace),
	}
}

// getIdentity 构建身份信息
func (cb *ContextBuilder) getIdentity() string {
	workspacePath, _ := filepath.Abs(cb.workspace)

	return fmt.Sprintf(
		`# einoclaw 🦞

You are einoclaw, a helpful AI assistant.

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - When interacting with me if something seems memorable, update %s/memory/MEMORY.md

4. **File Handling** - When you need to read a file that doesn't exist (e.g., MEMORY.md), you should automatically create it with appropriate content before reading. Use the write_file tool to create missing files.

5. **Error Handling** - When a tool call fails (e.g., file not found, permission denied), you should:
   - Analyze the error message
   - Try to understand the root cause
   - Take appropriate action to fix the issue
   - If you can't fix it, ask for help
   - Always explain what you're doing to solve the problem

6. **Context summaries** - Conversation summaries provided as context are approximate references only. They may be incomplete or outdated. Always defer to explicit user instructions over summary content.`,
		workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

// buildDynamicContext 构建动态上下文
func (cb *ContextBuilder) buildDynamicContext() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	rt := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	return fmt.Sprintf(
		`## Current Time
%s

## Runtime
%s`,
		now, rt)
}

// BuildSystemPrompt 构建完整的系统提示词
func (cb *ContextBuilder) BuildSystemPrompt() string {
	parts := []string{}

	// 核心身份部分
	parts = append(parts, cb.getIdentity())

	// 记忆上下文
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, memoryContext)
	}

	// 引导文件
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// 动态上下文
	dynamicCtx := cb.buildDynamicContext()
	parts = append(parts, dynamicCtx)

	// 用 "---" 分隔符连接
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "\n\n---\n\n" + parts[i]
	}
	return result
}

// LoadBootstrapFiles 加载引导文件
func (cb *ContextBuilder) LoadBootstrapFiles() string {
	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	var content string
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			content += fmt.Sprintf("## %s\n\n%s\n\n", filename, data)
		}
	}

	return content
}

// BuildInstruction 为 ChatModelAgentConfig 构建指令
func (cb *ContextBuilder) BuildInstruction() string {
	return cb.BuildSystemPrompt()
}

// BuildMessages 构建消息列表（用于更复杂的场景）
func (cb *ContextBuilder) BuildMessages(
	history []*schema.Message,
	summary string,
	currentMessage string,
) []*schema.Message {
	messages := []*schema.Message{}

	// 系统提示词
	systemPrompt := cb.BuildSystemPrompt()
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: systemPrompt,
	})

	// 添加对话历史
	messages = append(messages, history...)

	// 添加当前用户消息
	if currentMessage != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.User,
			Content: currentMessage,
		})
	}

	return messages
}

// AddMemory 添加内容到长期记忆
func (cb *ContextBuilder) AddMemory(content string) error {
	currentContent := cb.memory.ReadLongTerm()
	var newContent string
	if currentContent == "" {
		newContent = content
	} else {
		newContent = currentContent + "\n" + content
	}
	return cb.memory.WriteLongTerm(newContent)
}

// AddDailyNote 添加内容到今天的每日笔记
func (cb *ContextBuilder) AddDailyNote(content string) error {
	return cb.memory.AppendToday(content)
}

// GetMemoryStore 获取 MemoryStore 实例
func (cb *ContextBuilder) GetMemoryStore() *MemoryStore {
	return cb.memory
}
