package domain

import (
	"crypto/md5"
	"encoding/hex"
)

// ServerID derives the stable public id for a client name. It is md5(name) so
// the frontend can address rows and metrics samples without exposing internals.
func ServerID(name string) string {
	sum := md5.Sum([]byte(name))
	return hex.EncodeToString(sum[:])
}

// MetricSample is one point of time-series telemetry for a server. Ts is unix
// seconds (UTC), which keeps range queries and bucketing simple integer math.
type MetricSample struct {
	Ts         int64   `json:"ts"`
	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"memory"`
	Disk       float64 `json:"disk"`
	NetworkIn  float64 `json:"network_in"`
	NetworkOut float64 `json:"network_out"`
}

// FleetMetric is one fleet-wide KPI: its value over the current window and the
// delta versus the previous window of equal length (for trend badges).
type FleetMetric struct {
	Value float64 `json:"value"`
	Delta float64 `json:"delta"`
}

// FleetSummary aggregates fleet KPIs for the dashboard cards.
type FleetSummary struct {
	RangeSeconds  int64       `json:"range_seconds"`
	ActiveServers int         `json:"active_servers"`
	TotalServers  int         `json:"total_servers"`
	CPU           FleetMetric `json:"cpu"`
	Memory        FleetMetric `json:"memory"`
	Disk          FleetMetric `json:"disk"`
	Network       FleetMetric `json:"network"`
	// UptimePercent is the share of the window the fleet had at least one
	// running server reporting, expressed 0-100.
	UptimePercent float64 `json:"uptime_percent"`
}
