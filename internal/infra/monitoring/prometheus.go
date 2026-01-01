package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "storage_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	FileUploadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_file_uploads_total",
			Help: "Total number of file uploads",
		},
		[]string{"type", "status"},
	)

	FileUploadSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "storage_file_upload_size_bytes",
			Help:    "Size of uploaded files in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760},
		},
		[]string{"type"},
	)
)
