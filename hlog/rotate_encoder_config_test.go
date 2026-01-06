package hlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRotateEncoderConfigHideFields 测试轮转日志配置中的字段隐藏功能
func TestRotateEncoderConfigHideFields(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试在轮转配置中隐藏caller字段
	t.Run("HideCallerInRotateConfig", func(t *testing.T) {
		rotateConfig := RotateConfig{
			Level:      "info",
			Encoder:    "json",
			OutputType: "file",
			Filename:   "./log/rotate_test_hide_caller.log",
			EncoderConfig: &EncoderConfig{
				HideCaller: true, // 隐藏调用者信息
				TimeLayout: "2006-01-02 15:04:05",
			},
			MaxSize:    10, // 10MB
			MaxBackups: 3,  // 保留3个备份
			MaxAge:     30, // 保留30天
			Compress:   false,
		}

		logger, err := NewRotatingLogger(rotateConfig)
		if err != nil {
			t.Fatalf("Failed to create rotating logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without caller in rotating logger")

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试在轮转配置中隐藏level字段
	t.Run("HideLevelInRotateConfig", func(t *testing.T) {
		rotateConfig := RotateConfig{
			Level:      "info",
			Encoder:    "json",
			OutputType: "file",
			Filename:   "./log/rotate_test_hide_level.log",
			EncoderConfig: &EncoderConfig{
				HideLevel:  true, // 隐藏日志级别
				TimeLayout: "2006-01-02 15:04:05",
			},
			MaxSize:    10, // 10MB
			MaxBackups: 3,  // 保留3个备份
			MaxAge:     30, // 保留30天
			Compress:   false,
		}

		logger, err := NewRotatingLogger(rotateConfig)
		if err != nil {
			t.Fatalf("Failed to create rotating logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without level in rotating logger")

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试在轮转配置中隐藏time字段
	t.Run("HideTimeInRotateConfig", func(t *testing.T) {
		rotateConfig := RotateConfig{
			Level:      "info",
			Encoder:    "json",
			OutputType: "file",
			Filename:   "./log/rotate_test_hide_time.log",
			EncoderConfig: &EncoderConfig{
				HideTime:   true, // 隐藏时间戳
				TimeLayout: "2006-01-02 15:04:05",
			},
			MaxSize:    10, // 10MB
			MaxBackups: 3,  // 保留3个备份
			MaxAge:     30, // 保留30天
			Compress:   false,
		}

		logger, err := NewRotatingLogger(rotateConfig)
		if err != nil {
			t.Fatalf("Failed to create rotating logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without time in rotating logger")

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试在轮转配置中组合隐藏多个字段
	t.Run("HideMultipleFieldsInRotateConfig", func(t *testing.T) {
		rotateConfig := RotateConfig{
			Level:      "info",
			Encoder:    "json",
			OutputType: "file",
			Filename:   "./log/rotate_test_hide_multiple.log",
			EncoderConfig: &EncoderConfig{
				HideCaller: true, // 隐藏调用者信息
				HideLevel:  true, // 隐藏日志级别
				HideTime:   true, // 隐藏时间戳
				TimeLayout: "2006-01-02 15:04:05",
			},
			MaxSize:    10, // 10MB
			MaxBackups: 3,  // 保留3个备份
			MaxAge:     30, // 保留30天
			Compress:   false,
		}

		logger, err := NewRotatingLogger(rotateConfig)
		if err != nil {
			t.Fatalf("Failed to create rotating logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without caller, level and time in rotating logger")

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 验证日志文件是否已创建
	today := time.Now().Format("2006-01-02")
	logFiles := []string{
		"./log/rotate_test_hide_caller_" + today + ".log",
		"./log/rotate_test_hide_level_" + today + ".log",
		"./log/rotate_test_hide_time_" + today + ".log",
		"./log/rotate_test_hide_multiple_" + today + ".log",
	}

	// 检查轮转日志文件（根据logrotate的实现，文件名可能包含日期）
	for _, file := range logFiles {
		// 检查文件是否存在，如果不存在，尝试不带日期的文件名
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// 尝试不带日期的文件名
			baseName := filepath.Base(file)
			ext := filepath.Ext(baseName)
			baseWithoutExt := baseName[:len(baseName)-len(ext)]
			// 移除日期部分
			possibleName := filepath.Join(filepath.Dir(file), baseWithoutExt[:len(baseWithoutExt)-len("_"+today)]+ext)
			if _, err := os.Stat(possibleName); os.IsNotExist(err) {
				t.Logf("Log file was not created with expected name: %s", file)
			} else {
				t.Logf("Log file created successfully: %s", possibleName)
			}
		} else {
			t.Logf("Log file created successfully: %s", file)
		}
	}
}
