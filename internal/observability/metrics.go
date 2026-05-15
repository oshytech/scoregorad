package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Métricas HTTP
var (
	// HTTPRequestsTotal cuenta requests por método, ruta y código de estado.
	// Nomenclatura: <namespace>_<subsystem>_<name>_<unit>
	// El sufijo _total es convención Prometheus para contadores.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scoregorad",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration mide la latencia de cada request en segundos.
	// Usamos un Histogram (no Summary) porque permite agregar percentiles
	// entre múltiples instancias del servicio.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "scoregorad",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// Métricas de caché
var (
	CacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "scoregorad",
		Subsystem: "cache",
		Name:      "hits_total",
		Help:      "Total number of leaderboard cache hits.",
	})

	CacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "scoregorad",
		Subsystem: "cache",
		Name:      "misses_total",
		Help:      "Total number of leaderboard cache misses.",
	})
)

// Métricas del worker pool
var (
	WorkerQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "scoregorad",
		Subsystem: "worker",
		Name:      "queue_depth",
		Help:      "Current number of events waiting in the worker pool queue.",
	})

	WorkerEventsDropped = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "scoregorad",
		Subsystem: "worker",
		Name:      "events_dropped_total",
		Help:      "Total number of score events dropped due to full pool.",
	})
)
