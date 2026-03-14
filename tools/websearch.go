package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	searchTimeout = 10 * time.Second
)

// 预编译的正则表达式
var (
	reTags       = regexp.MustCompile(`<[^>]+>`)
	reDDGLink    = regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	reDDGSnippet = regexp.MustCompile(`<a class="result__snippet[^"]*".*?>([\s\S]*?)</a>`)
)

// SearchProvider 搜索提供者接口
type SearchProvider interface {
	Search(ctx context.Context, query string, count int) (string, error)
}

// DuckDuckGoSearchProvider DuckDuckGo 搜索提供者
type DuckDuckGoSearchProvider struct {
	client *http.Client
}

// NewDuckDuckGoSearchProvider 创建 DuckDuckGo 搜索提供者
func NewDuckDuckGoSearchProvider() *DuckDuckGoSearchProvider {
	return &DuckDuckGoSearchProvider{
		client: &http.Client{
			Timeout: searchTimeout,
		},
	}
}

// Search 执行搜索
func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return p.extractResults(string(body), count, query)
}

// extractResults 从 HTML 中提取搜索结果
func (p *DuckDuckGoSearchProvider) extractResults(html string, count int, query string) (string, error) {
	matches := reDDGLink.FindAllStringSubmatch(html, count+5)

	if len(matches) == 0 {
		return fmt.Sprintf("No results found for: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via DuckDuckGo)", query))

	snippetMatches := reDDGSnippet.FindAllStringSubmatch(html, count+5)
	maxItems := minInt(len(matches), count)

	for i := 0; i < maxItems; i++ {
		urlStr := matches[i][1]
		title := stripTags(matches[i][2])
		title = strings.TrimSpace(title)

		// URL 解码
		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				_, after, ok := strings.Cut(u, "uddg=")
				if ok {
					urlStr = after
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, title, urlStr))

		if i < len(snippetMatches) {
			snippet := stripTags(snippetMatches[i][1])
			snippet = strings.TrimSpace(snippet)
			if snippet != "" {
				lines = append(lines, fmt.Sprintf("   %s", snippet))
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

// SearXNGSearchProvider SearXNG 搜索提供者
type SearXNGSearchProvider struct {
	baseURL string
	client  *http.Client
}

// NewSearXNGSearchProvider 创建 SearXNG 搜索提供者
func NewSearXNGSearchProvider(baseURL string) *SearXNGSearchProvider {
	return &SearXNGSearchProvider{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: searchTimeout,
		},
	}
}

// Search 执行搜索
func (p *SearXNGSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=general",
		strings.TrimSuffix(p.baseURL, "/"),
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SearXNG returned status %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Engine  string  `json:"engine"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	if len(result.Results) > count {
		result.Results = result.Results[:count]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Results for: %s (via SearXNG)\n", query))
	for i, r := range result.Results {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		b.WriteString(fmt.Sprintf("   %s\n", r.URL))
		if r.Content != "" {
			b.WriteString(fmt.Sprintf("   %s\n", r.Content))
		}
	}

	return b.String(), nil
}

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	BaseTool
	provider   SearchProvider
	maxResults int
}

// NewWebSearchTool 创建网络搜索工具
// providerType: "duckduckgo" 或 "searxng"
// searxngURL: SearXNG 实例地址（仅当 providerType 为 "searxng" 时需要）
func NewWebSearchTool(providerType string, searxngURL string, maxResults int) *WebSearchTool {
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	var provider SearchProvider
	switch providerType {
	case "searxng":
		if searxngURL != "" {
			provider = NewSearXNGSearchProvider(searxngURL)
		} else {
			// 如果没有提供 SearXNG URL，回退到 DuckDuckGo
			provider = NewDuckDuckGoSearchProvider()
		}
	case "duckduckgo":
		fallthrough
	default:
		provider = NewDuckDuckGoSearchProvider()
	}

	params := map[string]*schema.ParameterInfo{
		"query": {
			Type:     "string",
			Desc:     "Search query",
			Required: true,
		},
		"count": {
			Type:     "integer",
			Desc:     "Number of results (1-10)",
			Required: false,
		},
	}

	return &WebSearchTool{
		BaseTool: BaseTool{
			info: &schema.ToolInfo{
				Name:        "web_search",
				Desc:        "Search the web for current information. Returns titles, URLs, and snippets from search results.",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
		provider:   provider,
		maxResults: maxResults,
	}
}

// InvokableRun 执行搜索
func (t *WebSearchTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	var params map[string]any
	if err := parseJSON(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	query, ok := params["query"].(string)
	if !ok {
		return "", fmt.Errorf("query is required")
	}

	count := t.maxResults
	if c, ok := params["count"].(float64); ok {
		if int(c) > 0 && int(c) <= 10 {
			count = int(c)
		}
	}

	result, err := t.provider.Search(ctx, query, count)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	return result, nil
}

// StreamableRun 流式执行（不支持）
func (t *WebSearchTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented for web_search")
}

// stripTags 移除 HTML 标签
func stripTags(content string) string {
	return reTags.ReplaceAllString(content, "")
}

// minInt 返回两个整数中的最小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
