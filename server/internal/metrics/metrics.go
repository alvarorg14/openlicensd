package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "openlicensd"

// Metrics holds Prometheus collectors for OpenLicensd.
type Metrics struct {
	registry *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	validations  *prometheus.CounterVec
	buildInfo    *prometheus.GaugeVec
}

// New builds a metrics registry with the standard OpenLicensd collectors.
// poolStat may be nil to skip database pool metrics.
func New(version string, poolStat func() *PoolStat) *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		registry: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests handled by the API server.",
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),
		validations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "license_validations_total",
			Help:      "Total license validation attempts.",
		}, []string{"result", "reason"}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "Build version information.",
		}, []string{"version"}),
	}

	reg.MustRegister(
		m.httpRequests,
		m.httpDuration,
		m.validations,
		m.buildInfo,
	)

	if poolStat != nil {
		reg.MustRegister(newPoolCollector(poolStat))
	}

	m.buildInfo.WithLabelValues(version).Set(1)
	return m
}

// Handler returns an HTTP handler that serves Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Gatherer returns the underlying registry for tests and advanced integrations.
func (m *Metrics) Gatherer() prometheus.Gatherer {
	return m.registry
}

// RecordValidation increments the license validation counter.
func (m *Metrics) RecordValidation(valid bool, reason string) {
	result := "invalid"
	if valid {
		result = "valid"
	}
	if reason == "" {
		if valid {
			reason = "ok"
		} else {
			reason = "unknown"
		}
	}
	m.validations.WithLabelValues(result, reason).Inc()
}

func (m *Metrics) observeHTTP(method, route string, status int, durationSeconds float64) {
	statusLabel := strconv.Itoa(status)
	m.httpRequests.WithLabelValues(method, route, statusLabel).Inc()
	m.httpDuration.WithLabelValues(method, route).Observe(durationSeconds)
}

// Middleware records HTTP request metrics keyed by chi route pattern.
func Middleware(m *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := routePattern(r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			m.observeHTTP(r.Method, route, status, time.Since(start).Seconds())
		})
	}
}

func routePattern(r *http.Request) string {
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "other"
}
