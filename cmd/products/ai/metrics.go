package ai

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	aiRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_description_requests_total",
			Help: "Total number of AI description generation requests",
		},
		[]string{"status"},
	)

	aiCacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ai_description_cache_hits_total",
			Help: "Total number of cache hits for AI descriptions",
		},
	)

	aiLatencySeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ai_description_latency_seconds",
			Help:    "Latency of AI description generation in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	aiErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_description_errors_total",
			Help: "Total number of AI description generation errors",
		},
		[]string{"error_type"},
	)
)

func RecordRequest(status string) {
	aiRequestsTotal.WithLabelValues(status).Inc()
}

func RecordCacheHit() {
	aiCacheHitsTotal.Inc()
}

func RecordLatency(seconds float64) {
	aiLatencySeconds.Observe(seconds)
}

func RecordError(errorType string) {
	aiErrorsTotal.WithLabelValues(errorType).Inc()
}
