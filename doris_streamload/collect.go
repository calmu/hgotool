package doris_streamload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/calmu/hgotool/hlog"
	"go.uber.org/zap"
)

// Collect 数据攒批结构体
type Collect struct {
	client             Client
	data               [][]byte
	maxSize            int              // 最大条数
	maxBytes           int              // 最大大小（字节）
	currentBytes       int              // 当前大小（字节）
	rwMutex            sync.RWMutex     // 并发安全锁
	callback           func(error)      // 回调函数，用于处理发送结果
	stopChan           chan struct{}    // 停止信号
	logger             hlog.HLoggerBase // 日志记录器
	once               sync.Once        // 保证只关闭一次
	lastTime           time.Time        // 最后一次刷新时间，避免频繁自动刷新
	buf                bytes.Buffer     // 用来攒批数据
	autoFlushWhenLimit bool             // 是否在达到阈值时自动刷新
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

// WithAutoFlushWhenLimit 设置是否在达到阈值时自动刷新，默认为true
func WithAutoFlushWhenLimit(autoFlushWhenLimit bool) CollectOption {
	return func(c *Collect) {
		c.autoFlushWhenLimit = autoFlushWhenLimit
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
					c.rwMutex.Lock()
					if c.lastTime.Add(interval).Before(time.Now()) {
						c.rwMutex.Unlock()
					} else {
						c.rwMutex.Unlock()
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
		client:             client,
		data:               make([][]byte, 0),
		maxSize:            1000,             // 默认最大1000条
		maxBytes:           10 * 1024 * 1024, // 默认最大10MB
		currentBytes:       0,
		stopChan:           make(chan struct{}),
		autoFlushWhenLimit: true,
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
	// 序列化数据以计算大小
	jsonData, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("Failed to marshal data", zap.Error(err))
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	return c.addRowByte(jsonData)
}

func (c *Collect) addRowByte(data []byte) error {
Loop:
	// 检查是否超出限制
	newBytes := c.Size() + len(data)
	if newBytes > c.maxBytes {
		if c.autoFlushWhenLimit {
			_ = c.Flush()
			goto Loop
		} else {
			c.logger.Info("Batch byte limit exceeded", zap.Int("current_bytes", newBytes), zap.Int("max_bytes", c.maxBytes))
			return fmt.Errorf("%w: max %d bytes, current %d", ErrLimitMaxBytes, c.maxBytes, newBytes)
		}
	}

	dataLen := c.Len()
	if dataLen >= c.maxSize {
		if c.autoFlushWhenLimit {
			_ = c.Flush()
			goto Loop
		} else {
			c.logger.Info("Batch size limit exceeded", zap.Int("current_size", dataLen), zap.Int("max_size", c.maxSize))
			return fmt.Errorf("%w: max %d items, current %d", ErrLimitMaxSize, c.maxSize, dataLen)
		}
	}

	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

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

func (c *Collect) AddRowByte(data []byte) error {
	return c.addRowByte(data)
}

func (c *Collect) AddListByte(list [][]byte) error {
	for _, data := range list {
		if err := c.addRowByte(data); err != nil {
			return err
		}
	}
	return nil
}

// Len 返回当前攒批中的数据条数
func (c *Collect) Len() int {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return len(c.data)
}

// Size 返回当前攒批的大小（字节数）
func (c *Collect) Size() int {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return c.currentBytes
}

// buildBatchData 构建攒批数据
func (c *Collect) buildBatchData() []byte {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	c.lastTime = time.Now()

	c.buf.Reset()
	c.buf.WriteString("[")
	for i, data := range c.data {
		if i > 0 {
			c.buf.WriteString(",")
		}
		c.buf.Write(data)
	}
	c.buf.WriteString("]")
	// 复制数据并清空原数据
	dataCopy := make([]byte, c.buf.Len())
	copy(dataCopy, c.buf.Bytes())
	c.buf.Reset()

	// 清空当前批次
	c.data = make([][]byte, 0)
	c.currentBytes = 0

	return dataCopy
}

// Flush 发送攒批数据到Doris并清空当前批次
func (c *Collect) Flush() error {
	// 如果没有数据，则直接返回
	dataLen := c.Len()
	if dataLen == 0 {
		c.logger.Info("No data to flush, skipping")
		return nil
	}
	dataCopy := c.buildBatchData()
	c.logger.Info("Flushing batch data", zap.Int("batch_size", dataLen), zap.Int("batch_bytes", len(dataCopy)))

	// 发送到Doris
	resp, err := c.client.Load(dataCopy)
	if err != nil {
		c.logger.Error("Failed to load data to Doris", zap.Error(err), zap.String("data", string(dataCopy)))
		if c.callback != nil {
			c.callback(err)
		}
		return err
	}

	// 检查响应状态
	if resp.Status != StreamLoadResponseLabelSuccess && resp.Status != "" {
		errMsg := fmt.Sprintf("stream load failed with status: %s, message: %s", resp.Status, resp.Message)
		c.logger.Warn("Stream load failed", zap.String("status", resp.Status), zap.String("message", resp.Message), zap.String("data", string(dataCopy)))
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
