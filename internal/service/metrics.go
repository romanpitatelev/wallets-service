package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "wallets_service"
	subsystem = "service"
)

type metrics struct {
	txFailed    *prometheus.CounterVec
	txCompleted *prometheus.CounterVec
	txDuration  *prometheus.HistogramVec
}

func newMetrics() *metrics {
	return &metrics{
		txFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "tx_failed_total",
				Help:      "Number of failed transactions",
			},
			[]string{"endpoint"}),
		txCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "tx_completed_total",
				Help:      "Number of completed transactions",
			},
			[]string{"endpoint"}),
		txDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "tx_duration_seconds",
				Help:      "Time spent making transaction",
			},
			[]string{"method"}),
	}
}
