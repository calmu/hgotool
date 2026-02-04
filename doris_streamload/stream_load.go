package doris_streamload

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/calmu/hgotool/hlog"
	"go.uber.org/zap"
)

// Config StreamLoad客户端配置
type Config struct {
	Host       string            // Doris FE节点地址
	Port       string            // FE HTTP端口，默认8030
	Username   string            // 用户名
	Password   string            // 密码
	Database   string            // 目标数据库
	Table      string            // 目标表
	Label      string            // Label名称，用于保证幂等性
	BufferSize int               // 缓冲区大小，单位MB
	BufferRows int               // 缓冲行数
	Timeout    time.Duration     // 请求超时时间
	EnableGzip bool              // 是否启用gzip压缩
	Headers    map[string]string // 自定义HTTP头部参数
	HLogger    hlog.HLoggerBase  // 日志记录器
}

// 默认配置
var DefaultConfig = &Config{
	Port:       "8030",
	BufferSize: 10, // 10MB
	BufferRows: 1000,
	Timeout:    30 * time.Second,
	EnableGzip: false,
}

// StreamLoadResponse StreamLoad响应结果
type StreamLoadResponse struct {
	Status          string                   `json:"status"`
	Message         string                   `json:"msg"`
	Label           string                   `json:"label"`
	StmtID          int64                    `json:"stmtId"`
	LoadBytes       int64                    `json:"loadBytes"`
	LoadRows        int64                    `json:"loadRows"`
	LoadTimeMs      int64                    `json:"loadTimeMs"`
	ErrorURL        string                   `json:"errorURL,omitempty"`
	UnfinishedTasks []map[string]interface{} `json:"unfinishTasks,omitempty"`
	TrackingURL     string                   `json:"tracking_url"`
	Data            map[string]interface{}   `json:"data,omitempty"`
}

// StreamLoadClient StreamLoad客户端
type StreamLoadClient struct {
	config *Config
	client *http.Client
	mutex  sync.Mutex
	logger hlog.HLoggerBase
}

// NewStreamLoadClient 创建新的StreamLoad客户端
func NewStreamLoadClient(config *Config) (*StreamLoadClient, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if config.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if config.Database == "" {
		return nil, fmt.Errorf("database is required")
	}
	if config.Table == "" {
		return nil, fmt.Errorf("table is required")
	}
	if config.Port == "" {
		config.Port = DefaultConfig.Port
	}
	if config.BufferSize == 0 {
		config.BufferSize = DefaultConfig.BufferSize
	}
	if config.BufferRows == 0 {
		config.BufferRows = DefaultConfig.BufferRows
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultConfig.Timeout
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	// 初始化日志记录器
	logger := config.HLogger
	if logger == nil {
		logger = hlog.GlobalLoggers["default"]
		if logger == nil {
			// 如果没有默认日志记录器，则创建一个
			defaultLogger, _ := hlog.NewZapLogger(hlog.LoggerConfig{
				Level:      "info",
				OutputPath: []string{"stdout"},
				Encoder:    "console",
			})
			logger = defaultLogger
		}
	}

	return &StreamLoadClient{
		config: config,
		client: client,
		logger: logger,
	}, nil
}

// Load 将JSON数据加载到Doris
func (s *StreamLoadClient) Load(data []byte) (*StreamLoadResponse, error) {
	url := fmt.Sprintf("http://%s:%s/api/%s/%s/_stream_load", s.config.Host, s.config.Port, s.config.Database, s.config.Table)

	s.logger.Info("Starting StreamLoad request", zap.String("url", url), zap.Int("data_size", len(data)))

	// 创建请求
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		s.logger.Error("Failed to create HTTP request", zap.Error(err))
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Authorization", "Basic "+basicAuth(s.config.Username, s.config.Password))
	if s.config.Label != "" {
		req.Header.Set("label", s.config.Label)
	} else {
		// 自动生成label
		label := fmt.Sprintf("stream_load_%d", time.Now().Unix())
		req.Header.Set("label", label)
		s.logger.Info("Generated stream load label", zap.String("label", label))
	}
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "true") // 支持数组格式的JSON数据

	// 设置自定义头部参数
	for headerName, headerValue := range s.config.Headers {
		req.Header.Set(headerName, headerValue)
	}

	// 如果启用gzip压缩
	if s.config.EnableGzip {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(data); err != nil {
			s.logger.Error("Failed to compress data", zap.Error(err))
			return nil, err
		}
		if err := gz.Close(); err != nil {
			s.logger.Error("Failed to close gzip writer", zap.Error(err))
			return nil, err
		}
		req.Body = io.NopCloser(&buf)
		req.Header.Set("Content-Encoding", "gzip")
		s.logger.Info("Data compressed with gzip", zap.Int("original_size", len(data)), zap.Int("compressed_size", buf.Len()))
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error("Failed to send HTTP request", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body", zap.Error(err))
		return nil, err
	}

	// 解析响应
	var streamLoadResp StreamLoadResponse
	if err := json.Unmarshal(responseBody, &streamLoadResp); err != nil {
		s.logger.Error("Failed to unmarshal response", zap.Error(err), zap.String("response_body", string(responseBody)))
		return nil, fmt.Errorf("failed to unmarshal response: %v, body: %s", err, string(responseBody))
	}

	s.logger.Info("StreamLoad response received", zap.String("status", streamLoadResp.Status), zap.String("label", streamLoadResp.Label), zap.Int64("load_bytes", streamLoadResp.LoadBytes))

	// 检查状态
	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("StreamLoad request failed", zap.Int("status_code", resp.StatusCode), zap.String("response", string(responseBody)))
		return &streamLoadResp, fmt.Errorf("request failed with status: %d, response: %s", resp.StatusCode, string(responseBody))
	}

	s.logger.Info("StreamLoad request succeeded", zap.String("status", streamLoadResp.Status), zap.String("label", streamLoadResp.Label))
	return &streamLoadResp, nil
}

// basicAuth 生成Basic认证字符串
func basicAuth(username, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return encoded
}
