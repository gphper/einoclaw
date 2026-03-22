package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	fetchTimeout    = 60 * time.Second
	defaultMaxChars = 50000
	maxRedirects    = 5
)

var (
	reScript     = regexp.MustCompile(`<script[\s\S]*?</script>`)
	reStyle      = regexp.MustCompile(`<style[\s\S]*?</style>`)
	reTags       = regexp.MustCompile(`<[^>]+>`)
	reWhitespace = regexp.MustCompile(`[^\S\n]+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
)

var allowPrivateWebFetchHosts atomic.Bool

type WebFetchTool struct {
	BaseTool
	maxChars        int
	fetchLimitBytes int64
	client          *http.Client
}

func NewWebFetchTool(maxChars int, fetchLimitBytes int64) (*WebFetchTool, error) {
	if maxChars <= 0 {
		maxChars = defaultMaxChars
	}

	client := &http.Client{
		Timeout: fetchTimeout,
	}

	if transport, ok := client.Transport.(*http.Transport); ok {
		dialer := &net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		transport.DialContext = newSafeDialContext(dialer)
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		if isObviousPrivateHost(req.URL.Hostname()) {
			return fmt.Errorf("redirect target is private or local network host")
		}
		return nil
	}

	if fetchLimitBytes <= 0 {
		fetchLimitBytes = 10 * 1024 * 1024
	}

	params := map[string]*schema.ParameterInfo{
		"url": {
			Type:     "string",
			Desc:     "URL to fetch",
			Required: true,
		},
		"maxChars": {
			Type:     "integer",
			Desc:     "Maximum characters to extract",
			Required: false,
		},
	}

	return &WebFetchTool{
		BaseTool: BaseTool{
			info: &schema.ToolInfo{
				Name:        "web_fetch",
				Desc:        "Fetch a URL and extract readable content (HTML to text). Use this to get weather info, news, articles, or any web content.",
				ParamsOneOf: schema.NewParamsOneOfByParams(params),
			},
		},
		maxChars:        maxChars,
		fetchLimitBytes: fetchLimitBytes,
		client:          client,
	}, nil
}

func (t *WebFetchTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	var params map[string]any
	if err := parseJSON(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	urlStr, ok := params["url"].(string)
	if !ok {
		return "", fmt.Errorf("url is required")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("only http/https URLs are allowed")
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("missing domain in URL")
	}

	hostname := parsedURL.Hostname()
	if isObviousPrivateHost(hostname) {
		return "", fmt.Errorf("fetching private or local network hosts is not allowed")
	}

	maxChars := t.maxChars
	if mc, ok := params["maxChars"].(float64); ok {
		if int(mc) > 100 {
			maxChars = int(mc)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", userAgent)
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}

	resp.Body = http.MaxBytesReader(nil, resp.Body, t.fetchLimitBytes)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return "", fmt.Errorf("failed to read response: size exceeded %d bytes limit", t.fetchLimitBytes)
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")

	var text, extractor string

	if strings.Contains(contentType, "application/json") {
		var jsonData any
		if err := json.Unmarshal(body, &jsonData); err == nil {
			formatted, _ := json.MarshalIndent(jsonData, "", "  ")
			text = string(formatted)
			extractor = "json"
		} else {
			text = string(body)
			extractor = "raw"
		}
	} else if strings.Contains(contentType, "text/html") || len(body) > 0 &&
		(strings.HasPrefix(string(body), "<!DOCTYPE") || strings.HasPrefix(strings.ToLower(string(body)), "<html")) {
		text = t.extractText(string(body))
		extractor = "html"
	} else {
		text = string(body)
		extractor = "raw"
	}

	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars]
	}

	result := map[string]any{
		"url":       urlStr,
		"status":    resp.StatusCode,
		"extractor": extractor,
		"truncated": truncated,
		"length":    len(text),
		"text":      text,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return string(resultJSON), nil
}

func (t *WebFetchTool) StreamableRun(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
	return nil, fmt.Errorf("streaming not implemented for web_fetch")
}

func (t *WebFetchTool) extractText(htmlContent string) string {
	result := reScript.ReplaceAllLiteralString(htmlContent, "")
	result = reStyle.ReplaceAllLiteralString(result, "")
	result = reTags.ReplaceAllLiteralString(result, "")

	result = strings.TrimSpace(result)

	result = reWhitespace.ReplaceAllString(result, " ")
	result = reBlankLines.ReplaceAllString(result, "\n\n")

	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

func newSafeDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if allowPrivateWebFetchHosts.Load() {
			return dialer.DialContext(ctx, network, address)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid target address %q: %w", address, err)
		}
		if host == "" {
			return nil, fmt.Errorf("empty target host")
		}

		if ip := net.ParseIP(host); ip != nil {
			if isPrivateOrRestrictedIP(ip) {
				return nil, fmt.Errorf("blocked private or local target: %s", host)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}

		ipAddrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
		}

		attempted := 0
		var lastErr error
		for _, ipAddr := range ipAddrs {
			if isPrivateOrRestrictedIP(ipAddr.IP) {
				continue
			}
			attempted++
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}

		if attempted == 0 {
			return nil, fmt.Errorf("all resolved addresses for %s are private or restricted", host)
		}
		if lastErr != nil {
			return nil, fmt.Errorf("failed connecting to public addresses for %s: %w", host, lastErr)
		}
		return nil, fmt.Errorf("failed connecting to public addresses for %s", host)
	}
}

func isObviousPrivateHost(host string) bool {
	if allowPrivateWebFetchHosts.Load() {
		return false
	}

	h := strings.ToLower(strings.TrimSpace(host))
	h = strings.TrimSuffix(h, ".")
	if h == "" {
		return true
	}

	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}

	if ip := net.ParseIP(h); ip != nil {
		return isPrivateOrRestrictedIP(ip)
	}

	return false
}

func isPrivateOrRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 ||
			ip4[0] == 127 ||
			ip4[0] == 0 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			(ip4[0] == 169 && ip4[1] == 254) ||
			(ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127) {
			return true
		}
		return false
	}

	if len(ip) == net.IPv6len {
		if (ip[0] & 0xfe) == 0xfc {
			return true
		}
		if ip[0] == 0x20 && ip[1] == 0x02 {
			embedded := net.IPv4(ip[2], ip[3], ip[4], ip[5])
			return isPrivateOrRestrictedIP(embedded)
		}
		if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x00 && ip[3] == 0x00 {
			client := net.IPv4(ip[12]^0xff, ip[13]^0xff, ip[14]^0xff, ip[15]^0xff)
			return isPrivateOrRestrictedIP(client)
		}
	}

	return false
}
