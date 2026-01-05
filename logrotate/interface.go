package logrotate

import "io"

// WriteSyncer 接口，用于与zap集成
type WriteSyncer interface {
	io.Writer
	Sync() error
}

// RotateWriter 接口，提供日志轮转功能
type RotateWriterInterface interface {
	WriteSyncer
	io.Closer
	Rotate() error
	GetLogFilePath() string
}
