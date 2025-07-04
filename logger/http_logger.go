package logger

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogConfig 日志配置
type LogConfig struct {
	// 是否启用HTTP日志
	EnableHTTPLog bool

	// 输出目标：stdout, file, both
	Output string

	// 日志文件路径
	LogFile string

	// 是否记录请求体
	LogRequestBody bool

	// 是否记录响应体
	LogResponseBody bool
}

// HTTPLogger HTTP日志记录器
type HTTPLogger struct {
	config *LogConfig
	logger *log.Logger
	file   *os.File
}

// ResponseWriter 包装器，用于捕获响应数据
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     200,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// NewHTTPLogger 创建HTTP日志记录器
func NewHTTPLogger(config *LogConfig) (*HTTPLogger, error) {
	if config == nil {
		config = &LogConfig{
			EnableHTTPLog:   true,
			Output:          "stdout",
			LogRequestBody:  false,
			LogResponseBody: true,
		}
	}

	httpLogger := &HTTPLogger{
		config: config,
	}

	// 设置输出目标
	if err := httpLogger.setupOutput(); err != nil {
		return nil, fmt.Errorf("设置日志输出失败: %v", err)
	}

	return httpLogger, nil
}

// setupOutput 设置日志输出目标
func (h *HTTPLogger) setupOutput() error {
	switch h.config.Output {
	case "stdout":
		h.logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

	case "file":
		if h.config.LogFile == "" {
			h.config.LogFile = "pi-keyboard.log"
		}

		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(h.config.LogFile), 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %v", err)
		}

		file, err := os.OpenFile(h.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %v", err)
		}
		h.file = file
		h.logger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)

	case "both":
		if h.config.LogFile == "" {
			h.config.LogFile = "pi-keyboard.log"
		}

		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(h.config.LogFile), 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %v", err)
		}

		file, err := os.OpenFile(h.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %v", err)
		}
		h.file = file

		// 同时输出到标准输出和文件
		multiWriter := io.MultiWriter(os.Stdout, file)
		h.logger = log.New(multiWriter, "", log.LstdFlags|log.Lmicroseconds)

	default:
		return fmt.Errorf("不支持的输出类型: %s", h.config.Output)
	}

	return nil
}

// Middleware HTTP日志中间件
func (h *HTTPLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否启用HTTP日志
		if !h.config.EnableHTTPLog {
			next.ServeHTTP(w, r)
			return
		}

		startTime := time.Now()

		// 读取请求体
		var requestBody []byte
		if h.config.LogRequestBody && r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 使用包装的ResponseWriter
		rw := newResponseWriter(w)

		// 处理请求
		next.ServeHTTP(rw, r)

		// 记录日志
		h.logRequest(r, rw, startTime, requestBody)
	})
}

// logRequest 记录请求日志
func (h *HTTPLogger) logRequest(r *http.Request, rw *responseWriter, startTime time.Time, requestBody []byte) {
	duration := time.Since(startTime)
	responseBody := rw.body.String()

	// 构建日志信息
	logParts := []string{
		fmt.Sprintf("[HTTP] %s %s", r.Method, r.RequestURI),
		fmt.Sprintf("%d", rw.statusCode),
		duration.String(),
		r.RemoteAddr,
	}

	// 添加响应体信息
	if h.config.LogResponseBody && responseBody != "" {
		// 限制响应体长度
		if len(responseBody) > 100 {
			responseBody = responseBody[:100] + "..."
		}
		logParts = append(logParts, fmt.Sprintf("→ %s", responseBody))
	}

	// 添加请求/响应大小信息
	logParts = append(logParts, fmt.Sprintf("req:%db resp:%db", len(requestBody), len(responseBody)))

	// 如果需要记录请求体
	if h.config.LogRequestBody && len(requestBody) > 0 {
		reqBodyStr := string(requestBody)
		if len(reqBodyStr) > 200 {
			reqBodyStr = reqBodyStr[:200] + "..."
		}
		logParts = append(logParts, fmt.Sprintf("body:%s", reqBodyStr))
	}

	// 输出日志
	h.logger.Printf("%s", strings.Join(logParts, " | "))
}

// Close 关闭日志记录器
func (h *HTTPLogger) Close() error {
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}

// DefaultConfig 默认配置
func DefaultConfig() *LogConfig {
	return &LogConfig{
		EnableHTTPLog:   true,
		Output:          "stdout",
		LogRequestBody:  false,
		LogResponseBody: true,
	}
}
