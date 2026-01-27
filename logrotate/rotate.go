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

	// 只有在时间轮转模式下才设置时间边界
	if config.TimeRotation != "" {
		rw.lastRotateTime = rw.getRotationTimeBoundary().AddDate(0, 0, -1) // 启动时，默认为昨天
	} else {
		// 对于大小轮转，初始化为当前时间，这样就不会触发时间轮转条件
		rw.lastRotateTime = time.Time{}
	}

	return rw, nil
}

// openNewFile 打开新文件
func (rw *RotateWriter) openNewFile() error {
	// 如果当前文件已打开，先关闭
	if rw.file != nil {
		rw.file.Close()
	}

	// 判断是时间轮转还是大小轮转
	if rw.config.TimeRotation != "" {
		// 时间轮转：创建带时间戳的新文件
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
	} else if rw.config.MaxSize > 0 {
		// 大小轮转：重命名当前文件并创建新文件
		if rw.file != nil {
			rw.file.Close()
			rw.file = nil
		}

		// 重命名现有的主文件
		if _, err := os.Stat(rw.config.Filename); err == nil {
			// 按照备份编号顺序重命名文件
			for i := rw.config.MaxBackups; i > 0; i-- {
				oldFile := fmt.Sprintf("%s.%d%s", rw.filePrefix, i, rw.fileExt)
				newFile := fmt.Sprintf("%s.%d%s", rw.filePrefix, i+1, rw.fileExt)

				// 如果存在第i+1个备份文件，则删除它
				if i == rw.config.MaxBackups {
					os.Remove(newFile)
				}

				// 将第i个备份文件重命名为第i+1个
				if _, err := os.Stat(oldFile); err == nil {
					os.Rename(oldFile, newFile)
				}
			}

			// 将当前主文件重命名为第一个备份
			backupFile := fmt.Sprintf("%s.1%s", rw.filePrefix, rw.fileExt)
			os.Rename(rw.config.Filename, backupFile)
		}

		// 创建新的主文件
		file, err := os.OpenFile(rw.config.Filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		rw.file = file
		rw.currentSize = 0
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
		// 清理过期文件
		if rw.config.MaxAge > 0 {
			rw.cleanupExpiredFiles()
		}
		return nil
	}

	// 检查是否需要按大小轮转
	maxSizeBytes := rw.config.MaxSize * 1024 * 1024 // 转换为字节
	if maxSizeBytes > 0 && rw.currentSize >= maxSizeBytes {
		if err := rw.openNewFile(); err != nil {
			return err
		}
		// 清理过期文件
		if rw.config.MaxAge > 0 {
			rw.cleanupExpiredFiles()
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

// cleanupExpiredFiles 清理过期文件
func (rw *RotateWriter) cleanupExpiredFiles() {
	// 当MaxAge为0时，不清理任何文件
	if rw.config.MaxAge <= 0 {
		return
	}

	// 获取目录和文件名模式
	dir := filepath.Dir(rw.config.Filename)
	baseName := filepath.Base(rw.config.Filename)
	ext := filepath.Ext(baseName)
	prefix := strings.TrimSuffix(baseName, ext)

	// 读取目录中的所有文件
	files, err := os.ReadDir(dir)
	if err != nil {
		return // 如果无法读取目录，跳过清理
	}

	// 计算过期时间点
	expireTime := time.Now().AddDate(0, 0, -rw.config.MaxAge)

	for _, file := range files {
		if file.IsDir() {
			continue // 跳过子目录
		}

		fileName := file.Name()

		// 检查是否为时间轮转文件 (前缀_日期.扩展名)
		if strings.HasPrefix(fileName, prefix+"_") && strings.HasSuffix(fileName, ext) {
			// 提取日期部分
			datePartStart := len(prefix) + 1
			datePartEnd := len(fileName) - len(ext)
			if datePartEnd > datePartStart {
				dateStr := fileName[datePartStart:datePartEnd]

				// 尝试解析日期
				var fileTime time.Time
				var parsed bool

				// 尝试不同的日期格式
				layouts := []string{
					"2006-01-02",       // daily
					"2006-01-02_15",    // hourly
					"2006-01-02_15_04", // minutely
				}

				for _, layout := range layouts {
					if t, err := time.Parse(layout, dateStr); err == nil {
						fileTime = t
						parsed = true
						break
					}
				}

				if parsed && fileTime.Before(expireTime) {
					// 删除过期文件
					filePath := filepath.Join(dir, fileName)
					os.Remove(filePath) // 忽略错误
				}
			}
		} else if rw.config.MaxSize > 0 && strings.HasPrefix(fileName, prefix) && ext == filepath.Ext(fileName) {
			// 对于大小轮转，如果启用了MaxBackups限制，处理编号备份文件
			if rw.config.MaxBackups > 0 {
				// 检查是否是原始文件，不是则跳过（因为大小轮转备份文件通常有不同的命名规则）
				if fileName != baseName {
					// 这里我们可以实现基于文件修改时间的清理，如果文件太旧就删除
					filePath := filepath.Join(dir, fileName)
					info, err := file.Info()
					if err == nil {
						if info.ModTime().Before(expireTime) {
							os.Remove(filePath) // 忽略错误
						}
					}
				}
			}
		}
	}
}
