package walletsservice

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "wallets_service"
	subsystem = "service"
)

type metrics struct {
	updateFailed    *prometheus.CounterVec
	updateCompleted *prometheus.CounterVec
	updateDuration  *prometheus.HistogramVec
}

func newMetrics() *metrics {
	return &metrics{
		updateFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "update_failed_total",
				Help:      "Number of failed updates",
			},
			[]string{"endpoint"}),
		updateCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "update_completed_total",
				Help:      "Number of completed updates",
			},
			[]string{"endpoint"}),
		updateDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "update_duration_seconds",
				Help:      "Time spent making update",
			},
			[]string{"method"}),
	}
}
