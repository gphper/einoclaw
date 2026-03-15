package tools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/schema"
)

// WindowsShell Windows 兼容的 Shell 实现
type WindowsShell struct {
	workingDir string
}

// NewWindowsShell 创建 Windows 兼容的 Shell
func NewWindowsShell(workingDir string) *WindowsShell {
	return &WindowsShell{
		workingDir: workingDir,
	}
}

// Execute 执行命令
func (ws *WindowsShell) Execute(ctx context.Context, req *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	command := req.Command
	if command == "" {
		return nil, fmt.Errorf("command is empty")
	}

	var cmd *exec.Cmd

	// 根据操作系统选择 shell
	if runtime.GOOS == "windows" {
		// Windows: 使用 PowerShell
		cmd = exec.CommandContext(ctx, "powershell", "-Command", command)
	} else {
		// Unix/Linux/Mac: 使用 sh
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// 设置工作目录
	if ws.workingDir != "" {
		cmd.Dir = ws.workingDir
	}

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 即使命令失败，也返回输出
		return &filesystem.ExecuteResponse{
			Output:   string(output),
			ExitCode: getExitCode(err),
		}, nil
	}

	return &filesystem.ExecuteResponse{
		Output:   string(output),
		ExitCode: nil,
	}, nil
}

// getExitCode 获取退出码
func getExitCode(err error) *int {
	if err == nil {
		return nil
	}

	// 尝试从错误中提取退出码
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		return &code
	}

	// 默认返回 1
	code := 1
	return &code
}

// StreamingWindowsShell Windows 兼容的流式 Shell 实现
type StreamingWindowsShell struct {
	workingDir string
}

// NewStreamingWindowsShell 创建 Windows 兼容的流式 Shell
func NewStreamingWindowsShell(workingDir string) *StreamingWindowsShell {
	return &StreamingWindowsShell{
		workingDir: workingDir,
	}
}

// ExecuteStreaming 流式执行命令
func (sws *StreamingWindowsShell) ExecuteStreaming(ctx context.Context, req *filesystem.ExecuteRequest) (*schema.StreamReader[*filesystem.ExecuteResponse], error) {
	command := req.Command
	if command == "" {
		return nil, fmt.Errorf("command is empty")
	}

	var cmd *exec.Cmd

	// 根据操作系统选择 shell
	if runtime.GOOS == "windows" {
		// Windows: 使用 PowerShell
		cmd = exec.CommandContext(ctx, "powershell", "-Command", command)
	} else {
		// Unix/Linux/Mac: 使用 sh
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// 设置工作目录
	if sws.workingDir != "" {
		cmd.Dir = sws.workingDir
	}

	// 创建管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// 创建流式读取器
	sr, sw := schema.Pipe[*filesystem.ExecuteResponse](10)

	// 异步读取输出
	go func() {
		defer sw.Close()

		// 读取 stdout 和 stderr
		output := make([]byte, 0, 1024)
		buf := make([]byte, 1024)

		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
				sw.Send(&filesystem.ExecuteResponse{
					Output: string(buf[:n]),
				}, nil)
			}
			if err != nil {
				break
			}
		}

		// 等待命令完成
		err = cmd.Wait()
		exitCode := getExitCode(err)

		// 发送最终结果
		sw.Send(&filesystem.ExecuteResponse{
			Output:   string(output),
			ExitCode: exitCode,
		}, nil)
	}()

	return sr, nil
}
