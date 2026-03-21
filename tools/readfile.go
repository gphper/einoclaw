package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const MaxReadFileSize = 64 * 1024 // 64KB limit to avoid context overflow

// ReadFileTool 读取文件工具
type ReadFileTool struct {
	BaseTool
	workspace string
	restrict  bool
	maxSize   int64
}

// NewReadFileTool 创建读取文件工具
func NewReadFileTool(workspace string, restrict bool, maxReadFileSize int) *ReadFileTool {
	maxSize := int64(maxReadFileSize)
	if maxSize <= 0 {
		maxSize = MaxReadFileSize
	}

	params := map[string]*schema.ParameterInfo{
		"path": {
			Type:     "string",
			Desc:     "Path to the file to read",
			Required: true,
		},
		"offset": {
			Type:     "integer",
			Desc:     "Byte offset to start reading from",
			Required: false,
		},
		"length": {
			Type:     "integer",
			Desc:     "Maximum number of bytes to read",
			Required: false,
		},
	}

	return &ReadFileTool{
		BaseTool: BaseTool{
			info: &schema.ToolInfo{
				Name:        "read_file",
				Desc:        "Read contents of a file. Supports pagination via offset and length",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
		workspace: workspace,
		restrict:  restrict,
		maxSize:   maxSize,
	}
}

// InvokableRun 执行文件读取
func (t *ReadFileTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	// 解析 JSON 参数
	var params map[string]any
	if err := parseJSON(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	// 验证路径
	validPath, err := t.validatePath(path)
	if err != nil {
		return "", err
	}

	// offset (optional, default 0)
	offset, err := getInt64Arg(params, "offset", 0)
	if err != nil {
		return "", err
	}
	if offset < 0 {
		return "", fmt.Errorf("offset must be >= 0")
	}

	// length (optional, capped at MaxReadFileSize)
	length, err := getInt64Arg(params, "length", t.maxSize)
	if err != nil {
		return "", err
	}
	if length <= 0 {
		return "", fmt.Errorf("length must be > 0")
	}
	if length > t.maxSize {
		length = t.maxSize
	}

	// 读取文件
	file, err := os.Open(validPath)
	if err != nil {

		if os.IsNotExist(err) {

			return fmt.Sprintf("[FILE NOT FOUND] The file %s does not exist.", validPath), nil
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取文件大小
	totalSize := int64(-1)
	if info, statErr := file.Stat(); statErr == nil {
		totalSize = info.Size()
	}

	// 检测二进制内容
	sniff := make([]byte, 512)
	_, _ = file.Read(sniff)

	// 重置文件位置
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to reset file position: %w", err)
	}

	// 跳转到指定偏移量
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	// 读取内容
	probe := make([]byte, length+1)
	n, err := io.ReadFull(file, probe)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	hasMore := int64(n) > length
	data := probe[:min(int64(n), length)]

	if len(data) == 0 {
		return "[END OF FILE - no content at this offset]", nil
	}

	// 构建结果
	readEnd := offset + int64(len(data))
	readRange := fmt.Sprintf("bytes %d-%d", offset, readEnd-1)
	displayPath := filepath.Base(path)

	var header string
	if totalSize >= 0 {
		header = fmt.Sprintf("[file: %s | total: %d bytes | read: %s]", displayPath, totalSize, readRange)
	} else {
		header = fmt.Sprintf("[file: %s | read: %s | total size unknown]", displayPath, readRange)
	}

	if hasMore {
		header += fmt.Sprintf("\n[TRUNCATED - file has more content. Call read_file again with offset=%d to continue.]", readEnd)
	} else {
		header += "\n[END OF FILE - no further content.]"
	}

	return header + "\n\n" + string(data), nil
}

// StreamableRun 流式执行（不支持）
func (t *ReadFileTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented for read_file")
}

// validatePath 验证路径
func (t *ReadFileTool) validatePath(path string) (string, error) {
	if t.workspace == "" {
		return path, nil
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	absPath := filepath.Join(t.workspace, path)
	return filepath.Clean(absPath), nil
}

// getInt64Arg 提取整数参数
func getInt64Arg(args map[string]any, key string, defaultVal int64) (int64, error) {
	raw, exists := args[key]
	if !exists {
		return defaultVal, nil
	}

	switch v := raw.(type) {
	case float64:
		if v != float64(int64(v)) {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported type for %s", key)
	}
}

// parseJSON 解析 JSON 参数
func parseJSON(jsonStr string, target interface{}) error {
	if jsonStr == "" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), target)
}

// min 返回最小值
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
