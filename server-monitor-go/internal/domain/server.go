// Package domain holds the core types shared across the monitoring backend.
// It depends on nothing else in the codebase: every other layer (storage,
// service, transport) may import domain, but domain imports none of them.
package domain

// Status is a server's lifecycle state. A server reports "running" while an
// agent is actively pushing metrics, transitions to "stopped" when it goes
// silent, and sits in "maintenance" (shown as "Pending" in the UI) between
// being registered by an admin and its agent connecting for the first time.
type Status string

const (
	StatusRunning     Status = "running"
	StatusStopped     Status = "stopped"
	StatusMaintenance Status = "maintenance"
)

// Valid reports whether s is one of the known statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusRunning, StatusStopped, StatusMaintenance:
		return true
	default:
		return false
	}
}

// Server is the full record for a monitored machine. The JSON tags match the
// contract expected by the Next.js frontend and the agent payload, so the same
// struct serves both API responses and the ingest path. This wire shape is
// frozen: deployed agents cannot be updated atomically.
type Server struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Location  string `json:"location"`
	IPAddress string `json:"ip_address"`
	// Hostname is the machine's OS hostname; TailscaleIP is its address on the
	// tailnet (100.64.0.0/10). Both surface in the dashboard for identification.
	Hostname    string  `json:"hostname"`
	TailscaleIP string  `json:"tailscale_ip"`
	Status      Status  `json:"status"`
	Uptime      int64   `json:"uptime"`
	NetworkIn   float64 `json:"network_in"`
	NetworkOut  float64 `json:"network_out"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	Disk        float64 `json:"disk"`
	OSType      string  `json:"os_type"`
	CPUInfo     string  `json:"cpu_info"`
	TotalMemory float64 `json:"total_memory"`
	TotalDisk   float64 `json:"total_disk"`
	OrderIndex  int     `json:"order_index"`
	FirstSeen   string  `json:"first_seen"`
	LastUpdate  string  `json:"last_update"`
}

// Client is an entry in the allow-list. Only named clients may report metrics.
type Client struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Alert severities and types.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"

	AlertStatusChange = "status_change"
	AlertThreshold    = "threshold"
)

// Alert is a recorded notable event: a status transition or a threshold breach.
// AcknowledgedAt is empty until an admin acknowledges it.
type Alert struct {
	ID             int64  `json:"id"`
	Type           string `json:"type"`
	ServerID       string `json:"server_id"`
	ServerName     string `json:"server_name"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	CreatedAt      string `json:"created_at"`
	AcknowledgedAt string `json:"acknowledged_at"`
}

// UnknownAgent records an agent that reported under a name not on the
// allow-list (a rejected ingest). Surfaced to admins to diagnose misnamed
// agents instead of needing a packet capture.
type UnknownAgent struct {
	Name       string `json:"name"`
	RemoteAddr string `json:"remote_addr"`
	LastSeen   string `json:"last_seen"`
	Count      int64  `json:"count"`
}
