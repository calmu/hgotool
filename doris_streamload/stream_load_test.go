package doris_streamload

import (
	"testing"
)

func TestNewStreamLoadClient(t *testing.T) {
	config := &Config{
		Host:     "localhost",
		Username: "root",
		Password: "",
		Database: "test_db",
		Table:    "test_table",
	}

	client, err := NewStreamLoadClient(config)
	if err != nil {
		t.Fatalf("Failed to create StreamLoad client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if client.config.Host != "localhost" {
		t.Errorf("Expected host to be 'localhost', got '%s'", client.config.Host)
	}
}

func TestNewStreamLoadClient_InvalidConfig(t *testing.T) {
	// 测试缺少必要字段的情况
	invalidConfigs := []*Config{
		{Host: "", Username: "user", Database: "db", Table: "table"},        // 缺少Host
		{Host: "localhost", Username: "", Database: "db", Table: "table"},   // 缺少Username
		{Host: "localhost", Username: "user", Database: "", Table: "table"}, // 缺少Database
		{Host: "localhost", Username: "user", Database: "db", Table: ""},    // 缺少Table
	}

	for i, config := range invalidConfigs {
		_, err := NewStreamLoadClient(config)
		if err == nil {
			t.Errorf("Test case %d: Expected error for invalid config, got nil", i)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	config := &Config{
		Host:     "localhost",
		Username: "root",
		Password: "",
		Database: "test_db",
		Table:    "test_table",
		// 不设置Port, BufferSize, BufferRows, Timeout，使用默认值
	}

	client, err := NewStreamLoadClient(config)
	if err != nil {
		t.Fatalf("Failed to create StreamLoad client: %v", err)
	}

	if client.config.Port != DefaultConfig.Port {
		t.Errorf("Expected port to be default '%s', got '%s'", DefaultConfig.Port, client.config.Port)
	}

	if client.config.BufferSize != DefaultConfig.BufferSize {
		t.Errorf("Expected bufferSize to be default %d, got %d", DefaultConfig.BufferSize, client.config.BufferSize)
	}

	if client.config.BufferRows != DefaultConfig.BufferRows {
		t.Errorf("Expected bufferRows to be default %d, got %d", DefaultConfig.BufferRows, client.config.BufferRows)
	}

	if client.config.Timeout != DefaultConfig.Timeout {
		t.Errorf("Expected timeout to be default %v, got %v", DefaultConfig.Timeout, client.config.Timeout)
	}
}
