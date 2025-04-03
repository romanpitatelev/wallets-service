package store

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	system    = "wallets_service"
	subsystem = "db"
)

type metrics struct {
	dbResponseDuration *prometheus.HistogramVec
}

func newMetrics() *metrics {
	return &metrics{
		dbResponseDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: system,
			Subsystem: subsystem,
			Name:      "db_response_duration_seconds",
			Help:      "Time spent doing db response",
		},
			[]string{"method"}),
	}
}
