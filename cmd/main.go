package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"simple-service/internal/handler"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Получение адреса Redis из переменных окружения
	redisAddr := os.Getenv("REDIS_HOST")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Инициализация Gin
	router := gin.Default()

	// Инициализация обработчика (без pod)
	h := handler.NewHandler(redisAddr)

	// Добавляем middleware для логирования
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Логируем только медленные запросы
		if duration > 100*time.Millisecond {
			log.Printf("Slow request: %s %s - %d - %v",
				c.Request.Method,
				c.Request.URL.Path,
				c.Writer.Status(),
				duration)
		}
	})

	// Подключаем middleware для сбора метрик Prometheus
	router.Use(h.MetricsMiddleware())

	// Основные маршруты
	router.POST("/metrics", h.HandleMetric)
	router.GET("/analytics", h.GetAnalytics)
	router.GET("/health", h.HealthCheck)

	// Метрики Prometheus
	router.GET("/metrics/prometheus", gin.WrapH(promhttp.Handler()))

	// Запуск основного сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
