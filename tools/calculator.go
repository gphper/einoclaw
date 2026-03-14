package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
				Name:        "calculator",
				Desc:        "执行数学计算",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
	}
}

// InvokableRun 执行计算
func (t *CalculatorTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	// 解析JSON参数
	var params map[string]string
	err := json.Unmarshal([]byte(args), &params)
	if err != nil {
		// 如果解析失败，尝试直接将args作为表达式
		result, err := evaluateExpression(args)
		if err != nil {
			return "", fmt.Errorf("计算错误: %v", err)
		}
		return fmt.Sprintf("计算结果: %f", result), nil
	}

	// 获取表达式参数
	expression, ok := params["expression"]
	if !ok {
		return "", fmt.Errorf("缺少表达式参数")
	}

	// 计算表达式
	result, err := evaluateExpression(expression)
	if err != nil {
		return "", fmt.Errorf("计算错误: %v", err)
	}
	return fmt.Sprintf("计算结果: %f", result), nil
}

// evaluateExpression 计算数学表达式
func evaluateExpression(expression string) (float64, error) {
	// 词法分析：将表达式分解为标记
	tokens, err := tokenize(expression)
	if err != nil {
		return 0, err
	}

	// 语法分析：将标记转换为后缀表达式（逆波兰表示法）
	postfix, err := infixToPostfix(tokens)
	if err != nil {
		return 0, err
	}

	// 计算后缀表达式
	return evaluatePostfix(postfix)
}

// token 表示表达式中的标记
type token struct {
	typ   string // "number" 或 "operator"
	value string
}

// tokenize 将表达式分解为标记
func tokenize(expression string) ([]token, error) {
	var tokens []token
	var numBuf []rune

	for _, ch := range expression {
		if ch >= '0' && ch <= '9' || ch == '.' {
			// 数字或小数点
			numBuf = append(numBuf, ch)
		} else if ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '(' || ch == ')' {
			// 运算符或括号
			if len(numBuf) > 0 {
				tokens = append(tokens, token{typ: "number", value: string(numBuf)})
				numBuf = nil
			}
			tokens = append(tokens, token{typ: "operator", value: string(ch)})
		} else if ch == ' ' {
			// 忽略空格
			if len(numBuf) > 0 {
				tokens = append(tokens, token{typ: "number", value: string(numBuf)})
				numBuf = nil
			}
		} else {
			return nil, fmt.Errorf("无效字符: %c", ch)
		}
	}

	// 处理最后一个数字
	if len(numBuf) > 0 {
		tokens = append(tokens, token{typ: "number", value: string(numBuf)})
	}

	return tokens, nil
}

// getPrecedence 获取运算符优先级
func getPrecedence(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}

// infixToPostfix 将中缀表达式转换为后缀表达式
func infixToPostfix(tokens []token) ([]token, error) {
	var output []token
	var stack []token

	for _, tok := range tokens {
		if tok.typ == "number" {
			// 数字直接加入输出
			output = append(output, tok)
		} else if tok.value == "(" {
			// 左括号入栈
			stack = append(stack, tok)
		} else if tok.value == ")" {
			// 右括号：弹出栈中元素直到遇到左括号
			for len(stack) > 0 && stack[len(stack)-1].value != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, fmt.Errorf("括号不匹配")
			}
			// 弹出左括号
			stack = stack[:len(stack)-1]
		} else {
			// 运算符：弹出栈中优先级更高或相等的运算符
			for len(stack) > 0 && stack[len(stack)-1].value != "(" && getPrecedence(stack[len(stack)-1].value) >= getPrecedence(tok.value) {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, tok)
		}
	}

	// 弹出栈中剩余的运算符
	for len(stack) > 0 {
		if stack[len(stack)-1].value == "(" {
			return nil, fmt.Errorf("括号不匹配")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

// evaluatePostfix 计算后缀表达式
func evaluatePostfix(tokens []token) (float64, error) {
	var stack []float64

	for _, tok := range tokens {
		if tok.typ == "number" {
			// 解析数字并入栈
			num, err := strconv.ParseFloat(tok.value, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数字: %s", tok.value)
			}
			stack = append(stack, num)
		} else {
			// 运算符：弹出两个操作数并计算
			if len(stack) < 2 {
				return 0, fmt.Errorf("表达式格式错误")
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var result float64
			switch tok.value {
			case "+":
				result = a + b
			case "-":
				result = a - b
			case "*":
				result = a * b
			case "/":
				if b == 0 {
					return 0, fmt.Errorf("除数不能为零")
				}
				result = a / b
			default:
				return 0, fmt.Errorf("未知运算符: %s", tok.value)
			}

			stack = append(stack, result)
		}
	}

	if len(stack) != 1 {
		return 0, fmt.Errorf("表达式格式错误")
	}

	return stack[0], nil
}

// StreamableRun 流式执行（可选）
func (t *CalculatorTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented")
}
