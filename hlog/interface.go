// Package hlog
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-5 10:34
//
// --------------------------------------------
package hlog

import "go.uber.org/zap"

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
