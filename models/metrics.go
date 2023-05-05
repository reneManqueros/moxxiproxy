package models

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	payloadTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payload_total",
			Help: "The total payload",
		},
		[]string{"user_id"},
	)
	// other metrics go here
)

func addPayloadSize(userID string, bytes float64) {
	payloadTotal.With(prometheus.Labels{"user_id": userID}).Add(bytes)
}
