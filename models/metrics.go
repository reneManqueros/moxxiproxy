package models

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"strings"
)

type MetricPayload struct {
	Protocol         string
	UserID           string
	BytesTransferred int64
	Direction        string
	Region           string
	Host             string
	Project          string
}

var vecFields = []string{
	"user_id",
	"region",
	"protocol",
	"host",
	"project",
}

var vecRXBytes = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "moxxi_rx_bytes",
		Help: "The total payload received",
	},
	vecFields,
)
var vecTXBytes = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "moxxi_tx_bytes",
		Help: "The total payload sent",
	},
	vecFields,
)

func (p *Proxy) LogPayload(payload MetricPayload) {
	if hostParts := strings.Split(payload.Host, ":"); len(hostParts) > 0 {
		payload.Host = hostParts[0]
	}
	if p.MetricsLogger == "stdout" {
		log.Trace().
			Str("protocol", payload.Protocol).
			Str("Direction", payload.Direction).
			Str("UserID", payload.UserID).
			Str("Region", payload.Region).
			Str("Host", payload.Host).
			Str("Project", payload.Project).
			Int64("BytesTransferred", payload.BytesTransferred).
			Msg("MetricRow")
	}
	if p.MetricsLogger == "prometheus" {
		labels := prometheus.Labels{
			"user_id":  payload.UserID,
			"region":   payload.Region,
			"protocol": payload.Protocol,
			"host":     payload.Host,
			"project":  payload.Project,
		}
		if payload.Direction == "rx" {
			vecRXBytes.With(labels).Add(float64(payload.BytesTransferred))
		}
		if payload.Direction == "tx" {
			vecTXBytes.With(labels).Add(float64(payload.BytesTransferred))
		}
	}
}
