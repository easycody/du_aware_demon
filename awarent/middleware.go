package awarent

import (
	"fmt"
	"net/http"
	"time"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type (
	//Option func with options param
	Option  func(*options)
	options struct {
		resourceExtract func(*gin.Context) string
		blockFallback   func(*gin.Context)
	}
)

func evaluateOptions(opts []Option) *options {
	optCopy := &options{}
	for _, opt := range opts {
		opt(optCopy)
	}

	return optCopy
}

// WithResourceExtractor sets the resource extractor of the web requests.
func WithResourceExtractor(fn func(*gin.Context) string) Option {
	return func(opts *options) {
		opts.resourceExtract = fn
	}
}

// WithBlockFallback sets the fallback handler when requests are blocked.
func WithBlockFallback(fn func(ctx *gin.Context)) Option {
	return func(opts *options) {
		opts.blockFallback = fn
	}
}

// SentinelMiddleware returns new gin.HandlerFunc
// Default resource name is {method}:{path}, such as "GET:/api/users/:id"
// Default block fallback is returning 429 code
// Define your own behavior by setting options
func SentinelMiddleware(endPoint string, opts ...Option) gin.HandlerFunc {
	options := evaluateOptions(opts)
	path := endPoint
	return func(c *gin.Context) {

		if c.Request.URL.Path != path {
			c.Next()
			return
		}

		start := time.Now()
		resourceName := c.Request.Method + ":" + c.FullPath()
		if options.resourceExtract != nil {
			resourceName = options.resourceExtract(c)
		}

		entry, err := sentinel.Entry(
			resourceName,
			sentinel.WithResourceType(base.ResTypeWeb),
			sentinel.WithTrafficType(base.Inbound),
		)

		if err != nil {
			if options.blockFallback != nil {
				options.blockFallback(c)
			} else {
				c.AbortWithStatus(http.StatusTooManyRequests)
			}
			status := fmt.Sprintf("%d", c.Writer.Status())
			endpoint := c.Request.URL.Path
			lvs := []string{status, endpoint, resourceName}
			blockCount.WithLabelValues(lvs...).Inc()
			reqCount.WithLabelValues(lvs...).Inc()
			reqDuration.WithLabelValues(lvs...).Observe(time.Since(start).Seconds())
			return
		}
		defer entry.Exit()
		c.Next()
		status := fmt.Sprintf("%d", c.Writer.Status())
		endpoint := c.Request.URL.Path
		lvs := []string{status, endpoint, resourceName}
		passCount.WithLabelValues(lvs...).Inc()
		reqCount.WithLabelValues(lvs...).Inc()
		reqDuration.WithLabelValues(lvs...).Observe(time.Since(start).Seconds())

	}
}

// -------------------------------------------------------------------------
// Prometheus

const namespace = "service"

var (
	labels = []string{"status", "endpoint", "resource"}

	uptime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "uptime",
			Help:      "HTTP uptime.",
		}, nil,
	)

	reqCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_req_total",
		Help:      "Total number of HTTP requests made.",
	}, labels)
	passCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_passed_total",
		Help:      "Total number of HTTP requests passed.",
	}, labels)

	blockCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_block_total",
		Help:      "Total number of HTTP requests blocked.",
	}, labels)
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latencies in seconds.",
			Buckets:   []float64{.01, .05, 0.1, 0.25, 0.5, 1, 5},
		}, labels,
	)
)

var promHandler http.Handler

// init registers the prometheus metrics
func init() {
	promRegistry := prometheus.NewRegistry()
	promRegistry.MustRegister(uptime, reqCount, passCount, blockCount, reqDuration)
	go recordUptime()
	promHandler = promhttp.InstrumentMetricHandler(promRegistry, promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
}

// recordUptime increases service uptime per second.
func recordUptime() {
	for range time.Tick(time.Second) {
		uptime.WithLabelValues().Inc()
	}
}

// PromHandler wrappers the standard http.Handler to gin.HandlerFunc
func PromHandler(c *gin.Context) {
	promHandler.ServeHTTP(c.Writer, c.Request)
}
