package logrotate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMaxAgeCleanup 测试MaxAge配置是否生效
func TestMaxAgeCleanup(t *testing.T) {
	// 创建临时目录
	tempDir := "./test_logs"
	os.RemoveAll(tempDir) // 清理之前可能存在的测试目录
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir) // 测试完成后清理

	// 创建一个带日期的过期文件（模拟3天前的文件）
	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	oldFileName := filepath.Join(tempDir, fmt.Sprintf("test_log_%s.log", threeDaysAgo.Format("2006-01-02")))
	oldFile, err := os.Create(oldFileName)
	if err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldFile.WriteString("This is an old log file from 3 days ago")
	oldFile.Close()

	// 创建一个较近的文件（模拟1天前的文件）
	oneDayAgo := time.Now().AddDate(0, 0, -1)
	recentFileName := filepath.Join(tempDir, fmt.Sprintf("test_log_%s.log", oneDayAgo.Format("2006-01-02")))
	recentFile, err := os.Create(recentFileName)
	if err != nil {
		t.Fatalf("Failed to create recent log file: %v", err)
	}
	recentFile.WriteString("This is a recent log file from 1 day ago")
	recentFile.Close()

	// 创建RotateWriter配置，设置MaxAge为2天
	config := RotateConfig{
		Filename:     filepath.Join(tempDir, "test_log.log"),
		MaxAge:       2,  // 只保留2天内的文件
		MaxSize:      10, // 10MB
		TimeRotation: "daily",
	}

	writer, err := NewRotateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer writer.Close()

	// 手动调用清理函数
	writer.cleanupExpiredFiles()

	// 检查过期文件是否被删除
	if _, err := os.Stat(oldFileName); !os.IsNotExist(err) {
		t.Errorf("Old file %s should have been deleted", oldFileName)
	}

	// 检查未过期文件是否还存在
	if _, err := os.Stat(recentFileName); os.IsNotExist(err) {
		t.Errorf("Recent file %s should still exist", recentFileName)
	}

	// 检查新创建的当前日志文件是否存在
	currentLogFile := writer.GetLogFilePath()
	if currentLogFile == "" {
		t.Error("Current log file path should not be empty")
	} else {
		if _, err := os.Stat(currentLogFile); os.IsNotExist(err) {
			t.Errorf("Current log file %s should exist", currentLogFile)
		}
	}
}

// TestMaxAgeZero 测试MaxAge为0时不清理文件
func TestMaxAgeZero(t *testing.T) {
	// 创建临时目录
	tempDir := "./test_logs_zero"
	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// 创建一个过期的文件
	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	oldFileName := filepath.Join(tempDir, fmt.Sprintf("test_log_%s.log", threeDaysAgo.Format("2006-01-02")))
	oldFile, err := os.Create(oldFileName)
	if err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}
	oldFile.WriteString("This is an old log file")
	oldFile.Close()

	// 创建RotateWriter配置，设置MaxAge为0（不清理）
	config := RotateConfig{
		Filename:     filepath.Join(tempDir, "test_log.log"),
		MaxAge:       0,  // 不限制保留天数
		MaxSize:      10, // 10MB
		TimeRotation: "daily",
	}

	writer, err := NewRotateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer writer.Close()

	// 手动调用清理函数
	writer.cleanupExpiredFiles()

	// 检查文件是否依然存在（因为MaxAge为0时不清理）
	if _, err := os.Stat(oldFileName); os.IsNotExist(err) {
		t.Errorf("File %s should still exist when MaxAge is 0", oldFileName)
	}
}

// TestDifferentTimeFormats 测试不同时间格式的文件名
func TestDifferentTimeFormats(t *testing.T) {
	// 创建临时目录
	tempDir := "./test_logs_formats"
	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// 创建不同格式的过期文件
	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	oldDailyFile := filepath.Join(tempDir, fmt.Sprintf("app_%s.log", threeDaysAgo.Format("2006-01-02")))
	oldHourlyFile := filepath.Join(tempDir, fmt.Sprintf("app_%s.log", twoDaysAgo.Format("2006-01-02_15")))
	oldMinutelyFile := filepath.Join(tempDir, fmt.Sprintf("app_%s.log", twoDaysAgo.Format("2006-01-02_15_04")))

	for _, filename := range []string{oldDailyFile, oldHourlyFile, oldMinutelyFile} {
		file, err := os.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
		file.WriteString("test content")
		file.Close()
	}

	// 创建RotateWriter配置，设置MaxAge为1天
	config := RotateConfig{
		Filename:     filepath.Join(tempDir, "app.log"),
		MaxAge:       1,  // 只保留1天内的文件
		MaxSize:      10, // 10MB
		TimeRotation: "daily",
	}

	writer, err := NewRotateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer writer.Close()

	// 手动调用清理函数
	writer.cleanupExpiredFiles()

	// 检查所有过期文件是否都被删除
	for _, filename := range []string{oldDailyFile, oldHourlyFile, oldMinutelyFile} {
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			t.Errorf("File %s should have been deleted", filename)
		}
	}
}
