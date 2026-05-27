package pkg

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	ragQueryDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rag_query_duration_seconds",
			Help:    "RAG query duration in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
	)

	ragCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rag_cache_hits_total",
			Help: "Total number of RAG cache hits",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, ragQueryDuration, ragCacheHits)
}

// MetricsMiddleware records HTTP request metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// RecordRAGQueryDuration records RAG query duration
func RecordRAGQueryDuration(seconds float64) {
	ragQueryDuration.Observe(seconds)
}

// RecordRAGCacheHit increments RAG cache hit counter
func RecordRAGCacheHit() {
	ragCacheHits.Inc()
}

// PrometheusHandler returns the Prometheus metrics HTTP handler
func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
