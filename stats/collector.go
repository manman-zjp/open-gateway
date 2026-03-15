package stats

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"gateway/model"
	"gateway/pkg/logger"
	"gateway/storage"

	"go.uber.org/zap"
)

// Collector 统计收集器
type Collector struct {
	buffer        chan *model.RequestRecord
	storage       storage.Storage
	batchSize     int
	flushInterval time.Duration
	wg            sync.WaitGroup
	stopCh        chan struct{}
}

// Config 收集器配置
type Config struct {
	BufferSize    int           // 缓冲区大小
	BatchSize     int           // 批量写入大小
	FlushInterval time.Duration // 刷新间隔
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		BufferSize:    10000,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}
}

// NewCollector 创建统计收集器
func NewCollector(cfg *Config, storage storage.Storage) *Collector {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	c := &Collector{
		buffer:        make(chan *model.RequestRecord, cfg.BufferSize),
		storage:       storage,
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		stopCh:        make(chan struct{}),
	}

	// 启动后台写入协程
	c.wg.Add(1)
	go c.worker()

	return c
}

// Collect 收集请求记录（异步）
func (c *Collector) Collect(record *model.RequestRecord) {
	select {
	case c.buffer <- record:
		// 成功入队
	default:
		// 缓冲区满，丢弃记录并记录警告
		logger.Warn("stats buffer full, record dropped",
			zap.String("request_id", record.RequestID),
		)
	}
}

// worker 后台工作协程
func (c *Collector) worker() {
	defer c.wg.Done()

	batch := make([]*model.RequestRecord, 0, c.batchSize)
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case record := <-c.buffer:
			batch = append(batch, record)
			if len(batch) >= c.batchSize {
				c.flush(batch)
				batch = make([]*model.RequestRecord, 0, c.batchSize)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				c.flush(batch)
				batch = make([]*model.RequestRecord, 0, c.batchSize)
			}

		case <-c.stopCh:
			// 处理剩余的缓冲数据
			close(c.buffer)
			for record := range c.buffer {
				batch = append(batch, record)
			}
			if len(batch) > 0 {
				c.flush(batch)
			}
			return
		}
	}
}

// flush 批量写入存储
func (c *Collector) flush(batch []*model.RequestRecord) {
	if c.storage == nil || len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, record := range batch {
		if err := c.storage.SaveRequest(ctx, record); err != nil {
			logger.Error("failed to save request record",
				zap.String("request_id", record.RequestID),
				zap.Error(err),
			)
		}
	}

	logger.Debug("flushed stats batch", zap.Int("count", len(batch)))
}

// Stop 停止收集器
func (c *Collector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// MemoryStats 内存中的统计数据（用于没有存储后端时）
type MemoryStats struct {
	mu                sync.RWMutex
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalTokens       int64
	PromptTokens      int64
	CompletionTokens  int64
	TotalLatencyMs    int64
	ByProvider        map[string]*ProviderStats
	ByModel           map[string]*ModelStats
	RecentRequests    []*model.RequestRecord
	MaxRecentRequests int
}

// ProviderStats 提供商统计
type ProviderStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalTokens     int64
	TotalLatencyMs  int64
}

// ModelStats 模型统计
type ModelStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalTokens     int64
	TotalLatencyMs  int64
}

// NewMemoryStats 创建内存统计
func NewMemoryStats(maxRecent int) *MemoryStats {
	return &MemoryStats{
		ByProvider:        make(map[string]*ProviderStats),
		ByModel:           make(map[string]*ModelStats),
		RecentRequests:    make([]*model.RequestRecord, 0),
		MaxRecentRequests: maxRecent,
	}
}

// Add 添加记录
func (ms *MemoryStats) Add(record *model.RequestRecord) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.TotalRequests++
	if record.Status >= 200 && record.Status < 300 {
		ms.SuccessRequests++
	} else {
		ms.FailedRequests++
	}

	ms.TotalTokens += int64(record.TotalTokens)
	ms.PromptTokens += int64(record.PromptTokens)
	ms.CompletionTokens += int64(record.CompletionTokens)
	ms.TotalLatencyMs += record.Latency.Milliseconds()

	// 按提供商统计
	if _, ok := ms.ByProvider[record.Provider]; !ok {
		ms.ByProvider[record.Provider] = &ProviderStats{}
	}
	ps := ms.ByProvider[record.Provider]
	ps.TotalRequests++
	if record.Status >= 200 && record.Status < 300 {
		ps.SuccessRequests++
	} else {
		ps.FailedRequests++
	}
	ps.TotalTokens += int64(record.TotalTokens)
	ps.TotalLatencyMs += record.Latency.Milliseconds()

	// 按模型统计
	if _, ok := ms.ByModel[record.Model]; !ok {
		ms.ByModel[record.Model] = &ModelStats{}
	}
	mst := ms.ByModel[record.Model]
	mst.TotalRequests++
	if record.Status >= 200 && record.Status < 300 {
		mst.SuccessRequests++
	} else {
		mst.FailedRequests++
	}
	mst.TotalTokens += int64(record.TotalTokens)
	mst.TotalLatencyMs += record.Latency.Milliseconds()

	// 保存最近请求
	ms.RecentRequests = append(ms.RecentRequests, record)
	if len(ms.RecentRequests) > ms.MaxRecentRequests {
		ms.RecentRequests = ms.RecentRequests[1:]
	}
}

// GetSummary 获取统计摘要
func (ms *MemoryStats) GetSummary() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	avgLatency := int64(0)
	if ms.TotalRequests > 0 {
		avgLatency = ms.TotalLatencyMs / ms.TotalRequests
	}

	return map[string]interface{}{
		"total_requests":    ms.TotalRequests,
		"success_requests":  ms.SuccessRequests,
		"failed_requests":   ms.FailedRequests,
		"total_tokens":      ms.TotalTokens,
		"prompt_tokens":     ms.PromptTokens,
		"completion_tokens": ms.CompletionTokens,
		"avg_latency_ms":    avgLatency,
		"by_provider":       ms.ByProvider,
		"by_model":          ms.ByModel,
	}
}

// ToJSON 转为JSON
func (ms *MemoryStats) ToJSON() ([]byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return json.Marshal(ms.GetSummary())
}

// 全局统计实例
var globalStats *MemoryStats
var globalCollector *Collector
var statsOnce sync.Once

// Init 初始化统计模块
func Init(cfg *Config, storage storage.Storage) {
	statsOnce.Do(func() {
		globalStats = NewMemoryStats(1000)
		if storage != nil {
			globalCollector = NewCollector(cfg, storage)
		}
	})
}

// Record 记录请求（全局方法）
func Record(record *model.RequestRecord) {
	if globalStats != nil {
		globalStats.Add(record)
	}
	if globalCollector != nil {
		globalCollector.Collect(record)
	}
}

// GetStats 获取统计数据
func GetStats() map[string]interface{} {
	if globalStats != nil {
		return globalStats.GetSummary()
	}
	return nil
}

// Shutdown 关闭统计模块
func Shutdown() {
	if globalCollector != nil {
		globalCollector.Stop()
	}
}
