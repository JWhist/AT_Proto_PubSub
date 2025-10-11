package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WebsocketConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_connections",
		Help: "Current number of active WebSocket connections",
	})
	MessagesSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "messages_sent_total",
		Help: "Total number of messages sent to clients",
	}, []string{"keyword"})
	MessagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_received_total",
		Help: "Total number of messages received from the firehose",
	})
	FiltersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "filters_created_total",
		Help: "Total number of filters created",
	})
	FiltersDeleted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "filters_deleted_total",
		Help: "Total number of filters deleted",
	})
)

func init() {
	prometheus.MustRegister(
		WebsocketConnections,
		MessagesSent,
		MessagesReceived,
		FiltersCreated,
		FiltersDeleted,
	)
}
