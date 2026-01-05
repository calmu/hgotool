// Package logrotate provides functionality for rotating log files based on size and time.
package logrotate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RotateConfig 定义轮转配置
type RotateConfig struct {
	// 时间轮转配置
	TimeRotation string // "daily", "hourly", "minutely"

	// 大小轮转配置
	MaxSize    int64 // MB
	MaxBackups int   // 最大备份文件数
	MaxAge     int   // 保留天数
	Compress   bool  // 是否压缩 (暂时不实现压缩功能)

	// 基础配置
	Filename string // 基础文件名
}

// RotateWriter 实现io.WriteCloser接口，支持轮转
type RotateWriter struct {
	config      RotateConfig
	file        *os.File
	currentSize int64
	mu          sync.Mutex

	// 用于时间轮转
	lastRotateTime time.Time
	filePrefix     string
	fileExt        string
}

// NewRotateWriter 创建新的轮转写入器
func NewRotateWriter(config RotateConfig) (*RotateWriter, error) {
	// 解析文件名获取前缀和扩展名
	ext := filepath.Ext(config.Filename)
	prefix := strings.TrimSuffix(config.Filename, ext)

	rw := &RotateWriter{
		config:     config,
		filePrefix: prefix,
		fileExt:    ext,
	}

	// 打开初始文件
	err := rw.openNewFile()
	if err != nil {
		return nil, err
	}

	// 设置初始轮转时间
	rw.lastRotateTime = rw.getRotationTimeBoundary()

	return rw, nil
}

// openNewFile 打开新文件
func (rw *RotateWriter) openNewFile() error {
	// 如果当前文件已打开，先关闭
	if rw.file != nil {
		rw.file.Close()
	}

	// 获取当前时间的文件路径
	currentPath := rw.getCurrentFilePath()

	// 确保目录存在
	dir := filepath.Dir(currentPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 打开文件
	file, err := os.OpenFile(currentPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	rw.file = file

	// 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		rw.currentSize = 0
	} else {
		rw.currentSize = stat.Size()
	}

	return nil
}

// getCurrentFilePath 获取当前时间对应的文件路径
func (rw *RotateWriter) getCurrentFilePath() string {
	now := time.Now()

	var timePart string
	switch rw.config.TimeRotation {
	case "hourly":
		timePart = now.Format("2006-01-02_15") // 年-月-日_时
	case "minutely":
		timePart = now.Format("2006-01-02_15_04") // 年-月-日_时_分
	default: // daily
		timePart = now.Format("2006-01-02") // 年-月-日
	}

	return fmt.Sprintf("%s_%s%s", rw.filePrefix, timePart, rw.fileExt)
}

// getRotationTimeBoundary 获取下一个轮转时间边界
func (rw *RotateWriter) getRotationTimeBoundary() time.Time {
	now := time.Now()
	switch rw.config.TimeRotation {
	case "hourly":
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	case "minutely":
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
	default: // daily
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}
}

// checkRotate 检查是否需要轮转
func (rw *RotateWriter) checkRotate() error {
	now := time.Now()

	// 检查是否需要按时间轮转
	if now.After(rw.lastRotateTime) {
		currentPath := rw.getCurrentFilePath()
		if rw.file == nil || rw.file.Name() != currentPath {
			if err := rw.openNewFile(); err != nil {
				return err
			}
			rw.lastRotateTime = rw.getRotationTimeBoundary()
		}
		return nil
	}

	// 检查是否需要按大小轮转
	maxSizeBytes := rw.config.MaxSize * 1024 * 1024 // 转换为字节
	if maxSizeBytes > 0 && rw.currentSize >= maxSizeBytes {
		if err := rw.openNewFile(); err != nil {
			return err
		}
	}

	return nil
}

// Write 实现io.Writer接口
func (rw *RotateWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// 检查是否需要轮转
	if err := rw.checkRotate(); err != nil {
		return 0, err
	}

	// 写入数据
	n, err = rw.file.Write(p)
	if err == nil {
		rw.currentSize += int64(n)
	}

	return n, err
}

// Sync 同步文件到磁盘
func (rw *RotateWriter) Sync() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Sync()
	}
	return nil
}

// Close 关闭写入器
func (rw *RotateWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		err := rw.file.Close()
		rw.file = nil
		return err
	}
	return nil
}

// Rotate 手动触发轮转
func (rw *RotateWriter) Rotate() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	return rw.openNewFile()
}

// GetLogFilePath 获取当前日志文件路径
func (rw *RotateWriter) GetLogFilePath() string {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Name()
	}
	return ""
}
