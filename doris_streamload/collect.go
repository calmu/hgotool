package doris_streamload

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/calmu/hgotool/hlog"
	"go.uber.org/zap"
)

// Collect 数据攒批结构体
type Collect struct {
	client       Client
	data         []interface{}
	maxSize      int              // 最大条数
	maxBytes     int              // 最大大小（字节）
	currentBytes int              // 当前大小（字节）
	mutex        sync.Mutex       // 并发安全锁
	callback     func(error)      // 回调函数，用于处理发送结果
	stopChan     chan struct{}    // 停止信号
	logger       hlog.HLoggerBase // 日志记录器
	once         sync.Once
	lastTime     time.Time // 最后一次刷新时间，避免频繁自动刷新
}

// CollectOption 配置Collect选项
type CollectOption func(*Collect)

// WithMaxSize 设置最大攒批条数
func WithMaxSize(size int) CollectOption {
	return func(c *Collect) {
		c.maxSize = size
	}
}

// WithMaxBytes 设置最大攒批字节数
func WithMaxBytes(bytes int) CollectOption {
	return func(c *Collect) {
		c.maxBytes = bytes
	}
}

// WithCallback 设置回调函数
func WithCallback(callback func(error)) CollectOption {
	return func(c *Collect) {
		c.callback = callback
	}
}

// WithAutoFlushInterval 设置自动刷新间隔
func WithAutoFlushInterval(interval time.Duration) CollectOption {
	return func(c *Collect) {
		// 启动周期性定时器自动刷新
		ticker := time.NewTicker(interval)
		// 创建一个停止通道
		go func() {
			for {
				select {
				case <-ticker.C:
					c.mutex.Lock()
					if c.lastTime.Add(interval).Before(time.Now()) {
						c.mutex.Unlock()
					} else {
						c.mutex.Unlock()
						_ = c.Flush() // 忽略错误，因为可能在关闭时触发
					}
				case <-c.stopChan:
					ticker.Stop()
					return
				}
			}
		}()
	}
}

// NewCollect 创建新的Collect实例
func NewCollect(client Client, opts ...CollectOption) *Collect {
	collect := &Collect{
		client:       client,
		data:         make([]interface{}, 0),
		maxSize:      1000,             // 默认最大1000条
		maxBytes:     10 * 1024 * 1024, // 默认最大10MB
		currentBytes: 0,
		stopChan:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(collect)
	}

	// 尝试从client获取日志记录器，否则使用默认记录器
	if streamLoadClient, ok := client.(*StreamLoadClient); ok && streamLoadClient.logger != nil {
		collect.logger = streamLoadClient.logger
	} else {
		// 如果无法获取日志记录器，尝试使用默认记录器
		collect.logger = hlog.GlobalLoggers["default"]
		if collect.logger == nil {
			// 如果没有默认日志记录器，则创建一个
			defaultLogger, _ := hlog.NewZapLogger(hlog.LoggerConfig{
				Level:      "info",
				OutputPath: []string{"stdout"},
				Encoder:    "console",
			})
			collect.logger = defaultLogger
		}
	}

	return collect
}

// Add 添加数据到攒批中
func (c *Collect) Add(data interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 序列化数据以计算大小
	jsonData, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("Failed to marshal data", zap.Error(err))
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	// 检查是否超出限制
	newBytes := c.currentBytes + len(jsonData)

	if len(c.data) >= c.maxSize {
		c.logger.Warn("Batch size limit exceeded", zap.Int("current_size", len(c.data)), zap.Int("max_size", c.maxSize))
		return fmt.Errorf("batch size limit exceeded: max %d items", c.maxSize)
	}

	if newBytes > c.maxBytes {
		c.logger.Warn("Batch byte limit exceeded", zap.Int("current_bytes", newBytes), zap.Int("max_bytes", c.maxBytes))
		return fmt.Errorf("batch byte limit exceeded: max %d bytes", c.maxBytes)
	}

	// 添加数据
	c.data = append(c.data, data)
	c.currentBytes = newBytes

	c.logger.Info("Data added to batch", zap.Int("batch_size", len(c.data)), zap.Int("batch_bytes", c.currentBytes))

	// 检查是否达到阈值，如果是则自动发送
	if len(c.data) >= c.maxSize || newBytes >= c.maxBytes {
		c.logger.Info("Batch threshold reached, auto-flushing", zap.Int("size_threshold", c.maxSize), zap.Int("bytes_threshold", c.maxBytes))
		go func() {
			_ = c.Flush()
		}()
	}

	return nil
}

// Len 返回当前攒批中的数据条数
func (c *Collect) Len() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return len(c.data)
}

// Size 返回当前攒批的大小（字节数）
func (c *Collect) Size() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.currentBytes
}

// Flush 发送攒批数据到Doris并清空当前批次
func (c *Collect) Flush() error {
	c.mutex.Lock()

	c.lastTime = time.Now()

	// 复制数据并清空原数据
	dataCopy := make([]interface{}, len(c.data))
	copy(dataCopy, c.data)

	// 清空当前批次
	c.data = c.data[:0]
	c.currentBytes = 0

	c.mutex.Unlock()

	// 如果没有数据，则直接返回
	if len(dataCopy) == 0 {
		c.logger.Info("No data to flush, skipping")
		return nil
	}

	c.logger.Info("Flushing batch data", zap.Int("batch_size", len(dataCopy)), zap.Int("batch_bytes", len(dataCopy)))

	// 将数据序列化为JSON数组
	jsonData, err := json.Marshal(dataCopy)
	if err != nil {
		c.logger.Error("Failed to marshal batch data", zap.Error(err))
		if c.callback != nil {
			c.callback(fmt.Errorf("failed to marshal batch data: %v", err))
		}
		return fmt.Errorf("failed to marshal batch data: %v", err)
	}

	// 发送到Doris
	resp, err := c.client.Load(jsonData)
	if err != nil {
		c.logger.Error("Failed to load data to Doris", zap.Error(err), zap.String("data", string(jsonData)))
		if c.callback != nil {
			c.callback(err)
		}
		return fmt.Errorf("failed to load data to Doris: %v", err)
	}

	// 检查响应状态
	if resp.Status != "Success" && resp.Status != "" {
		errMsg := fmt.Sprintf("stream load failed with status: %s, message: %s", resp.Status, resp.Message)
		c.logger.Warn("Stream load failed", zap.String("status", resp.Status), zap.String("message", resp.Message), zap.String("data", string(jsonData)))
		if c.callback != nil {
			c.callback(fmt.Errorf(errMsg))
		}
		return fmt.Errorf(errMsg)
	}

	c.logger.Info("Successfully flushed batch data", zap.String("status", resp.Status), zap.String("label", resp.Label))

	// 成功回调
	if c.callback != nil {
		c.callback(nil)
	}

	return nil
}

// Close 关闭Collect，发送剩余数据并清理资源
func (c *Collect) Close() error {
	c.logger.Info("Closing Collect, flushing remaining data")

	// 发送剩余数据
	err := c.Flush()

	c.once.Do(func() {
		close(c.stopChan)
	})

	c.logger.Info("Collect closed successfully")
	return err
}
