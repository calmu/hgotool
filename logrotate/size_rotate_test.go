package logrotate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSizeRotationAndCleanup 测试大小轮转和历史清理
func TestSizeRotationAndCleanup(t *testing.T) {
	// 创建临时目录
	tempDir := "./test_size_rotation"
	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "app.log")

	// 创建RotateWriter配置，设置MaxSize为1KB，MaxAge为1天
	config := RotateConfig{
		Filename:   logFile,
		MaxSize:    1, // 1MB (实际是1KB，这里为了测试方便)
		MaxAge:     1, // 保留1天
		MaxBackups: 5, // 最大备份文件数
	}

	writer, err := NewRotateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer writer.Close()

	// 写入超过1KB的数据以触发大小轮转
	largeData := make([]byte, 1500) // 1.5KB数据
	for i := range largeData {
		largeData[i] = byte('a' + (i % 26)) // 填充字母a-z循环
	}

	// 分多次写入，确保触发轮转
	for i := 0; i < 3; i++ {
		_, err := writer.Write(largeData)
		if err != nil {
			t.Fatalf("Failed to write data: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 确保写入完成
	}

	// 获取目录中的文件列表
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// 统计轮转后的文件数量
	var rotatedFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Base(logFile) != file.Name() {
			rotatedFiles = append(rotatedFiles, file.Name())
		}
	}

	t.Logf("Original file: %s", filepath.Base(logFile))
	t.Logf("Rotated files count: %d", len(rotatedFiles))
	for _, f := range rotatedFiles {
		t.Logf("  - %s", f)
	}

	// 至少应该有一个轮转后的文件
	if len(rotatedFiles) == 0 {
		t.Errorf("Expected at least one rotated file, got %d", len(rotatedFiles))
	}

	// 检查当前日志文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Current log file %s should exist", logFile)
	}

	// 模拟创建一些旧的轮转备份文件来测试清理功能
	oldBackupFile1 := filepath.Join(tempDir, "app.1.log")
	oldBackupFile2 := filepath.Join(tempDir, "app.2.log")

	// 创建一些模拟的旧备份文件并设置它们的修改时间为过去
	for _, filename := range []string{oldBackupFile1, oldBackupFile2} {
		f, err := os.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create old backup file: %v", err)
		}

		// 写入一些数据
		f.WriteString("old backup data for testing")
		f.Close()

		// 设置修改时间为两天前（超过MaxAge=1天）
		pastTime := time.Now().AddDate(0, 0, -2)
		os.Chtimes(filename, pastTime, pastTime)
	}

	// 手动调用清理函数
	writer.cleanupExpiredFiles()

	// 检查旧备份文件是否被清理
	if _, err := os.Stat(oldBackupFile1); !os.IsNotExist(err) {
		t.Errorf("Old backup file %s should have been cleaned up", oldBackupFile1)
	}
	if _, err := os.Stat(oldBackupFile2); !os.IsNotExist(err) {
		t.Errorf("Old backup file %s should have been cleaned up", oldBackupFile2)
	}

	// 检查当前日志文件是否仍然存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Current log file %s should still exist", logFile)
	}
}

// TestSizeRotationWithSimulatedBackups 测试大小轮转的备份文件清理
func TestSizeRotationWithSimulatedBackups(t *testing.T) {
	// 创建临时目录
	tempDir := "./test_backup_cleanup"
	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "app.log")

	// 创建RotateWriter配置
	config := RotateConfig{
		Filename:   logFile,
		MaxSize:    1, // 1MB
		MaxAge:     1, // 保留1天
		MaxBackups: 3, // 最大备份文件数
	}

	writer, err := NewRotateWriter(config)
	if err != nil {
		t.Fatalf("Failed to create RotateWriter: %v", err)
	}
	defer writer.Close()

	// 模拟创建一些备份文件
	backupFiles := []string{
		filepath.Join(tempDir, "app.1.log"),
		filepath.Join(tempDir, "app.2.log"),
		filepath.Join(tempDir, "app.3.log"),
	}

	// 创建这些备份文件
	for i, backupFile := range backupFiles {
		f, err := os.Create(backupFile)
		if err != nil {
			t.Fatalf("Failed to create backup file %s: %v", backupFile, err)
		}
		f.WriteString(fmt.Sprintf("backup content %d", i))
		f.Close()

		// 设置一些文件的修改时间为过期时间
		if i%2 == 0 { // 设置部分文件为过期
			pastTime := time.Now().AddDate(0, 0, -2) // 2天前，超过MaxAge=1天
			os.Chtimes(backupFile, pastTime, pastTime)
		}
	}

	// 手动调用清理函数
	writer.cleanupExpiredFiles()

	// 检查哪些过期的备份文件被清理了
	for i, backupFile := range backupFiles {
		_, err := os.Stat(backupFile)
		exists := !os.IsNotExist(err)
		if i%2 == 0 { // 这些应该是过期的
			if exists {
				t.Errorf("Expired backup file %s should have been cleaned up", backupFile)
			} else {
				t.Logf("Expired backup file %s was correctly cleaned up", backupFile)
			}
		} else { // 这些应该不过期
			if !exists {
				t.Errorf("Non-expired backup file %s should still exist", backupFile)
			} else {
				t.Logf("Non-expired backup file %s still exists", backupFile)
			}
		}
	}

	// 检查当前日志文件是否仍然存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Current log file %s should still exist", logFile)
	}
}
