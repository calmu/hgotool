package hlog

import (
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestEncoderConfigHideFields 测试编码器配置中的字段隐藏功能
func TestEncoderConfigHideFields(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试隐藏caller字段
	t.Run("HideCaller", func(t *testing.T) {
		config := LoggerConfig{
			Level:      "info",
			OutputPath: []string{"./log/test_hide_caller.log"},
			Encoder:    "json",
			EncoderConfig: &EncoderConfig{
				HideCaller: true, // 隐藏调用者信息
				TimeLayout: "2006-01-02 15:04:05",
			},
		}

		logger, err := NewZapLogger(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without caller", zap.String("test", "value"))

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试隐藏level字段
	t.Run("HideLevel", func(t *testing.T) {
		config := LoggerConfig{
			Level:      "info",
			OutputPath: []string{"./log/test_hide_level.log"},
			Encoder:    "json",
			EncoderConfig: &EncoderConfig{
				HideLevel:  true, // 隐藏日志级别
				TimeLayout: "2006-01-02 15:04:05",
			},
		}

		logger, err := NewZapLogger(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without level", zap.String("test", "value"))

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试隐藏time字段
	t.Run("HideTime", func(t *testing.T) {
		config := LoggerConfig{
			Level:      "info",
			OutputPath: []string{"./log/test_hide_time.log"},
			Encoder:    "json",
			EncoderConfig: &EncoderConfig{
				HideTime:   true, // 隐藏时间戳
				TimeLayout: "2006-01-02 15:04:05",
			},
		}

		logger, err := NewZapLogger(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without time", zap.String("test", "value"))

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试隐藏name字段
	t.Run("HideName", func(t *testing.T) {
		config := LoggerConfig{
			Level:      "info",
			OutputPath: []string{"./log/test_hide_name.log"},
			Encoder:    "json",
			EncoderConfig: &EncoderConfig{
				HideName:   true, // 隐藏名称字段
				TimeLayout: "2006-01-02 15:04:05",
			},
		}

		logger, err := NewZapLogger(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without name", zap.String("test", "value"))

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 测试组合隐藏多个字段
	t.Run("HideMultipleFields", func(t *testing.T) {
		config := LoggerConfig{
			Level:      "info",
			OutputPath: []string{"./log/test_hide_multiple.log"},
			Encoder:    "json",
			EncoderConfig: &EncoderConfig{
				HideCaller: true, // 隐藏调用者信息
				HideLevel:  true, // 隐藏日志级别
				HideTime:   true, // 隐藏时间戳
				TimeLayout: "2006-01-02 15:04:05",
			},
		}

		logger, err := NewZapLogger(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		logger.Info("Test message without caller, level and time", zap.String("test", "value"))

		// 等待确保日志写入文件
		time.Sleep(100 * time.Millisecond)
	})

	// 验证日志文件是否已创建
	logFiles := []string{
		"./log/test_hide_caller.log",
		"./log/test_hide_level.log",
		"./log/test_hide_time.log",
		"./log/test_hide_name.log",
		"./log/test_hide_multiple.log",
	}

	for _, file := range logFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Log file was not created: %s", file)
		} else {
			t.Logf("Log file created successfully: %s", file)
		}
	}
}

// TestEncoderConfigWithCustomKeys 测试自定义键名与隐藏字段的组合使用
func TestEncoderConfigWithCustomKeysAndHide(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/test_custom_keys_with_hide.log"},
		Encoder:    "json",
		EncoderConfig: &EncoderConfig{
			// 自定义键名
			TimeKey:    "timestamp",
			LevelKey:   "severity",
			CallerKey:  "source",
			MessageKey: "message",
			// 但隐藏某些字段
			HideCaller: true, // 即使定义了自定义CallerKey，也会隐藏
			HideTime:   true, // 即使定义了自定义TimeKey，也会隐藏
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	logger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	logger.Info("Test message with custom keys and hidden fields", zap.String("test", "value"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/test_custom_keys_with_hide.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logFile)
	} else {
		t.Logf("Log file created successfully: %s", logFile)
	}
}
