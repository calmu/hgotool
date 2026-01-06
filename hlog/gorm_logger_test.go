package hlog

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestGormLoggerAdapter 测试GORM Logger适配器的基本功能
func TestGormLoggerAdapter(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 创建一个zap logger配置
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/gorm_test.log"},
		Encoder:    "json",
		EncoderConfig: &EncoderConfig{
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	hlogger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create hlog logger: %v", err)
	}
	defer hlogger.Close()

	// 创建GORM适配器
	gormLogger := NewGormLogger(hlogger, &logger.Config{
		SlowThreshold:             100 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})

	// 测试LogMode方法
	newLogger := gormLogger.LogMode(logger.Error)
	if newLogger == nil {
		t.Fatal("LogMode should return a logger")
	}

	// 测试Info方法
	gormLogger.Info(context.Background(), "Test info message: %s", "hello")

	// 测试Warn方法
	gormLogger.Warn(context.Background(), "Test warn message: %s", "warning")

	// 测试Error方法
	gormLogger.Error(context.Background(), "Test error message: %s", "error")

	// 测试Trace方法 - 模拟正常SQL执行
	gormLogger.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM users", 1
	}, nil)

	// 测试Trace方法 - 模拟慢SQL
	gormLogger.Trace(context.Background(), time.Now().Add(-300*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM large_table", 100
	}, nil)

	// 测试Trace方法 - 模拟SQL错误
	gormLogger.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM non_existent_table", 0
	}, fmt.Errorf("table does not exist"))

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/gorm_test.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("GORM log file was not created: %s", logFile)
	} else {
		t.Logf("GORM log file created successfully: %s", logFile)
	}
}

// TestGormWithSQLite 测试GORM与SQLite数据库集成
func TestGormWithSQLite(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 创建一个zap logger配置
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/gorm_sqlite_test.log", "stdout"},
		Encoder:    "console",
		EncoderConfig: &EncoderConfig{
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	hlogger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create hlog logger: %v", err)
	}
	defer hlogger.Close()

	// 创建GORM适配器
	gormLogger := NewGormLogger(hlogger, &logger.Config{
		SlowThreshold:             100 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})

	// 连接SQLite数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		t.Fatalf("Failed to connect to SQLite: %v", err)
	}

	// 创建一个测试表
	type User struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:255"`
		Age  int
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// 执行一些操作来测试日志记录
	user := User{Name: "Test User", Age: 30}
	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("Failed to create user: %v", result.Error)
	}

	var retrievedUser User
	result = db.First(&retrievedUser, "name = ?", "Test User")
	if result.Error != nil {
		t.Fatalf("Failed to retrieve user: %v", result.Error)
	}

	// 模拟慢查询
	_ = db.WithContext(context.Background()).Session(&gorm.Session{Logger: gormLogger.LogMode(logger.Info)})
	var users []User
	db.Raw("SELECT * FROM users WHERE name LIKE ?", "%Test%").Find(&users)

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/gorm_sqlite_test.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("SQLite GORM log file was not created: %s", logFile)
	} else {
		t.Logf("SQLite GORM log file created successfully: %s", logFile)
	}
}

// TestGormWithMySQL 测试GORM与MySQL数据库集成 (如果MySQL可用)
func TestGormWithMySQL(t *testing.T) {
	// 检查是否有MySQL环境变量配置
	mysqlDSN := os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		t.Skip("MYSQL_DSN not set, skipping MySQL test")
		return
	}

	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 创建一个zap logger配置
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/gorm_mysql_test.log", "stdout"},
		Encoder:    "console",
		EncoderConfig: &EncoderConfig{
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	hlogger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create hlog logger: %v", err)
	}
	defer hlogger.Close()

	// 创建GORM适配器
	gormLogger := NewGormLogger(hlogger, &logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})

	// 连接MySQL数据库
	db, err := gorm.Open(mysql.Open(mysqlDSN), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// 获取原生SQL DB以执行ping
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get SQL DB: %v", err)
	}
	defer sqlDB.Close()

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping MySQL: %v", err)
	}

	// 创建一个测试表
	type Product struct {
		ID    uint   `gorm:"primaryKey"`
		Name  string `gorm:"size:255"`
		Price uint
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// 执行一些操作来测试日志记录
	product := Product{Name: "Test Product", Price: 100}
	result := db.Create(&product)
	if result.Error != nil {
		t.Fatalf("Failed to create product: %v", result.Error)
	}

	var retrievedProduct Product
	result = db.First(&retrievedProduct, "name = ?", "Test Product")
	if result.Error != nil {
		t.Fatalf("Failed to retrieve product: %v", result.Error)
	}

	// 模拟慢查询
	var products []Product
	db.Raw("SELECT * FROM products WHERE name LIKE ?", "%Test%").Find(&products)

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/gorm_mysql_test.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("MySQL GORM log file was not created: %s", logFile)
	} else {
		t.Logf("MySQL GORM log file created successfully: %s", logFile)
	}
}

// TestGormWithPostgreSQL 测试GORM与PostgreSQL数据库集成 (如果PostgreSQL可用)
func TestGormWithPostgreSQL(t *testing.T) {
	// 检查是否有PostgreSQL环境变量配置
	pgDSN := os.Getenv("POSTGRES_DSN")
	if pgDSN == "" {
		t.Skip("POSTGRES_DSN not set, skipping PostgreSQL test")
		return
	}

	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 创建一个zap logger配置
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/gorm_postgres_test.log"},
		Encoder:    "json",
		EncoderConfig: &EncoderConfig{
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	hlogger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create hlog logger: %v", err)
	}
	defer hlogger.Close()

	// 创建GORM适配器
	gormLogger := NewGormLogger(hlogger, &logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})

	// 连接PostgreSQL数据库
	db, err := gorm.Open(postgres.Open(pgDSN), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// 获取原生SQL DB以执行ping
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get SQL DB: %v", err)
	}
	defer sqlDB.Close()

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	// 创建一个测试表
	type Order struct {
		ID        uint `gorm:"primaryKey"`
		UserID    uint `gorm:"index"`
		Total     uint
		Status    string `gorm:"size:50"`
		CreatedAt time.Time
	}

	if err := db.AutoMigrate(&Order{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// 执行一些操作来测试日志记录
	order := Order{UserID: 1, Total: 200, Status: "pending", CreatedAt: time.Now()}
	result := db.Create(&order)
	if result.Error != nil {
		t.Fatalf("Failed to create order: %v", result.Error)
	}

	var retrievedOrder Order
	result = db.First(&retrievedOrder, "user_id = ?", 1)
	if result.Error != nil {
		t.Fatalf("Failed to retrieve order: %v", result.Error)
	}

	// 模拟慢查询
	var orders []Order
	db.Raw("SELECT * FROM orders WHERE status = ?", "pending").Find(&orders)

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/gorm_postgres_test.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("PostgreSQL GORM log file was not created: %s", logFile)
	} else {
		t.Logf("PostgreSQL GORM log file created successfully: %s", logFile)
	}
}

// TestSlowQueryLogging 测试慢查询日志记录
func TestSlowQueryLogging(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 创建一个zap logger配置
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/slow_query_test.log"},
		Encoder:    "json",
		EncoderConfig: &EncoderConfig{
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	hlogger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create hlog logger: %v", err)
	}
	defer hlogger.Close()

	// 创建GORM适配器，设置较低的慢查询阈值以方便测试
	gormLogger := NewGormLogger(hlogger, &logger.Config{
		SlowThreshold:             50 * time.Millisecond, // 设置较低阈值
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})

	// 模拟慢查询（超过阈值）
	startTime := time.Now().Add(-100 * time.Millisecond) // 模拟执行时间超过阈值
	gormLogger.Trace(context.Background(), startTime, func() (string, int64) {
		return "SELECT SLEEP(1) FROM users", 0
	}, nil)

	// 模拟正常查询（低于阈值）
	startTime2 := time.Now().Add(-20 * time.Millisecond) // 模拟执行时间低于阈值
	gormLogger.Trace(context.Background(), startTime2, func() (string, int64) {
		return "SELECT id FROM users LIMIT 1", 1
	}, nil)

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/slow_query_test.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Slow query log file was not created: %s", logFile)
	} else {
		t.Logf("Slow query log file created successfully: %s", logFile)
	}
}

// TestGormErrorLogging 测试GORM错误日志记录
func TestGormErrorLogging(t *testing.T) {
	// 确保日志目录存在
	os.MkdirAll("./log", 0755)

	// 创建一个zap logger配置
	config := LoggerConfig{
		Level:      "info",
		OutputPath: []string{"./log/gorm_error_test.log"},
		Encoder:    "json",
		EncoderConfig: &EncoderConfig{
			TimeLayout: "2006-01-02 15:04:05",
		},
	}

	hlogger, err := NewZapLogger(config)
	if err != nil {
		t.Fatalf("Failed to create hlog logger: %v", err)
	}
	defer hlogger.Close()

	// 创建GORM适配器
	gormLogger := NewGormLogger(hlogger, &logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Error,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})

	// 模拟SQL错误
	gormLogger.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM non_existent_table", 0
	}, fmt.Errorf("table does not exist"))

	// 模拟记录未找到错误 (这应该被记录，因为我们设置了IgnoreRecordNotFoundError为false)
	gormLogger.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM users WHERE id = ?", 999999
	}, gorm.ErrRecordNotFound)

	// 使用zap字段进行额外的日志记录，确保导入的zap包被使用
	_ = zap.String("test", "value")

	// 等待确保日志写入文件
	time.Sleep(100 * time.Millisecond)

	// 验证日志文件是否已创建
	logFile := "./log/gorm_error_test.log"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("GORM error log file was not created: %s", logFile)
	} else {
		t.Logf("GORM error log file created successfully: %s", logFile)
	}
}
