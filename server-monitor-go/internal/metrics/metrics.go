// Package metrics collects system resource statistics on the host where the
// agent runs, using gopsutil so the agent is a single static binary with no
// Python or external tools required.
package metrics

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Collector gathers metrics and caches values that rarely change (location,
// public IP, CPU model, OS, machine id) to keep each sample cheap.
type Collector struct {
	name string

	machineID   string
	cpuModel    string
	osType      string
	srvType     string
	location    string
	ipAddr      string
	hostname    string
	tailscaleIP string

	lastNet  net.IOCountersStat
	lastTime time.Time
	netReady bool

	http *http.Client
}

// NewCollector builds a Collector for the given node name and primes the
// static fields once.
func NewCollector(name string) *Collector {
	c := &Collector{
		name: name,
		http: &http.Client{Timeout: 5 * time.Second},
	}
	c.machineID = machineID(name)
	c.cpuModel = cpuModel()
	c.osType = osType()
	c.srvType = serverType()
	c.location = c.lookupLocation()
	c.ipAddr = c.lookupIP()
	c.hostname = hostname()
	c.tailscaleIP = tailscaleIP()
	return c
}

// Sample returns a fresh metrics snapshot for the host.
func (c *Collector) Sample() domain.Server {
	cpuPct := cpuPercent()
	vm, _ := mem.VirtualMemory()
	diskPct, totalDiskGB := diskUsage()
	in, out := c.netSpeed()
	up, _ := host.Uptime()

	var memPct, totalMemGB float64
	if vm != nil {
		memPct = vm.UsedPercent
		totalMemGB = float64(vm.Total) / (1024 * 1024 * 1024)
	}

	return domain.Server{
		ID:          c.machineID,
		Name:        c.name,
		Type:        c.srvType,
		Location:    c.location,
		IPAddress:   c.ipAddr,
		Hostname:    c.hostname,
		TailscaleIP: c.tailscaleIP,
		Uptime:      int64(up),
		NetworkIn:   round2(in),
		NetworkOut:  round2(out),
		CPU:         round2(cpuPct),
		Memory:      round2(memPct),
		Disk:        round2(diskPct),
		OSType:      c.osType,
		CPUInfo:     c.cpuModel,
		TotalMemory: round2(totalMemGB),
		TotalDisk:   round2(totalDiskGB),
	}
}

// netSpeed returns bytes/sec in and out since the previous call.
func (c *Collector) netSpeed() (in, out float64) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0, 0
	}
	cur := counters[0]
	now := time.Now()

	if !c.netReady {
		c.lastNet, c.lastTime, c.netReady = cur, now, true
		return 0, 0
	}
	secs := now.Sub(c.lastTime).Seconds()
	if secs <= 0 {
		return 0, 0
	}
	in = float64(cur.BytesRecv-c.lastNet.BytesRecv) / secs
	out = float64(cur.BytesSent-c.lastNet.BytesSent) / secs
	c.lastNet, c.lastTime = cur, now
	if in < 0 {
		in = 0
	}
	if out < 0 {
		out = 0
	}
	return in, out
}

func cpuPercent() float64 {
	// Non-blocking: percentage since the previous call.
	pct, err := cpu.Percent(0, false)
	if err != nil || len(pct) == 0 {
		return 0
	}
	return pct[0]
}

func cpuModel() string {
	threads, _ := cpu.Counts(true)
	info, err := cpu.Info()
	if err != nil || len(info) == 0 || info[0].ModelName == "" {
		return fmt.Sprintf("CPU (%d threads)", threads)
	}
	return fmt.Sprintf("%s (%d threads)", strings.TrimSpace(info[0].ModelName), threads)
}

func diskUsage() (usedPct, totalGB float64) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return 0, 0
	}
	var total, used uint64
	for _, p := range parts {
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u == nil {
			continue
		}
		total += u.Total
		used += u.Used
	}
	if total == 0 {
		return 0, 0
	}
	return float64(used) / float64(total) * 100, float64(total) / (1024 * 1024 * 1024)
}

// osType returns a friendly OS / distribution name.
func osType() string {
	info, err := host.Info()
	if err != nil {
		return "Unknown"
	}
	if strings.EqualFold(info.OS, "windows") {
		if info.Platform != "" {
			return "Windows " + info.PlatformVersion
		}
		return "Windows"
	}
	if info.Platform != "" {
		return strings.Title(info.Platform) //nolint:staticcheck // simple capitalization
	}
	return strings.Title(info.OS) //nolint:staticcheck
}

// serverType distinguishes virtualized hosts ("VPS") from bare metal
// ("Dedicated Server") using gopsutil's virtualization detection.
func serverType() string {
	info, err := host.Info()
	if err == nil && info.VirtualizationSystem != "" && info.VirtualizationRole == "guest" {
		return "VPS"
	}
	return "Dedicated Server"
}

// machineID derives a stable id from hostname + first MAC, falling back to the
// node name. (The backend keys updates by name, so this is informational.)
func machineID(name string) string {
	hn, _ := os.Hostname()
	mac := firstMAC()
	seed := hn + "-" + mac
	if hn == "" && mac == "" {
		seed = name
	}
	sum := md5.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func firstMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.HardwareAddr != "" && !strings.HasPrefix(iface.HardwareAddr, "00:00:00") {
			return iface.HardwareAddr
		}
	}
	return ""
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
