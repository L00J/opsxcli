package websearch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"opsxcli/internal/logger"
	"opsxcli/internal/tui"
)

// SearchEngine 搜索引擎配置
type SearchEngine struct {
	Name        string
	URL         string
	QueryParam  string
	UserAgent   string
	NeedToken   bool
	Region      string // "CN" 或 "Global"
}

// 预定义的搜索引擎列表
var SearchEngines = map[string]SearchEngine{
	// 国内搜索引擎
	"baidu": {
		Name:       "百度",
		URL:        "https://www.baidu.com/s",
		QueryParam: "wd",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "CN",
	},
	"sogou": {
		Name:       "搜狗",
		URL:        "https://www.sogou.com/web",
		QueryParam: "query",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "CN",
	},
	"so360": {
		Name:       "360搜索",
		URL:        "https://www.so.com/s",
		QueryParam: "q",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "CN",
	},
	"bing_cn": {
		Name:       "必应中国",
		URL:        "https://cn.bing.com/search",
		QueryParam: "q",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "CN",
	},

	// 海外搜索引擎
	"google": {
		Name:       "Google",
		URL:        "https://www.google.com/search",
		QueryParam: "q",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "Global",
	},
	"bing": {
		Name:       "Bing",
		URL:        "https://www.bing.com/search",
		QueryParam: "q",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "Global",
	},
	"duckduckgo": {
		Name:       "DuckDuckGo",
		URL:        "https://duckduckgo.com/",
		QueryParam: "q",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "Global",
	},
	"yahoo": {
		Name:       "Yahoo",
		URL:        "https://search.yahoo.com/search",
		QueryParam: "p",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		NeedToken:  false,
		Region:     "Global",
	},
}

// Search 执行搜索
func Search(query string, engines []string, saveHTML bool) error {
	// 如果没有指定搜索引擎,使用默认
	if len(engines) == 0 {
		engines = []string{"baidu", "google"}
	}

	// 创建状态提示
	status := tui.NewToolStatus()
	engineNames := getEngineNames(engines)
	statusMsg := fmt.Sprintf("\"%s\" via %s", query, strings.Join(engineNames, ", "))
	status.StartWithMessage("web_search", statusMsg)

	results := make(map[string]SearchResult)
	var lastErr error

	// 对每个搜索引擎执行搜索
	for _, engineKey := range engines {
		engine, ok := SearchEngines[engineKey]
		if !ok {
			logger.Error("未知的搜索引擎: %s", engineKey)
			continue
		}

		result, err := searchSingle(query, engine, saveHTML)
		if err != nil {
			logger.Error("%s 搜索失败: %v", engine.Name, err)
			lastErr = err
			continue
		}

		results[engine.Name] = result
	}

	// 停止状态提示
	if len(results) > 0 {
		msg := fmt.Sprintf("找到 %d 个搜索结果", len(results))
		status.Stop(true, msg)

		// 显示结果摘要
		displayResults(results)
	} else {
		status.Stop(false, "所有搜索引擎都失败了")
		return lastErr
	}

	return nil
}

// SearchResult 搜索结果
type SearchResult struct {
	Engine   string
	URL      string
	HTMLSize int
	Duration time.Duration
	Error    error
}

// searchSingle 单个搜索引擎搜索
func searchSingle(query string, engine SearchEngine, saveHTML bool) (SearchResult, error) {
	startTime := time.Now()

	// 构建搜索 URL
	searchURL := fmt.Sprintf("%s?%s=%s", engine.URL, engine.QueryParam, url.QueryEscape(query))

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return SearchResult{Engine: engine.Name, Error: err}, err
	}

	// 设置请求头,模拟浏览器
	req.Header.Set("User-Agent", engine.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 允许重定向,但限制次数
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			return nil
		},
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return SearchResult{Engine: engine.Name, Error: err}, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResult{Engine: engine.Name, Error: err}, err
	}

	duration := time.Since(startTime)

	// 保存 HTML (可选)
	if saveHTML {
		filename := fmt.Sprintf("%s_%s.html",
			engine.Name,
			time.Now().Format("20060102_150405"))

		if err := saveToFile(filename, body); err != nil {
			logger.Error("保存 HTML 失败: %v", err)
		} else {
			logger.Success("HTML 已保存到: %s", filename)
		}
	}

	result := SearchResult{
		Engine:   engine.Name,
		URL:      searchURL,
		HTMLSize: len(body),
		Duration: duration,
	}

	return result, nil
}

// displayResults 显示搜索结果摘要
func displayResults(results map[string]SearchResult) {
	fmt.Println("\n搜索结果:")
	fmt.Println(strings.Repeat("─", 80))

	for engineName, result := range results {
		icon := getRegionIcon(engineName)
		size := formatBytes(result.HTMLSize)

		fmt.Printf("%s %s\n", icon, engineName)
		fmt.Printf("  URL: %s\n", result.URL)
		fmt.Printf("  大小: %s | 耗时: %s\n", size, result.Duration)
		fmt.Println()
	}
}

// getRegionIcon 获取区域图标
func getRegionIcon(engineName string) string {
	for key, engine := range SearchEngines {
		if engine.Name == engineName {
			if SearchEngines[key].Region == "CN" {
				return "🇨🇳"
			}
			return "🌍"
		}
	}
	return "🔍"
}

// getEngineNames 获取搜索引擎名称列表
func getEngineNames(keys []string) []string {
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		if engine, ok := SearchEngines[key]; ok {
			names = append(names, engine.Name)
		}
	}
	return names
}

// saveToFile 保存到文件
func saveToFile(filename string, data []byte) error {
	// TODO: 实现文件保存逻辑
	return nil
}

// formatBytes 格式化字节数
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ListEngines 列出所有可用的搜索引擎
func ListEngines() {
	fmt.Println("可用的搜索引擎:")
	fmt.Println(strings.Repeat("─", 80))

	fmt.Println("\n🇨🇳 国内搜索引擎:")
	for key, engine := range SearchEngines {
		if engine.Region == "CN" {
			tokenStr := "✓ 免Token"
			if engine.NeedToken {
				tokenStr = "✗ 需要Token"
			}
			fmt.Printf("  %-12s %s (%s)\n", key, engine.Name, tokenStr)
		}
	}

	fmt.Println("\n🌍 海外搜索引擎:")
	for key, engine := range SearchEngines {
		if engine.Region == "Global" {
			tokenStr := "✓ 免Token"
			if engine.NeedToken {
				tokenStr = "✗ 需要Token"
			}
			fmt.Printf("  %-12s %s (%s)\n", key, engine.Name, tokenStr)
		}
	}

	fmt.Println("\n使用示例:")
	fmt.Println("  opsxcli websearch \"你的搜索关键词\" --engines baidu,google")
	fmt.Println("  opsxcli websearch \"语雀 API 文档\" --engines baidu,bing_cn,google")
}
