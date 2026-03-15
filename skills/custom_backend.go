package skills

import (
	"context"
	"log"
	"strings"

	"github.com/cloudwego/eino/adk/middlewares/skill"
)

// CustomBackend 自定义技能后端，处理 Windows shell 兼容性
type CustomBackend struct {
	baseBackend skill.Backend
}

// NewCustomBackend 创建自定义技能后端
func NewCustomBackend(baseBackend skill.Backend) *CustomBackend {
	return &CustomBackend{
		baseBackend: baseBackend,
	}
}

// List 列出所有可用技能
func (cb *CustomBackend) List(ctx context.Context) ([]skill.FrontMatter, error) {

	tmp, err := cb.baseBackend.List(ctx)

	log.Printf("%+v", tmp)

	return tmp, err
}

// Get 获取技能内容，并修改命令执行方式
func (cb *CustomBackend) Get(ctx context.Context, name string) (skill.Skill, error) {
	s, err := cb.baseBackend.Get(ctx, name)
	if err != nil {
		return skill.Skill{}, err
	}

	// 修改 SKILL.md 内容，将命令调用改为使用 exec 工具
	s.Content = cb.modifyContentForWindows(s.Content, s.BaseDirectory)

	return s, nil
}

// modifyContentForWindows 修改 SKILL.md 内容以适配 Windows
func (cb *CustomBackend) modifyContentForWindows(content, baseDir string) string {
	// 如果内容已经包含 "call the exec tool" 或 "use the exec tool"，则不需要修改
	if strings.Contains(content, "call the exec tool") || strings.Contains(content, "use the exec tool") {
		return content
	}

	// 查找代码块中的命令
	lines := strings.Split(content, "\n")
	modifiedLines := make([]string, 0, len(lines))
	inCodeBlock := false
	codeBlockContent := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检测代码块开始
		if strings.HasPrefix(trimmed, "```") && !inCodeBlock {
			inCodeBlock = true
			codeBlockContent = []string{line}
			continue
		}

		// 检测代码块结束
		if strings.HasPrefix(trimmed, "```") && inCodeBlock {
			inCodeBlock = false
			codeBlockContent = append(codeBlockContent, line)

			// 处理代码块内容
			modifiedBlock := cb.processCodeBlock(codeBlockContent, baseDir)
			modifiedLines = append(modifiedLines, modifiedBlock...)
			continue
		}

		// 在代码块中
		if inCodeBlock {
			codeBlockContent = append(codeBlockContent, line)
			continue
		}

		modifiedLines = append(modifiedLines, line)
	}

	return strings.Join(modifiedLines, "\n")
}

// processCodeBlock 处理代码块，将命令转换为 exec 工具调用
func (cb *CustomBackend) processCodeBlock(codeBlock []string, baseDir string) []string {
	if len(codeBlock) < 2 {
		return codeBlock
	}

	// 获取代码块内容（去掉标记）
	content := strings.Join(codeBlock[1:len(codeBlock)-1], "\n")
	trimmed := strings.TrimSpace(content)

	// 如果已经是 JSON 格式，不需要修改
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return codeBlock
	}

	// 如果代码块包含命令，转换为 exec 工具调用
	if cb.isCommand(trimmed) {
		// 替换 {{.BaseDirectory}} 为实际路径
		command := strings.ReplaceAll(trimmed, "{{.BaseDirectory}}", baseDir)

		// 创建新的代码块
		newBlock := []string{
			codeBlock[0], // 开始标记
			`{"command": "` + command + `"}`,
			codeBlock[len(codeBlock)-1], // 结束标记
		}
		return newBlock
	}

	return codeBlock
}

// isCommand 判断是否是命令
func (cb *CustomBackend) isCommand(text string) bool {
	// 常见命令模式
	commandPrefixes := []string{
		"python", "python3", "pip", "pip3",
		"node", "npm", "yarn",
		"java", "javac",
		"go", "gofmt",
		"git", "curl", "wget",
		"ls", "cd", "pwd", "mkdir", "rm", "cp", "mv",
		"cat", "echo", "grep", "find",
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	for _, prefix := range commandPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	// 检查是否是路径（可能是脚本）
	if strings.Contains(trimmed, ".py") || strings.Contains(trimmed, ".js") ||
		strings.Contains(trimmed, ".sh") || strings.Contains(trimmed, ".bat") ||
		strings.Contains(trimmed, ".exe") {
		return true
	}

	return false
}
