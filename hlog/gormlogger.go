// Package hlog
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-6 17:00
//
// --------------------------------------------
package hlog

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

// NewGormLogger 创建一个新的GORM日志适配器
func NewGormLogger(hlogger HLogger, config *logger.Config) logger.Interface {
	if config == nil {
		// 使用默认配置
		config = &logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		}
	}

	gLogger := &gormLogger{
		Logger:                    hlogger,
		SlowThreshold:             config.SlowThreshold,
		LogLevel:                  config.LogLevel,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
		Context:                   context.Background(),
	}

	// 获取zapLogger的配置
	if _, ok := hlogger.(*zapLogger); ok {
		if hlogger.(*zapLogger).config != nil {
			gLogger.config = hlogger.(*zapLogger).config
		}
		if hlogger.(*zapLogger).rotateConfig != nil {
			gLogger.rotateConfig = hlogger.(*zapLogger).rotateConfig
		}
	}

	return gLogger
}

// LogMode 设置日志级别
func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *g
	newLogger.LogLevel = level
	return &newLogger
}

// Info 记录信息日志
func (g *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= logger.Info {
		formattedMsg := fmt.Sprintf(msg, data...)
		g.Logger.Info(formattedMsg)
	}
}

// Warn 记录警告日志
func (g *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= logger.Warn {
		formattedMsg := fmt.Sprintf(msg, data...)
		g.Logger.Warn(formattedMsg)
	}
}

// Error 记录错误日志
func (g *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= logger.Error {
		formattedMsg := fmt.Sprintf(msg, data...)
		g.Logger.Error(formattedMsg)
	}
}

// Trace 记录SQL执行追踪日志
func (g *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if g.LogLevel < logger.Info {
		return
	}

	elapsed := time.Since(begin)
	var consoleFlag bool
	if g.config != nil && g.config.Encoder == "console" {
		consoleFlag = true
	}
	if !consoleFlag && g.rotateConfig != nil && g.rotateConfig.Encoder == "console" {
		consoleFlag = true
	}
	switch {
	case err != nil && g.LogLevel >= logger.Error && (!g.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		// 记录错误
		sql, rows := fc()
		if consoleFlag {
			g.Logger.Error(
				fmt.Sprintf("SQL Error: %v \r\n[%v] [rows: %v] %v", err, elapsed, rows, sql),
			)
		} else {
			g.Logger.Error("SQL Error",
				zap.String("sql", sql),
				zap.Int64("rows", rows),
				zap.Duration("elapsed", elapsed),
				zap.Error(err),
			)
		}

	case elapsed > g.SlowThreshold && g.LogLevel >= logger.Warn:
		// 记录慢查询
		sql, rows := fc()
		if consoleFlag {
			g.Logger.Warn(
				fmt.Sprintf("SLOW SQL > %v \r\n[%v] [rows: %v] %v", g.SlowThreshold, elapsed, rows, sql),
			)
		} else {
			g.Logger.Warn("SLOW SQL",
				zap.String("sql", sql),
				zap.Int64("rows", rows),
				zap.Duration("elapsed", elapsed),
				zap.Float64("threshold_ms", g.SlowThreshold.Seconds()*1000),
			)
		}
	case g.LogLevel == logger.Info:
		// 记录所有SQL
		sql, rows := fc()
		if consoleFlag {
			g.Logger.Info(
				fmt.Sprintf("SQL \r\n[%v] [rows: %v] %v", elapsed, rows, sql),
			)
		} else {
			g.Logger.Info("SQL",
				zap.String("sql", sql),
				zap.Int64("rows", rows),
				zap.Duration("elapsed", elapsed),
			)
		}
	}
}
