// Package metrics holds the time-series use cases: recording a sample on every
// accepted report, serving per-server history and fleet summaries for the
// dashboard, and the periodic rollup/prune compaction job. It depends only on
// domain and a Repo interface it defines.
package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

const (
	maxPoints       = 500            // cap per series; downsample server-side
	rawRetention    = 48 * 3600      // keep raw samples ~48h
	rollupRetention = 30 * 24 * 3600 // keep 5-minute rollups ~30 days
	rollupBucket    = 300            // 5 minutes
)

// Repo is the metrics persistence the service needs; sqlite.Store satisfies it.
type Repo interface {
	InsertSample(serverID string, m domain.MetricSample) error
	RawSeries(serverID string, from, to, bucket int64) ([]domain.MetricSample, error)
	RollupSeries(serverID string, from, to, bucket int64) ([]domain.MetricSample, error)
	FleetAverage(from, to int64, useRollup bool) (cpu, mem, disk, net float64, err error)
	UptimeBuckets(from, to int64) (withData, total int64, err error)
	ServerUptimeBuckets(serverID string, from, to int64) (withData, total int64, err error)
	CompactRollups(rawCutoff, rollupCutoff int64) error
	ListServers() ([]domain.Server, error)
}

// Clock abstracts time for testable windows and compaction.
type Clock interface{ Now() time.Time }

// SystemClock is the production Clock.
type SystemClock struct{}

// Now returns the current time.
func (SystemClock) Now() time.Time { return time.Now() }

// Service implements the metrics use cases.
type Service struct {
	repo  Repo
	clock Clock
	log   *slog.Logger
}

// New constructs the metrics service.
func New(repo Repo, clock Clock, log *slog.Logger) *Service {
	if clock == nil {
		clock = SystemClock{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, clock: clock, log: log}
}

// rangeSeconds maps the API range token to a window length. Unknown values fall
// back to 24h.
func rangeSeconds(r string) int64 {
	switch r {
	case "1h":
		return 3600
	case "6h":
		return 6 * 3600
	case "7d":
		return 7 * 24 * 3600
	case "24h", "":
		return 24 * 3600
	default:
		return 24 * 3600
	}
}

// Record persists one telemetry sample from an accepted agent report.
func (s *Service) Record(report domain.Server) error {
	return s.repo.InsertSample(domain.ServerID(report.Name), domain.MetricSample{
		Ts:         s.clock.Now().Unix(),
		CPU:        report.CPU,
		Memory:     report.Memory,
		Disk:       report.Disk,
		NetworkIn:  report.NetworkIn,
		NetworkOut: report.NetworkOut,
	})
}

// History returns a downsampled series for a server over the requested range,
// capped at maxPoints. Long ranges read from the 5-minute rollups.
func (s *Service) History(serverID, rng string) ([]domain.MetricSample, error) {
	secs := rangeSeconds(rng)
	now := s.clock.Now().Unix()
	from := now - secs
	// The series query is half-open [from, to); add a second so a sample taken
	// in the current second (e.g. the report that just arrived) is included.
	to := now + 1
	bucket := ceilDiv(secs, maxPoints)
	if secs > rawRetention {
		if bucket < rollupBucket {
			bucket = rollupBucket
		}
		return s.repo.RollupSeries(serverID, from, to, bucket)
	}
	return s.repo.RawSeries(serverID, from, to, bucket)
}

// Summary returns fleet KPIs over the range plus deltas versus the previous
// window of equal length, for the dashboard cards and trend badges.
func (s *Service) Summary(rng string) (domain.FleetSummary, error) {
	secs := rangeSeconds(rng)
	now := s.clock.Now().Unix()
	curFrom := now - secs
	prevFrom := now - 2*secs
	useRollup := secs > rawRetention

	cCPU, cMem, cDisk, cNet, err := s.repo.FleetAverage(curFrom, now, useRollup)
	if err != nil {
		return domain.FleetSummary{}, err
	}
	pCPU, pMem, pDisk, pNet, err := s.repo.FleetAverage(prevFrom, curFrom, useRollup)
	if err != nil {
		return domain.FleetSummary{}, err
	}

	servers, err := s.repo.ListServers()
	if err != nil {
		return domain.FleetSummary{}, err
	}
	active := 0
	for _, sv := range servers {
		if sv.Status == domain.StatusRunning {
			active++
		}
	}

	withData, total, err := s.repo.UptimeBuckets(curFrom, now)
	if err != nil {
		return domain.FleetSummary{}, err
	}
	uptime := 0.0
	if total > 0 {
		uptime = float64(withData) / float64(total) * 100
		if uptime > 100 {
			uptime = 100
		}
	}

	return domain.FleetSummary{
		RangeSeconds:  secs,
		ActiveServers: active,
		TotalServers:  len(servers),
		CPU:           domain.FleetMetric{Value: cCPU, Delta: cCPU - pCPU},
		Memory:        domain.FleetMetric{Value: cMem, Delta: cMem - pMem},
		Disk:          domain.FleetMetric{Value: cDisk, Delta: cDisk - pDisk},
		Network:       domain.FleetMetric{Value: cNet, Delta: cNet - pNet},
		UptimePercent: uptime,
	}, nil
}

// ServerStats returns per-server analytics over the range: current resource
// levels, observed uptime, and a disk capacity projection from the disk trend.
func (s *Service) ServerStats(rng string) ([]domain.ServerStat, error) {
	secs := rangeSeconds(rng)
	now := s.clock.Now().Unix()
	from := now - secs

	servers, err := s.repo.ListServers()
	if err != nil {
		return nil, err
	}
	out := make([]domain.ServerStat, 0, len(servers))
	for _, sv := range servers {
		stat := domain.ServerStat{
			ID: sv.ID, Name: sv.Name, Status: sv.Status,
			CPU: sv.CPU, Memory: sv.Memory, Disk: sv.Disk,
			DiskDaysToFull: -1,
		}
		withData, total, err := s.repo.ServerUptimeBuckets(sv.ID, from, now)
		if err != nil {
			return nil, err
		}
		if total > 0 {
			stat.UptimePercent = float64(withData) / float64(total) * 100
			if stat.UptimePercent > 100 {
				stat.UptimePercent = 100
			}
		}
		if series, err := s.History(sv.ID, rng); err == nil {
			stat.DiskDaysToFull = diskDaysToFull(series, sv.Disk)
		}
		out = append(out, stat)
	}
	return out, nil
}

// diskDaysToFull fits a line to the disk series and projects days until 100%.
// Returns -1 when the trend is flat/falling or there is too little data.
func diskDaysToFull(series []domain.MetricSample, currentDisk float64) float64 {
	if len(series) < 3 {
		return -1
	}
	// Least-squares slope of disk over time (percent per second).
	var n, sx, sy, sxx, sxy float64
	t0 := float64(series[0].Ts)
	for _, m := range series {
		x := float64(m.Ts) - t0
		y := m.Disk
		n++
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		return -1
	}
	slopePerSec := (n*sxy - sx*sy) / denom
	perDay := slopePerSec * 86400
	if perDay <= 0.01 { // essentially flat or falling
		return -1
	}
	remaining := 100 - currentDisk
	if remaining <= 0 {
		return 0
	}
	return remaining / perDay
}

// Compact runs one rollup/prune cycle.
func (s *Service) Compact() error {
	now := s.clock.Now().Unix()
	rawCutoff := (now - rawRetention) / rollupBucket * rollupBucket // bucket-aligned
	rollupCutoff := now - rollupRetention
	return s.repo.CompactRollups(rawCutoff, rollupCutoff)
}

// RunCompactor compacts every interval until ctx is cancelled.
func (s *Service) RunCompactor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Compact(); err != nil {
				s.log.Error("metrics compaction failed", "err", err)
			}
		}
	}
}

func ceilDiv(a int64, b int64) int64 {
	if b <= 0 {
		return 1
	}
	v := (a + b - 1) / b
	if v < 1 {
		return 1
	}
	return v
}
