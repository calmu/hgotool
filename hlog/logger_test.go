package hlog

import (
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMultiTypeLoggers(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 初始化不同类型的logger
	initialLoggers := map[string]LoggerConfig{
		"access": {
			Level:      "info",
			OutputPath: []string{"./log/access.log"},
			Encoder:    "json",
		},
		"error": {
			Level:      "error",
			OutputPath: []string{"./log/error.log"},
			Encoder:    "console",
		},
		"debug": {
			Level:      "debug",
			OutputPath: []string{"./log/debug.log", "stdout"},
			Encoder:    "console",
		},
	}

	// 初始化logger
	for loggerType, config := range initialLoggers {
		InitLogger(loggerType, config)
	}

	// 测试访问日志
	accessLogger := GetLogger("access")
	accessLogger.Info("Access log message", zap.String("user", "test_user"), zap.String("action", "login"))

	// 测试错误日志
	errorLogger := GetLogger("error")
	errorLogger.Error("Error log message", zap.String("error", "sample error"), zap.Int("code", 500))
	errorLogger.Info("This info should not appear in error log due to level setting") // 这个不会记录，因为error logger只记录error及以上级别

	// 测试调试日志
	debugLogger := GetLogger("debug")
	debugLogger.Debug("Debug log message", zap.String("debug_info", "debug_value"))
	debugLogger.Info("Debug info message", zap.String("info", "info_value"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	files := []string{"./log/access.log", "./log/error.log", "./log/debug.log"}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Log file was not created: %s", file)
		} else {
			t.Logf("Log file created successfully: %s", file)
		}
	}
}

func TestConfigChangeForSpecificLogger(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 初始化logger
	initialLoggers := map[string]LoggerConfig{
		"test": {
			Level:      "info",
			OutputPath: []string{"./log/test_before.log"},
			Encoder:    "console",
		},
	}

	// 初始化logger
	for loggerType, config := range initialLoggers {
		InitLogger(loggerType, config)
	}

	// 获取logger并记录日志
	testLogger := GetLogger("test")
	testLogger.Info("Message before config change", zap.String("status", "before"))

	// 更改特定logger的配置
	newConfig := LoggerConfig{
		Level:      "debug",
		OutputPath: []string{"./log/test_after.log", "./log/test_stdout.log"},
		Encoder:    "json",
	}

	// 直接初始化新配置的logger（因为移除了通道机制）
	InitLogger("test", newConfig)

	// 等待配置更改
	time.Sleep(100 * time.Millisecond)

	// 记录新配置下的日志
	testLogger = GetLogger("test") // 重新获取更新后的logger
	testLogger.Info("Message after config change", zap.String("status", "after"))
	testLogger.Debug("Debug message after config change", zap.String("status", "debug"))

	// 等待日志写入
	time.Sleep(100 * time.Millisecond)

	// 验证新日志文件是否已创建
	newFiles := []string{"./log/test_after.log", "./log/test_stdout.log"}
	for _, file := range newFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("New log file was not created: %s", file)
		} else {
			t.Logf("New log file created successfully: %s", file)
		}
	}
}

func TestGlobalLoggersMap(t *testing.T) {
	// 测试GlobalLoggers映射是否可以被外部访问
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"stdout"},
		Encoder:    "console",
	}

	InitLogger("external", config)

	// 验证外部可以访问GlobalLoggers
	if len(GlobalLoggers) == 0 {
		t.Error("GlobalLoggers map is empty")
	}

	logger := GetLogger("external")
	if logger == nil {
		t.Error("Failed to get external logger")
	}

	logger.Info("Test message from external logger", zap.String("test", "value"))

	// 调用Close方法
	if closer, ok := logger.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestDateRotationLogger(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试按日期分割的日志功能
	rotationConfig := map[string]RotateConfig{
		"date_rotation": {
			Level:        "info",
			Encoder:      "json",
			OutputType:   "file",
			Filename:     "./log/rotated/app.log",
			TimeRotation: "daily", // 按天轮转
			MaxSize:      1,       // 1MB后轮转
			MaxBackups:   3,       // 保留3个备份
			MaxAge:       7,       // 保留7天
		},
	}

	// 初始化轮转logger
	for loggerType, config := range rotationConfig {
		InitRotatingLogger(loggerType, config)
	}

	// 获取logger并记录日志
	rotationLogger := GetLogger("date_rotation")
	rotationLogger.Info("Message with date rotation", zap.String("feature", "date_rotation"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志目录是否已创建
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join("./log/rotated", "app_"+today+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Date-based log file was not created: %s", logFile)
	} else {
		t.Logf("Date-based log file created successfully: %s", logFile)
	}

	// 调用Close方法
	if closer, ok := rotationLogger.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestSizeRotationLogger(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试按大小分割的日志功能
	rotationConfig := map[string]RotateConfig{
		"size_rotation": {
			Level:        "info",
			Encoder:      "json",
			OutputType:   "file",
			Filename:     "./log/sizerotated/app.log",
			TimeRotation: "", // 不按时间轮转 - 但logrotate包仍会添加时间戳
			MaxSize:      1,  // 1MB后轮转
			MaxBackups:   2,  // 保留2个备份
			MaxAge:       5,  // 保留5天
		},
	}

	// 初始化轮转logger
	for loggerType, config := range rotationConfig {
		InitRotatingLogger(loggerType, config)
	}

	// 获取logger并记录日志
	rotationLogger := GetLogger("size_rotation")
	rotationLogger.Info("Message with size rotation", zap.String("feature", "size_rotation"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建 - 对于大小轮转，logrotate包仍会添加时间戳
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join("./log/sizerotated", "app_"+today+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Size-based log file was not created: %s", logFile)
	} else {
		t.Logf("Size-based log file created successfully: %s", logFile)
	}

	// 调用Close方法
	if closer, ok := rotationLogger.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestTimeSizeRotationLogger(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试同时按时间和大小分割的日志功能
	rotationConfig := map[string]RotateConfig{
		"timesize_rotation": {
			Level:        "info",
			Encoder:      "json",
			OutputType:   "file",
			Filename:     "./log/timesizerotated/app.log",
			TimeRotation: "hourly", // 按小时轮转
			MaxSize:      10,       // 10MB后轮转
			MaxBackups:   5,        // 保留5个备份
			MaxAge:       10,       // 保留10天
		},
	}

	// 初始化轮转logger
	for loggerType, config := range rotationConfig {
		InitRotatingLogger(loggerType, config)
	}

	// 获取logger并记录日志
	rotationLogger := GetLogger("timesize_rotation")
	rotationLogger.Info("Message with time and size rotation", zap.String("feature", "time_and_size_rotation"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	now := time.Now()
	timePart := now.Format("2006-01-02_15") // 小时格式
	logFile := filepath.Join("./log/timesizerotated", "app_"+timePart+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Time and size-based log file was not created: %s", logFile)
	} else {
		t.Logf("Time and size-based log file created successfully: %s", logFile)
	}

	// 调用Close方法
	if closer, ok := rotationLogger.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestCloseMethod(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试Close方法
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/test_close.log"},
		Encoder:    "json",
	}

	logger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 记录一些日志
	logger.Info("Test message before close", zap.String("test", "value"))

	// 测试Close方法
	err = logger.Close()
	if err != nil {
		t.Errorf("Close method returned error: %v", err)
	}

	// 验证日志文件是否已创建
	logFile := "./log/test_close.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logFile)
	} else {
		t.Logf("Log file created successfully: %s", logFile)
	}
}

func TestCustomEncoderConfig(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试自定义编码器配置 - 使用自定义时间格式
	customConfig := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/custom_time.log"},
		Encoder:    "console",
		EncoderConfig: &EncoderConfig{
			TimeLayout:  "2006-01-02 15:04:05", // 自定义时间格式
			EncodeLevel: "uppercase",           // 级别大写显示
			TimeKey:     "timestamp",           // 自定义时间字段名
			LevelKey:    "level",               // 自定义级别字段名
		},
	}

	logger, err := NewZapLogger(customConfig)
	if err != nil {
		t.Fatalf("Failed to create logger with custom config: %v", err)
	}

	// 记录日志
	logger.Info("Test message with custom time format", zap.String("test", "custom_format"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/custom_time.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Custom time log file was not created: %s", logFile)
	} else {
		t.Logf("Custom time log file created successfully: %s", logFile)
	}

	// 调用Close方法
	if closer, ok := logger.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestCustomJSONEncoderConfig(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 测试自定义JSON编码器配置
	jsonConfig := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/custom_json.log"},
		Encoder:    "json",
		EncoderConfig: &EncoderConfig{
			TimeLayout:  "2006-01-02T15:04:05Z07:00", // RFC3339格式
			EncodeLevel: "capital",                   // 首字母大写
			TimeKey:     "timestamp",                 // 自定义时间字段名
			LevelKey:    "severity",                  // 自定义级别字段名
			MessageKey:  "message",                   // 自定义消息字段名
		},
	}

	logger, err := NewZapLogger(jsonConfig)
	if err != nil {
		t.Fatalf("Failed to create JSON logger with custom config: %v", err)
	}

	// 记录日志
	logger.Info("Test message with custom JSON format", zap.String("test", "json_format"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/custom_json.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Custom JSON log file was not created: %s", logFile)
	} else {
		t.Logf("Custom JSON log file created successfully: %s", logFile)
	}

	// 调用Close方法
	if closer, ok := logger.(interface{ Close() error }); ok {
		closer.Close()
	}
}
