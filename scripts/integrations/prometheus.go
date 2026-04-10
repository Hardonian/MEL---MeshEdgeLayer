// Package integrations provides Prometheus metrics for MEL
package integrations

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MeshNodes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mel_mesh_nodes_total",
		Help": "Total mesh nodes",
	}, []string{"status"})

	SignalStrength = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mel_signal_strength_dbm",
		Help:   "Signal strength in dBm",
		Buckets: []float64{-90, -75, -60, -50},
	}, []string{"node"})

	PacketLoss = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mel_packet_loss_total",
		Help": "Total packet loss",
	}, []string{"source", "destination", "reason"})

	RouteChanges = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mel_route_changes_total",
		Help": "Route changes",
	}, []string{"reason"})

	DeviceConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mel_device_connections_active",
		Help: "Active device connections",
	})
)
