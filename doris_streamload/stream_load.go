package doris_streamload

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/calmu/hgotool/hlog"
	"go.uber.org/zap"
)

// Config StreamLoad客户端配置
type Config struct {
	Host        string            // Doris FE节点地址
	Port        string            // FE HTTP端口，默认8030
	Username    string            // 用户名
	Password    string            // 密码
	Database    string            // 目标数据库
	Table       string            // 目标表
	Label       string            // Label名称，用于保证幂等性
	LabelPrefix string            // Label前缀，用于自动生成Label，保证幂等性
	BufferSize  int               // 缓冲区大小，单位MB
	BufferRows  int               // 缓冲行数
	Timeout     time.Duration     // 请求超时时间
	EnableGzip  bool              // 是否启用gzip压缩
	Headers     map[string]string // 自定义HTTP头部参数
	HLogger     hlog.HLoggerBase  // 日志记录器
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

	if config.LabelPrefix == "" {
		config.LabelPrefix = "stream_load_"
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

	var resp *http.Response
	var responseBody []byte
	var streamLoadResp StreamLoadResponse

	// 生成Label
	var label, tmpLabel string
	if s.config.Label != "" {
		label = s.config.Label
	} else {
		// 自动生成label
		label = fmt.Sprintf("%s%d", s.config.LabelPrefix, time.Now().Unix())
		s.logger.Info("Generated stream load label", zap.String("label", label))
		tmpLabel = label // 用来判断是否能重试时改label
	}

	// 最大重试次数
	attempts := uint(3)

	err := retry.New(
		retry.Attempts(attempts),
		retry.Delay(time.Millisecond*100),
		retry.DelayType(retry.RandomDelay),
		retry.MaxDelay(time.Millisecond*800),
		retry.MaxJitter(time.Millisecond*500),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(retryNo uint, err error) {
			s.logger.Info("StreamLoad request failed, retrying", zap.Error(err), zap.Uint("retry_no", retryNo))
		}),
		retry.RetryIf(func(err error) bool {
			if err == nil {
				return false
			}
			s.logger.Info("StreamLoad request failed, retrying", zap.Error(err), zap.Uint("attempts_left", attempts-1))
			attempts--
			return true
		}),
	).Do(func() error {
		var body io.Reader
		var contentEncoding string
		// 处理压缩
		if s.config.EnableGzip {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			if _, err := gz.Write(data); err != nil {
				s.logger.Info("Failed to compress data", zap.Error(err))
				return fmt.Errorf("%w: write fail,%v", ErrCompressFail, err)
			}
			if err := gz.Close(); err != nil {
				s.logger.Info("Failed to close gzip writer", zap.Error(err))
				return fmt.Errorf("%w: close failed %v", ErrCompressFail, err)
			}
			body = &buf
			contentEncoding = "gzip"
			s.logger.Info("Data compressed with gzip", zap.Int("original_size", len(data)), zap.Int("compressed_size", buf.Len()))
		} else {
			body = bytes.NewReader(data)
		}

		// 设置请求头
		req, err := http.NewRequest("PUT", url, body)
		if err != nil {
			s.logger.Info("Failed to create HTTP request", zap.Error(err))
			return fmt.Errorf("%w: %v", ErrCreateRequestFail, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Expect", "100-continue")
		req.Header.Set("Authorization", "Basic "+basicAuth(s.config.Username, s.config.Password))

		req.Header.Set("label", label)

		req.Header.Set("format", "json")
		req.Header.Set("strip_outer_array", "true") // 支持数组格式的JSON数据

		// 设置自定义头部参数
		for headerName, headerValue := range s.config.Headers {
			req.Header.Set(headerName, headerValue)
		}

		if contentEncoding != "" {
			req.Header.Set("Content-Encoding", contentEncoding)
		}

		// 发送请求
		resp, err = s.client.Do(req)
		if err != nil {
			s.logger.Info("Failed to send HTTP request", zap.Error(err))
			return fmt.Errorf("%w: %v", ErrSendFail, err)
		}
		defer resp.Body.Close()

		// 判断非正常状态码，此时应该重试
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%w: need 200 got %d", ErrHttpStatusNotOk, resp.StatusCode)
		} else {
			// 读取响应
			responseBody, err = io.ReadAll(resp.Body)
			if err != nil {
				s.logger.Info("Failed to read response body", zap.Error(err))
				return fmt.Errorf("%w: %v", ErrReadResponseFail, err)
			}
			// 解析响应
			if len(responseBody) > 0 {
				if err = json.Unmarshal(responseBody, &streamLoadResp); err != nil {
					s.logger.Info("Failed to unmarshal response JSON, using default response", zap.Error(err), zap.String("response_body", string(responseBody)))
					return fmt.Errorf("%w: %v", ErrUnmarshalResponseFail, err)
				}
				// 增加判断，如果是返回信息说label重复，且attempts < 3，且tmpLabel==label， 则应给label加个后缀
				if streamLoadResp.Status == "Label Already Exists" && attempts > 0 && tmpLabel == label {
					label = fmt.Sprintf("%s_%d", label, attempts)
					s.logger.Info("Label already exists, retrying with new label", zap.String("label", label))
					return ErrReturnLabelAlreadyExist
				}
			} else {
				s.logger.Info("Received empty response body, assuming success")
				return ErrEmptyResponse
			}
		}

		return nil
	})

	if errors.Is(err, ErrUnmarshalResponseFail) {
		// 如果JSON解析失败，使用默认响应并记录警告
		streamLoadResp = StreamLoadResponse{
			Status:  "Success",
			Label:   s.config.Label,
			Message: fmt.Sprintf("Warning: Could not parse response JSON: %v", err),
		}
	} else if errors.Is(err, ErrEmptyResponse) {
		// 如果响应为空，假设成功并使用默认响应
		streamLoadResp = StreamLoadResponse{
			Status:  "Success",
			Label:   s.config.Label,
			Message: "Empty response received",
		}
	} else if err != nil {
		// 因为retry包会重试，所以这里返回err，未走到streamLoadResp步骤
		s.logger.Error("StreamLoad request failed", zap.Error(err))
		return nil, err
	}

	s.logger.Info("StreamLoad response received", zap.String("status", streamLoadResp.Status), zap.String("label", streamLoadResp.Label), zap.Int64("load_bytes", streamLoadResp.LoadBytes))

	// 检查状态
	// Doris StreamLoad 常见的状态码: 200(成功), 307(临时重定向到BE节点,此时如果是这个状态，则代表请求BE节点未通)
	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("StreamLoad request failed", zap.Int("status_code", resp.StatusCode), zap.Any("headers", resp.Header), zap.String("response", string(responseBody)), zap.Uint("attempts", attempts))
		return &streamLoadResp, fmt.Errorf("%w: request failed with status: %d, response: %s", ErrHttpStatusNotOk, resp.StatusCode, string(responseBody))
	} else {
		s.logger.Info("StreamLoad request succeeded", zap.Int("status_code", resp.StatusCode), zap.String("status", streamLoadResp.Status), zap.String("label", streamLoadResp.Label))
	}

	return &streamLoadResp, nil
}

// basicAuth 生成Basic认证字符串
func basicAuth(username, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return encoded
}
