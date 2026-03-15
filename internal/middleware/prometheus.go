package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP 请求指标
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{"method", "path"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// Provider 请求指标
	providerRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_provider_requests_total",
			Help: "Total number of provider requests",
		},
		[]string{"provider", "model", "status"},
	)

	providerRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_provider_request_duration_seconds",
			Help:    "Provider request duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
		[]string{"provider", "model"},
	)

	// Token 使用指标
	tokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_tokens_total",
			Help: "Total number of tokens used",
		},
		[]string{"provider", "model", "type"},
	)

	// 活跃连接数
	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gateway_active_connections",
			Help: "Number of active connections",
		},
	)

	// 流式请求指标
	streamRequestsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_stream_requests_active",
			Help: "Number of active streaming requests",
		},
		[]string{"provider", "model"},
	)

	// 错误指标
	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_errors_total",
			Help: "Total number of errors",
		},
		[]string{"provider", "type"},
	)

	// 限流指标
	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"type"},
	)

	// API Key 指标
	apiKeyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_api_key_requests_total",
			Help: "Total number of requests per API key",
		},
		[]string{"key_id", "status"},
	)
)

// Prometheus 返回 Prometheus 指标中间件
func Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过 metrics 端点本身
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		reqSize := computeRequestSize(c)

		activeConnections.Inc()
		defer activeConnections.Dec()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
		httpRequestSize.WithLabelValues(c.Request.Method, path).Observe(float64(reqSize))
		httpResponseSize.WithLabelValues(c.Request.Method, path).Observe(float64(c.Writer.Size()))
	}
}

func computeRequestSize(c *gin.Context) int {
	size := 0
	if c.Request.URL != nil {
		size += len(c.Request.URL.String())
	}
	size += len(c.Request.Method)
	size += len(c.Request.Proto)
	for name, values := range c.Request.Header {
		size += len(name)
		for _, v := range values {
			size += len(v)
		}
	}
	size += len(c.Request.Host)
	if c.Request.ContentLength > 0 {
		size += int(c.Request.ContentLength)
	}
	return size
}

// PrometheusHandler 返回 Prometheus HTTP handler
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordProviderRequest 记录 Provider 请求指标
func RecordProviderRequest(provider, model, status string, duration time.Duration) {
	providerRequestsTotal.WithLabelValues(provider, model, status).Inc()
	providerRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordTokens 记录 Token 使用
func RecordTokens(provider, model string, promptTokens, completionTokens int) {
	if promptTokens > 0 {
		tokensTotal.WithLabelValues(provider, model, "prompt").Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		tokensTotal.WithLabelValues(provider, model, "completion").Add(float64(completionTokens))
	}
}

// RecordError 记录错误
func RecordError(provider, errorType string) {
	errorsTotal.WithLabelValues(provider, errorType).Inc()
}

// RecordRateLimitHit 记录限流命中
func RecordRateLimitHit(limitType string) {
	rateLimitHits.WithLabelValues(limitType).Inc()
}

// RecordAPIKeyRequest 记录 API Key 请求
func RecordAPIKeyRequest(keyID, status string) {
	apiKeyRequestsTotal.WithLabelValues(keyID, status).Inc()
}

// StreamRequestStart 流式请求开始
func StreamRequestStart(provider, model string) {
	streamRequestsActive.WithLabelValues(provider, model).Inc()
}

// StreamRequestEnd 流式请求结束
func StreamRequestEnd(provider, model string) {
	streamRequestsActive.WithLabelValues(provider, model).Dec()
}

// Metrics 指标结构（用于自定义注册）
type Metrics struct {
	Registry *prometheus.Registry
}

// NewMetrics 创建自定义指标注册器
func NewMetrics() *Metrics {
	return &Metrics{
		Registry: prometheus.NewRegistry(),
	}
}

// Register 注册自定义指标
func (m *Metrics) Register(collectors ...prometheus.Collector) error {
	for _, c := range collectors {
		if err := m.Registry.Register(c); err != nil {
			return err
		}
	}
	return nil
}
