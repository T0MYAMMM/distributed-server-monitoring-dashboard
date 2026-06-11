// Package models defines the core data types shared across the monitoring
// backend and the metrics protocol used by agents.
package models

// Status values a server can hold. A server reports "running" while an agent
// is actively pushing metrics, transitions to "stopped" when it goes silent,
// and sits in "maintenance" (shown as "Pending") between being registered by
// an admin and its agent connecting for the first time.
const (
	StatusRunning     = "running"
	StatusStopped     = "stopped"
	StatusMaintenance = "maintenance"
)

// Server is the full record for a monitored machine. The JSON tags match the
// contract expected by the Next.js frontend and the agent payload, so the same
// struct serves both the API responses and the ingest path.
type Server struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Location    string  `json:"location"`
	IPAddress   string  `json:"ip_address"`
	// Hostname is the machine's OS hostname; TailscaleIP is its address on the
	// tailnet (100.64.0.0/10). Both surface in the dashboard for identification
	// and are not masked, since they only identify nodes within the tailnet.
	Hostname    string  `json:"hostname"`
	TailscaleIP string  `json:"tailscale_ip"`
	Status      string  `json:"status"`
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
