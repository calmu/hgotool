// Package hlog
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-5 10:34
//
// --------------------------------------------
package hlog

import (
	"context"
	"go.uber.org/zap"
	"time"

	"gorm.io/gorm/logger"
)

type HLoggerBase interface {
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type HLogger interface {
	HLoggerBase
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Close() error
}

// GormLoggerInterface GORM Logger接口定义
type GormLoggerInterface interface {
	LogMode(level int) GormLoggerInterface
	Info(ctx context.Context, msg string, data ...interface{})
	Warn(ctx context.Context, msg string, data ...interface{})
	Error(ctx context.Context, msg string, data ...interface{})
	Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}

// 实现GORM logger.Interface
type gormLogger struct {
	Logger                    HLogger
	config                    *LoggerConfig
	rotateConfig              *RotateConfig
	SlowThreshold             time.Duration   // 慢查询阈值
	LogLevel                  logger.LogLevel // GORM日志级别
	IgnoreRecordNotFoundError bool            // 是否忽略记录未找到错误
	Context                   context.Context
}
