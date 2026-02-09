package doris_streamload

import (
	"errors"
	"testing"
	"time"
)

// MockStreamLoadClient 用于测试的模拟客户端
type MockStreamLoadClient struct {
	loadFunc func(data []byte) (*StreamLoadResponse, error)
}

func (m *MockStreamLoadClient) Load(data []byte) (*StreamLoadResponse, error) {
	if m.loadFunc != nil {
		return m.loadFunc(data)
	}
	// 默认成功响应
	return &StreamLoadResponse{
		Status:           "Success",
		Label:            "test_label",
		LoadBytes:        len(data),
		NumberLoadedRows: 1,
		LoadTimeMs:       100,
	}, nil
}

// Ensure MockStreamLoadClient implements Client interface
var _ Client = (*MockStreamLoadClient)(nil)

func TestNewCollect(t *testing.T) {
	mockClient := &MockStreamLoadClient{}

	collect := NewCollect(mockClient)

	if collect == nil {
		t.Fatal("Expected collect to be created, got nil")
	}

	if collect.client != mockClient {
		t.Error("Expected collect to have the provided client")
	}

	if collect.maxSize != 1000 {
		t.Errorf("Expected default maxSize to be 1000, got %d", collect.maxSize)
	}

	if collect.maxBytes != 10*1024*1024 {
		t.Errorf("Expected default maxBytes to be 10MB, got %d", collect.maxBytes)
	}

	if err := collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}

func TestCollectWithOptions(t *testing.T) {
	mockClient := &MockStreamLoadClient{}

	callbackCalled := false
	callbackErr := error(nil)

	collect := NewCollect(mockClient,
		WithMaxSize(500),
		WithMaxBytes(5*1024*1024), // 5MB
		WithCallback(func(err error) {
			callbackCalled = true
			callbackErr = err
		}),
	)

	if collect.maxSize != 500 {
		t.Errorf("Expected maxSize to be 500, got %d", collect.maxSize)
	}

	if collect.maxBytes != 5*1024*1024 {
		t.Errorf("Expected maxBytes to be 5MB, got %d", collect.maxBytes)
	}

	// 测试回调是否被调用
	testData := map[string]interface{}{"id": 1, "name": "test"}
	err := collect.Add(testData)
	if err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	// 手动调用Flush触发回调
	err = collect.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	if callbackErr != nil {
		t.Errorf("Expected callback error to be nil, got %v", callbackErr)
	}

	if err = collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}

func TestCollectAddAndLen(t *testing.T) {
	mockClient := &MockStreamLoadClient{}
	collect := NewCollect(mockClient)

	initialLen := collect.Len()
	if initialLen != 0 {
		t.Errorf("Expected initial length to be 0, got %d", initialLen)
	}

	testData := []map[string]interface{}{
		{"id": 1, "name": "test1"},
		{"id": 2, "name": "test2"},
		{"id": 3, "name": "test3"},
	}

	for i, data := range testData {
		err := collect.Add(data)
		if err != nil {
			t.Fatalf("Failed to add data at index %d: %v", i, err)
		}

		expectedLen := i + 1
		actualLen := collect.Len()
		if actualLen != expectedLen {
			t.Errorf("Expected length to be %d after adding item %d, got %d", expectedLen, i, actualLen)
		}
	}

	if err := collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}

func TestCollectSize(t *testing.T) {
	mockClient := &MockStreamLoadClient{}
	collect := NewCollect(mockClient)

	initialSize := collect.Size()
	if initialSize != 0 {
		t.Errorf("Expected initial size to be 0, got %d", initialSize)
	}

	testData := map[string]interface{}{"id": 1, "name": "test", "value": 123}
	err := collect.Add(testData)
	if err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	sizeAfterAdd := collect.Size()
	if sizeAfterAdd <= 0 {
		t.Errorf("Expected size to be greater than 0 after adding data, got %d", sizeAfterAdd)
	}

	if err = collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}

func TestCollectFlush(t *testing.T) {
	mockClient := &MockStreamLoadClient{
		loadFunc: func(data []byte) (*StreamLoadResponse, error) {
			return &StreamLoadResponse{
				Status: "Success",
				Label:  "test_label",
			}, nil
		},
	}

	collect := NewCollect(mockClient)

	testData := map[string]interface{}{"id": 1, "name": "test"}
	err := collect.Add(testData)
	if err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	if collect.Len() != 1 {
		t.Errorf("Expected length to be 1 before flush, got %d", collect.Len())
	}

	err = collect.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	if collect.Len() != 0 {
		t.Errorf("Expected length to be 0 after flush, got %d", collect.Len())
	}

	if collect.Size() != 0 {
		t.Errorf("Expected size to be 0 after flush, got %d", collect.Size())
	}

	if err = collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}

func TestCollectClose(t *testing.T) {
	mockClient := &MockStreamLoadClient{
		loadFunc: func(data []byte) (*StreamLoadResponse, error) {
			return &StreamLoadResponse{
				Status: "Success",
				Label:  "test_label",
			}, nil
		},
	}

	collect := NewCollect(mockClient)

	testData := map[string]interface{}{"id": 1, "name": "test"}
	err := collect.Add(testData)
	if err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	if collect.Len() != 1 {
		t.Errorf("Expected length to be 1 before close, got %d", collect.Len())
	}

	err = collect.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	if collect.Len() != 0 {
		t.Errorf("Expected length to be 0 after close, got %d", collect.Len())
	}
}

func TestCollectTicker(t *testing.T) {
	mockClient := &MockStreamLoadClient{
		loadFunc: func(data []byte) (*StreamLoadResponse, error) {
			return &StreamLoadResponse{
				Status: "Success",
				Label:  "test_label",
			}, nil
		},
	}

	collect := NewCollect(mockClient, WithAutoFlushInterval(time.Second*5))

	testData := map[string]interface{}{"id": 1, "name": "test"}
	err := collect.Add(testData)
	if err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	if collect.Len() != 1 {
		t.Errorf("Expected length to be 1 before close, got %d", collect.Len())
	}

	time.Sleep(time.Second * 6)

	err = collect.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	if collect.Len() != 0 {
		t.Errorf("Expected length to be 0 after close, got %d", collect.Len())
	}
}

func TestCollectAddExceedsSizeLimit(t *testing.T) {
	mockClient := &MockStreamLoadClient{}
	collect := NewCollect(mockClient, WithMaxSize(2)) // 最大2条

	testData := []map[string]interface{}{
		{"id": 1, "name": "test1"},
		{"id": 2, "name": "test2"},
		{"id": 3, "name": "test3"}, // 这个会超过限制
	}

	// 前两个应该能添加成功
	for i := 0; i < 2; i++ {
		err := collect.Add(testData[i])
		if err != nil {
			t.Fatalf("Failed to add data at index %d: %v", i, err)
		}
	}

	// 第三个应该失败
	err := collect.Add(testData[2])
	if err == nil {
		t.Error("Expected error when adding data that exceeds size limit, got nil")
	}

	if err = collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}

func TestCollectWithFailureCallback(t *testing.T) {
	expectedError := errors.New("mock load error")
	mockClient := &MockStreamLoadClient{
		loadFunc: func(data []byte) (*StreamLoadResponse, error) {
			return nil, expectedError
		},
	}

	callbackErr := error(nil)
	collect := NewCollect(mockClient,
		WithCallback(func(err error) {
			callbackErr = err
		}),
	)

	testData := map[string]interface{}{"id": 1, "name": "test"}
	err := collect.Add(testData)
	if err != nil {
		t.Fatalf("Failed to add data: %v", err)
	}

	err = collect.Flush()
	if err == nil {
		t.Error("Expected flush to return error")
	}

	if callbackErr == nil {
		t.Error("Expected callback to be called with error")
	}

	if callbackErr.Error() != expectedError.Error() {
		t.Errorf("Expected callback error to be '%v', got '%v'", expectedError, callbackErr)
	}

	if err = collect.Close(); err != nil {
		t.Errorf("Expected close error to be nil, got %v", err)
	}
}
