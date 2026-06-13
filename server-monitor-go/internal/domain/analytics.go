package domain

// ServerStat is a per-server analytics row: current resource levels, observed
// uptime over the window, and a simple disk capacity projection.
type ServerStat struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Status        Status  `json:"status"`
	CPU           float64 `json:"cpu"`
	Memory        float64 `json:"memory"`
	Disk          float64 `json:"disk"`
	UptimePercent float64 `json:"uptime_percent"`
	// DiskDaysToFull projects days until disk reaches 100% from the window's
	// trend. -1 means flat or falling (no projection).
	DiskDaysToFull float64 `json:"disk_days_to_full"`
}

// LogVolumePoint is the per-bucket log count split by level, for the volume
// histogram over time.
type LogVolumePoint struct {
	Ts    int64 `json:"ts"`
	Debug int   `json:"debug"`
	Info  int   `json:"info"`
	Warn  int   `json:"warn"`
	Error int   `json:"error"`
}

// ModuleStat counts log lines for one module, with the error subset, for the
// "top error sources" view.
type ModuleStat struct {
	Module string `json:"module"`
	Total  int    `json:"total"`
	Errors int    `json:"errors"`
}
