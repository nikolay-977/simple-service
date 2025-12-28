package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	Timestamp time.Time `json:"timestamp"`
	CPU       float64   `json:"cpu"`
	RPS       float64   `json:"rps"`
}

type Analytics struct {
	mu             sync.RWMutex
	windowSize     int
	metricsWindow  []Metric
	rollingAverage float64
	totalMetrics   int64
}

type Handler struct {
	redisClient *redis.Client
	analytics   *Analytics
	ctx         context.Context

	// Объявляем метрики как поля структуры, а не глобально
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
}

func NewHandler(redisAddr string) *Handler {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	analytics := &Analytics{
		windowSize:    50,
		metricsWindow: make([]Metric, 0, 50),
	}

	// Создаем метрики внутри конструктора
	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "simple_service_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "simple_service_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status_code"},
	)

	// Регистрируем метрики
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)

	return &Handler{
		redisClient:         rdb,
		analytics:           analytics,
		ctx:                 context.Background(),
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
	}
}

func (h *Handler) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Продолжаем обработку
		c.Next()

		// Записываем метрики после обработки
		duration := time.Since(start).Seconds()
		statusCode := fmt.Sprintf("%d", c.Writer.Status())

		// Обновляем метрики
		h.httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			statusCode,
		).Inc()

		h.httpRequestDuration.WithLabelValues(
			c.Request.Method,
			path,
			statusCode,
		).Observe(duration)
	}
}

func (h *Handler) HandleMetric(c *gin.Context) {
	var metric Metric
	if err := c.ShouldBindJSON(&metric); err != nil {
		// Увеличиваем счетчик ошибок
		h.httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			"400",
		).Inc()

		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	log.Printf("Received metric: RPS=%.2f, CPU=%.2f", metric.RPS, metric.CPU)

	// Обработка в Redis
	metricJSON, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Error marshaling metric: %v", err)
	} else {
		err = h.redisClient.Set(h.ctx,
			fmt.Sprintf("metric:%d", metric.Timestamp.UnixNano()),
			metricJSON,
			10*time.Minute).Err()
		if err != nil {
			log.Printf("Error saving to Redis: %v", err)
		}
	}

	// Обновляем аналитику
	average := h.analytics.AddMetric(metric)

	c.JSON(200, gin.H{
		"status":    "processed",
		"rps":       metric.RPS,
		"cpu":       metric.CPU,
		"avg_rps":   average,
		"timestamp": metric.Timestamp,
	})
}

func (h *Handler) GetAnalytics(c *gin.Context) {
	h.analytics.mu.RLock()
	defer h.analytics.mu.RUnlock()

	c.JSON(200, gin.H{
		"window_size":     len(h.analytics.metricsWindow),
		"rolling_average": h.analytics.rollingAverage,
		"total_metrics":   h.analytics.totalMetrics,
	})
}

func (h *Handler) HealthCheck(c *gin.Context) {
	if err := h.redisClient.Ping(h.ctx).Err(); err != nil {
		c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "healthy"})
}

func (a *Analytics) AddMetric(metric Metric) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.totalMetrics++

	// Добавление метрики в окно
	a.metricsWindow = append(a.metricsWindow, metric)
	if len(a.metricsWindow) > a.windowSize {
		a.metricsWindow = a.metricsWindow[1:]
	}

	// Расчет среднего значения
	a.calculateAverage()

	return a.rollingAverage
}

func (a *Analytics) calculateAverage() {
	n := len(a.metricsWindow)
	if n == 0 {
		a.rollingAverage = 0
		return
	}

	sum := 0.0
	for _, m := range a.metricsWindow {
		sum += m.RPS
	}
	a.rollingAverage = sum / float64(n)
}
