package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolStat is a snapshot of pgxpool statistics.
type PoolStat = pgxpool.Stat

type poolCollector struct {
	stat func() *PoolStat
}

func newPoolCollector(stat func() *PoolStat) prometheus.Collector {
	return &poolCollector{stat: stat}
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc("acquired_connections", "Number of currently acquired connections in the pool.")
	ch <- c.desc("idle_connections", "Number of idle connections in the pool.")
	ch <- c.desc("total_connections", "Total number of connections in the pool.")
	ch <- c.desc("max_connections", "Maximum size of the connection pool.")
	ch <- c.desc("acquire_count_total", "Total number of successful connection acquires from the pool.")
	ch <- c.desc("empty_acquire_count_total", "Total number of acquires that waited because the pool was empty.")
	ch <- c.desc("canceled_acquire_count_total", "Total number of acquires canceled by context.")
	ch <- c.desc("new_connections_count_total", "Total number of new connections opened.")
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	stat := c.stat()
	acquired := 0.0
	idle := 0.0
	total := 0.0
	max := 0.0
	acquireCount := 0.0
	emptyAcquire := 0.0
	canceledAcquire := 0.0
	newConns := 0.0
	if stat != nil {
		acquired = float64(stat.AcquiredConns())
		idle = float64(stat.IdleConns())
		total = float64(stat.TotalConns())
		max = float64(stat.MaxConns())
		acquireCount = float64(stat.AcquireCount())
		emptyAcquire = float64(stat.EmptyAcquireCount())
		canceledAcquire = float64(stat.CanceledAcquireCount())
		newConns = float64(stat.NewConnsCount())
	}

	ch <- c.gauge("acquired_connections", acquired)
	ch <- c.gauge("idle_connections", idle)
	ch <- c.gauge("total_connections", total)
	ch <- c.gauge("max_connections", max)
	ch <- c.counter("acquire_count_total", acquireCount)
	ch <- c.counter("empty_acquire_count_total", emptyAcquire)
	ch <- c.counter("canceled_acquire_count_total", canceledAcquire)
	ch <- c.counter("new_connections_count_total", newConns)
}

func (c *poolCollector) desc(name, help string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "db_pool", name),
		help,
		nil,
		nil,
	)
}

func (c *poolCollector) gauge(name string, value float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(c.desc(name, ""), prometheus.GaugeValue, value)
}

func (c *poolCollector) counter(name string, value float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(c.desc(name, ""), prometheus.CounterValue, value)
}
