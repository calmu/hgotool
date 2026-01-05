package hlog

import (
	"github.com/calmu/hgotool/logrotate" // 引入我们自己的轮转包
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"sync"
)

// zapLogger 是基于zap的HLogger接口实现
type zapLogger struct {
	logger *zap.Logger
}

// Warn 实现Warn方法
func (zl *zapLogger) Warn(msg string, fields ...zap.Field) {
	zl.logger.Warn(msg, fields...)
}

// Error 实现Error方法
func (zl *zapLogger) Error(msg string, fields ...zap.Field) {
	zl.logger.Error(msg, fields...)
}

// Info 实现Info方法
func (zl *zapLogger) Info(msg string, fields ...zap.Field) {
	zl.logger.Info(msg, fields...)
}

// Debug 实现Debug方法
func (zl *zapLogger) Debug(msg string, fields ...zap.Field) {
	zl.logger.Debug(msg, fields...)
}

// Fatal 实现Fatal方法
func (zl *zapLogger) Fatal(msg string, fields ...zap.Field) {
	zl.logger.Fatal(msg, fields...)
}

// Close 关闭logger，释放资源
func (zl *zapLogger) Close() error {
	return zl.logger.Sync()
}

// EncoderConfig 编码器配置结构
type EncoderConfig struct {
	TimeKey        string // 时间字段的键名，默认为 "ts"
	LevelKey       string // 级别字段的键名，默认为 "level"
	NameKey        string // 名称字段的键名，默认为 "logger"
	CallerKey      string // 调用者字段的键名，默认为 "caller"
	MessageKey     string // 消息字段的键名，默认为 "msg"
	StacktraceKey  string // 堆栈跟踪字段的键名，默认为 "stacktrace"
	LineEnding     string // 行结束符，默认为 "\n"
	EncodeLevel    string // 级别编码方式: "lowercase", "uppercase", "capital", "capitalColor", "color"
	EncodeTime     string // 时间编码方式: "iso8601", "millis", "nanos", "epoch", "rfc3339", "rfc3339nano"
	EncodeDuration string // 持续时间编码方式: "seconds", "nanos", "string"
	EncodeCaller   string // 调用者编码方式: "full", "short"
	TimeLayout     string // 自定义时间格式布局，例如 "2006-01-02 15:04:05"
}

// LoggerConfig 日志配置结构
type LoggerConfig struct {
	Level         string         // 日志级别: debug, info, warn, error, dpanic, panic, fatal
	OutputPath    []string       // 输出路径
	Encoder       string         // 编码器: json, console
	EncoderConfig *EncoderConfig // 编码器详细配置
}

// RotateConfig 定义轮转配置
type RotateConfig struct {
	// 时间轮转配置
	TimeRotation string // "daily", "hourly", "minutely"

	// 大小轮转配置
	MaxSize    int64 // MB
	MaxBackups int   // 最大备份文件数
	MaxAge     int   // 保留天数
	Compress   bool  // 是否压缩

	// 基础配置
	Filename      string         // 基础文件名
	Level         string         // 日志级别
	Encoder       string         // 编码器: json, console
	EncoderConfig *EncoderConfig // 编码器详细配置
	OutputType    string         // 输出类型: file, stdout, 或两者
}

// 全局logger映射，用于存储不同类型的logger
var (
	GlobalLoggers = make(map[string]HLogger)
	loggersMutex  sync.RWMutex
)

// GetLogger 获取指定类型的全局logger实例
func GetLogger(loggerType string) HLogger {
	loggersMutex.RLock()
	defer loggersMutex.RUnlock()

	logger, exists := GlobalLoggers[loggerType]
	if !exists {
		// 如果不存在，则返回默认logger
		defaultLogger := createDefaultLogger()
		return defaultLogger
	}

	return logger
}

// SetLogger 设置指定类型的全局logger
func SetLogger(loggerType string, logger HLogger) {
	loggersMutex.Lock()
	defer loggersMutex.Unlock()

	GlobalLoggers[loggerType] = logger
}

// createDefaultLogger 创建默认logger
func createDefaultLogger() HLogger {
	config := LoggerConfig{
		Level:         "info",
		OutputPath:    []string{"stdout"},
		Encoder:       "console",
		EncoderConfig: nil, // 使用默认编码器配置
	}

	logger, _ := NewZapLogger(config)
	return logger
}

// NewZapLogger 根据普通配置创建新的zap logger
func NewZapLogger(config LoggerConfig) (HLogger, error) {
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "dpanic":
		level = zapcore.DPanicLevel
	case "panic":
		level = zapcore.PanicLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	var encoder zapcore.Encoder
	if config.Encoder == "json" {
		encoderConfig := getEncoderConfig(config.EncoderConfig, "json")
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig := getEncoderConfig(config.EncoderConfig, "console")
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	writeSyncer := zapcore.NewMultiWriteSyncer(getWriteSyncers(config.OutputPath)...)
	core := zapcore.NewCore(encoder, writeSyncer, level)

	loggerInstance := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &zapLogger{
		logger: loggerInstance,
	}, nil
}

// getWriteSyncers 根据路径创建WriteSyncer
func getWriteSyncers(paths []string) []zapcore.WriteSyncer {
	var writeSyncers []zapcore.WriteSyncer
	for _, path := range paths {
		if path == "stdout" {
			writeSyncers = append(writeSyncers, zapcore.AddSync(zapcore.Lock(os.Stdout)))
		} else {
			// 确保目录存在
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				// 如果创建目录失败，仍然使用标准输出
				writeSyncers = append(writeSyncers, zapcore.AddSync(zapcore.Lock(os.Stdout)))
				continue
			}

			file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				// 如果打开文件失败，仍然使用标准输出
				writeSyncers = append(writeSyncers, zapcore.AddSync(zapcore.Lock(os.Stdout)))
			} else {
				writeSyncers = append(writeSyncers, zapcore.AddSync(file))
			}
		}
	}
	return writeSyncers
}

// NewRotatingLogger 创建支持轮转的日志记录器
func NewRotatingLogger(rotateConfig RotateConfig) (HLogger, error) {
	var level zapcore.Level
	switch rotateConfig.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "dpanic":
		level = zapcore.DPanicLevel
	case "panic":
		level = zapcore.PanicLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	var encoder zapcore.Encoder
	if rotateConfig.Encoder == "json" {
		encoderConfig := getEncoderConfig(rotateConfig.EncoderConfig, "json")
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig := getEncoderConfig(rotateConfig.EncoderConfig, "console")
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writeSyncers []zapcore.WriteSyncer

	// 添加标准输出
	if rotateConfig.OutputType == "stdout" || rotateConfig.OutputType == "both" {
		writeSyncers = append(writeSyncers, zapcore.AddSync(zapcore.Lock(os.Stdout)))
	}

	// 添加轮转文件输出
	if rotateConfig.OutputType == "file" || rotateConfig.OutputType == "both" {
		// 确保目录存在 - logrotate包内部会处理目录创建
		rotatingConfig := logrotate.RotateConfig{
			TimeRotation: rotateConfig.TimeRotation,
			MaxSize:      rotateConfig.MaxSize,
			MaxBackups:   rotateConfig.MaxBackups,
			MaxAge:       rotateConfig.MaxAge,
			Compress:     rotateConfig.Compress,
			Filename:     rotateConfig.Filename,
		}

		rotatingWriter, err := logrotate.NewRotateWriter(rotatingConfig)
		if err != nil {
			return nil, err
		}

		writeSyncers = append(writeSyncers, zapcore.AddSync(rotatingWriter))
	}

	writeSyncer := zapcore.NewMultiWriteSyncer(writeSyncers...)
	core := zapcore.NewCore(encoder, writeSyncer, level)

	loggerInstance := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &zapLogger{
		logger: loggerInstance,
	}, nil
}

// InitLogger 初始化指定类型的logger
func InitLogger(loggerType string, config LoggerConfig) {
	logger, err := NewZapLogger(config)
	if err == nil {
		SetLogger(loggerType, logger)
	}
}

// InitRotatingLogger 初始化指定类型的轮转logger
func InitRotatingLogger(loggerType string, rotateConfig RotateConfig) {
	logger, err := NewRotatingLogger(rotateConfig)
	if err == nil {
		SetLogger(loggerType, logger)
	}
}

// getEncoderConfig 根据配置获取编码器配置
func getEncoderConfig(config *EncoderConfig, encoderType string) zapcore.EncoderConfig {
	// 根据编码器类型设置默认配置
	var encoderConfig zapcore.EncoderConfig
	if encoderType == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 如果没有提供自定义配置，则返回默认配置
	if config == nil {
		return encoderConfig
	}

	// 应用自定义配置
	if config.TimeKey != "" {
		encoderConfig.TimeKey = config.TimeKey
	}
	if config.LevelKey != "" {
		encoderConfig.LevelKey = config.LevelKey
	}
	if config.NameKey != "" {
		encoderConfig.NameKey = config.NameKey
	}
	if config.CallerKey != "" {
		encoderConfig.CallerKey = config.CallerKey
	}
	if config.MessageKey != "" {
		encoderConfig.MessageKey = config.MessageKey
	}
	if config.StacktraceKey != "" {
		encoderConfig.StacktraceKey = config.StacktraceKey
	}
	if config.LineEnding != "" {
		encoderConfig.LineEnding = config.LineEnding
	}

	// 设置时间编码格式
	if config.EncodeTime != "" {
		switch config.EncodeTime {
		case "iso8601":
			encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		case "millis":
			encoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder
		case "nanos":
			encoderConfig.EncodeTime = zapcore.EpochNanosTimeEncoder
		case "epoch":
			encoderConfig.EncodeTime = zapcore.EpochTimeEncoder
		case "rfc3339":
			encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		case "rfc3339nano":
			encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
		default:
			// 默认使用 ISO8601
			encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}
	} else if config.TimeLayout != "" {
		// 如果提供了自定义时间格式，则使用自定义格式
		customLayout := config.TimeLayout
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(customLayout)
	} else {
		// 使用默认时间格式
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// 设置级别编码格式
	if config.EncodeLevel != "" {
		switch config.EncodeLevel {
		case "lowercase":
			encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		case "uppercase":
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		case "capital":
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		case "capitalColor":
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		case "color":
			encoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
		default:
			encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		}
	}

	// 设置持续时间编码格式
	if config.EncodeDuration != "" {
		switch config.EncodeDuration {
		case "seconds":
			encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
		case "nanos":
			encoderConfig.EncodeDuration = zapcore.NanosDurationEncoder
		case "string":
			encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		default:
			encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
		}
	}

	// 设置调用者编码格式
	if config.EncodeCaller != "" {
		switch config.EncodeCaller {
		case "full":
			encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
		case "short":
			encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		default:
			encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		}
	}

	return encoderConfig
}
