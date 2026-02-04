package doris_streamload

// Client StreamLoad客户端接口
type Client interface {
	Load(data []byte) (*StreamLoadResponse, error)
}

// BatchCollector 批量收集器接口
type BatchCollector interface {
	Add(data interface{}) error
	Len() int
	Size() int
	Flush() error
	Close() error
}

// Option 配置选项接口
type Option interface{}

// CollectorOption 攒批收集器配置选项
type CollectorOption func(*Collect)
