package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"awesomeproject/internal/platform/apperr"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
)

const RequestIDKey = "request_id"

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_http_requests_total",
		Help: "Total number of HTTP requests processed by the service.",
	}, []string{"service", "method", "route", "status"})
	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chat_http_request_duration_seconds",
		Help:    "Duration of HTTP requests processed by the service.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "route", "status"})
)

func HTTPRequestsTotalMetricName() string {
	return "chat_http_requests_total"
}

func HTTPRequestDurationMetricName() string {
	return "chat_http_request_duration_seconds"
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, contracts.Envelope{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, contracts.Envelope{
		Code:    status,
		Message: message,
	})
}

func FailError(c *gin.Context, err error) {
	appErr := apperr.From(err)
	c.JSON(appErr.Status, contracts.Envelope{
		Code:      appErr.Status,
		Message:   appErr.Message,
		ErrorCode: appErr.Code,
	})
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
		if spanContext := trace.SpanContextFromContext(c.Request.Context()); spanContext.IsValid() {
			c.Header("X-Trace-Id", spanContext.TraceID().String())
		}
	}
}

func StructuredLogger(serviceName string) gin.HandlerFunc {
	logger := slog.Default()
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		requestID, _ := c.Get(RequestIDKey)
		spanContext := trace.SpanContextFromContext(c.Request.Context())
		logger.Info("http_request",
			"service", serviceName,
			"request_id", requestID,
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
			"method", c.Request.Method,
			"route", route,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func Metrics(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(serviceName, c.Request.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(serviceName, c.Request.Method, route, status).Observe(time.Since(startedAt).Seconds())
	}
}

func MetricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

func CORS(origins []string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Refresh-Token,X-Device-Id")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func newRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buf)
}
