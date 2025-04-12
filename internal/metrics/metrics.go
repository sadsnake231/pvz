package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	HTTPRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path"},
	)

	HTTPResponseStatusCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_response_status_total",
			Help: "Total number of HTTP responses by status code",
		},
		[]string{"status", "path"},
	)

	CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"type"},
	)

	CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"type"},
	)

	CacheOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "status"},
	)

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration distribution",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"operation"},
	)

	OrdersProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_processed_total",
			Help: "Total number of processed orders",
		},
		[]string{"status"},
	)

	APIResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_response_time_seconds",
			Help:    "API response time distribution",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"endpoint", "method", "status"},
	)

	KafkaMessages = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_messages_total",
		Help: "Total messages consumed",
	})
	KafkaErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kafka_consumer_errors_total",
		Help: "Total processing errors",
	})

	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "status"},
	)

	RequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_requests_in_flight",
			Help: "Current number of requests being processed",
		},
		[]string{"method"},
	)

	OrderValueDistribution = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "order_value",
			Help:    "Order value distribution",
			Buckets: []float64{100, 500, 1000, 3000, 5000, 10000},
		},
	)

	OrderWeightDistribution = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "order_weight_kg",
			Help:    "Order weight distribution",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
	)

	OrderReturns = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "order_returns_total",
			Help: "Number of returned orders",
		},
	)

	OrdersByStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "orders_by_status",
			Help: "Current number of orders by status",
		},
		[]string{"status"},
	)

	FailedOrderCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "failed_order_count",
			Help: "Total number of not accepted orders",
		},
	)
)

func RegisterMetrics() error {
	collectors := []prometheus.Collector{
		HTTPRequestCount,
		HTTPResponseStatusCount,
		CacheHits,
		CacheMisses,
		CacheOperations,
		DBQueryDuration,
		OrdersProcessed,
		APIResponseTime,
		KafkaMessages,
		KafkaErrors,
		RequestCount,
		RequestsInFlight,
		OrderValueDistribution,
		OrderWeightDistribution,
		OrderReturns,
		OrdersByStatus,
		FailedOrderCount,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			return err
		}
	}

	return nil
}
