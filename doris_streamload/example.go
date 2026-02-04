package doris_streamload

import (
	"fmt"
	"log"
	"time"

	"github.com/calmu/hgotool/hlog"
)

// ExampleUsage 示例：如何使用StreamLoad客户端和Collect攒批结构体
func ExampleUsage() {
	// 创建日志记录器
	logger, err := hlog.NewZapLogger(hlog.LoggerConfig{
		Level:      "info",
		OutputPath: []string{"stdout"},
		Encoder:    "console",
	})
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// 1. 创建StreamLoad客户端配置
	config := &Config{
		Host:       "localhost",      // Doris FE节点地址
		Port:       "8030",           // FE HTTP端口
		Username:   "root",           // 用户名
		Password:   "",               // 密码
		Database:   "example_db",     // 目标数据库
		Table:      "example_table",  // 目标表
		BufferSize: 5 * 1024 * 1024,  // 5MB缓冲区
		BufferRows: 500,              // 500行缓冲
		Timeout:    60 * time.Second, // 60秒超时
		EnableGzip: true,             // 启用gzip压缩
		Headers: map[string]string{ // 自定义HTTP头部参数
			"strict_mode":      "true",          // 严格模式
			"timezone":         "Asia/Shanghai", // 时区设置
			"max_filter_ratio": "0.1",           // 最大过滤比例
		},
		HLogger: logger, // 日志记录器
	}

	// 2. 创建StreamLoad客户端
	client, err := NewStreamLoadClient(config)
	if err != nil {
		log.Fatalf("Failed to create StreamLoad client: %v", err)
	}

	// 3. 创建Collect攒批结构体
	collect := NewCollect(client,
		WithMaxSize(1000),          // 最大1000条数据
		WithMaxBytes(10*1024*1024), // 最大10MB
		WithCallback(func(err error) { // 发送结果回调
			if err != nil {
				fmt.Printf("Stream load failed: %v\n", err)
			} else {
				fmt.Println("Stream load succeeded")
			}
		}),
		WithAutoFlushInterval(30*time.Second), // 每30秒自动刷新
	)

	// 4. 添加数据到攒批中
	for i := 0; i < 10; i++ {
		data := map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("user_%d", i),
			"value": i * 10,
		}

		err := collect.Add(data)
		if err != nil {
			log.Printf("Failed to add data: %v", err)
			continue
		}

		fmt.Printf("Added data item %d, current batch size: %d\n", i, collect.Len())
	}

	// 5. 显示当前攒批状态
	fmt.Printf("Final batch size: %d items, %d bytes\n", collect.Len(), collect.Size())

	// 6. 手动刷新攒批（可选，如果未达到阈值且想立即发送）
	err = collect.Flush()
	if err != nil {
		log.Printf("Failed to flush batch: %v", err)
	}

	// 7. 关闭Collect，确保所有数据都被发送
	err = collect.Close()
	if err != nil {
		log.Printf("Error closing collect: %v", err)
	}

	fmt.Println("All data sent successfully")
}

// ExampleDirectUsage 示例：直接使用StreamLoad客户端发送JSON数据
func ExampleDirectUsage() {
	config := &Config{
		Host:     "localhost",
		Username: "root",
		Password: "",
		Database: "example_db",
		Table:    "example_table",
	}

	client, err := NewStreamLoadClient(config)
	if err != nil {
		log.Fatalf("Failed to create StreamLoad client: %v", err)
	}

	// 准备JSON数据
	jsonData := []byte(`[
		{"id": 1, "name": "Alice", "age": 25},
		{"id": 2, "name": "Bob", "age": 30},
		{"id": 3, "name": "Charlie", "age": 35}
	]`)

	// 发送数据到Doris
	resp, err := client.Load(jsonData)
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	fmt.Printf("Stream load response: %+v\n", resp)
}
