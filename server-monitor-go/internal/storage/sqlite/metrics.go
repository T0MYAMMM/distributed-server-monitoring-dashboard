package sqlite

import (
	"database/sql"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// rollupBucket is the rollup granularity in seconds (5 minutes).
const rollupBucket = 300

// InsertSample appends one raw telemetry sample for a server.
func (s *Store) InsertSample(serverID string, m domain.MetricSample) error {
	_, err := s.db.Exec(`INSERT INTO metrics_samples
		(server_id, ts, cpu, mem, disk, net_in, net_out)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		serverID, m.Ts, m.CPU, m.Memory, m.Disk, m.NetworkIn, m.NetworkOut)
	return err
}

// RawSeries returns per-bucket averages from raw samples for a server over
// [from, to), bucketed to bucketSize seconds.
func (s *Store) RawSeries(serverID string, from, to, bucketSize int64) ([]domain.MetricSample, error) {
	return s.series(`metrics_samples`, `ts`, serverID, from, to, bucketSize)
}

// RollupSeries returns per-bucket averages from the 5-minute rollups for a
// server over [from, to), re-bucketed to bucketSize seconds.
func (s *Store) RollupSeries(serverID string, from, to, bucketSize int64) ([]domain.MetricSample, error) {
	return s.series(`metrics_rollup_5m`, `bucket`, serverID, from, to, bucketSize)
}

func (s *Store) series(table, tsCol, serverID string, from, to, bucketSize int64) ([]domain.MetricSample, error) {
	if bucketSize < 1 {
		bucketSize = 1
	}
	q := `SELECT (` + tsCol + `/?)*? AS b,
			AVG(cpu), AVG(mem), AVG(disk), AVG(net_in), AVG(net_out)
		FROM ` + table + `
		WHERE server_id = ? AND ` + tsCol + ` >= ? AND ` + tsCol + ` < ?
		GROUP BY b ORDER BY b`
	rows, err := s.db.Query(q, bucketSize, bucketSize, serverID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.MetricSample, 0)
	for rows.Next() {
		var m domain.MetricSample
		if err := rows.Scan(&m.Ts, &m.CPU, &m.Memory, &m.Disk, &m.NetworkIn, &m.NetworkOut); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// FleetAverage returns the fleet-wide average cpu/mem/disk and combined network
// (in+out) over [from, to). useRollup selects the rollup table for long ranges.
func (s *Store) FleetAverage(from, to int64, useRollup bool) (cpu, mem, disk, net float64, err error) {
	table, tsCol := `metrics_samples`, `ts`
	if useRollup {
		table, tsCol = `metrics_rollup_5m`, `bucket`
	}
	var c, m2, d, n sql.NullFloat64
	err = s.db.QueryRow(`SELECT AVG(cpu), AVG(mem), AVG(disk), AVG(net_in+net_out)
		FROM `+table+` WHERE `+tsCol+` >= ? AND `+tsCol+` < ?`, from, to).
		Scan(&c, &m2, &d, &n)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return c.Float64, m2.Float64, d.Float64, n.Float64, nil
}

// UptimeBuckets returns the count of distinct 5-minute buckets in [from, to)
// that contain at least one sample, and the total number of buckets in the
// window, for an approximate fleet uptime percentage driven by real history.
func (s *Store) UptimeBuckets(from, to int64) (withData, total int64, err error) {
	err = s.db.QueryRow(`SELECT COUNT(DISTINCT (ts/?)) FROM metrics_samples
		WHERE ts >= ? AND ts < ?`, rollupBucket, from, to).Scan(&withData)
	if err != nil {
		return 0, 0, err
	}
	total = (to - from) / rollupBucket
	return withData, total, nil
}

// ServerUptimeBuckets is the per-server analogue of UptimeBuckets, counting
// distinct 5-minute buckets with data across both the raw samples and the
// rollups (so long ranges stay accurate after raw samples are pruned).
func (s *Store) ServerUptimeBuckets(serverID string, from, to int64) (withData, total int64, err error) {
	err = s.db.QueryRow(`SELECT COUNT(DISTINCT b) FROM (
			SELECT (ts/?) AS b FROM metrics_samples
				WHERE server_id = ? AND ts >= ? AND ts < ?
			UNION
			SELECT (bucket/?) AS b FROM metrics_rollup_5m
				WHERE server_id = ? AND bucket >= ? AND bucket < ?
		)`,
		rollupBucket, serverID, from, to,
		rollupBucket, serverID, from, to).Scan(&withData)
	if err != nil {
		return 0, 0, err
	}
	total = (to - from) / rollupBucket
	return withData, total, nil
}

// CompactRollups aggregates raw samples into 5-minute rollups, then prunes raw
// samples older than rawCutoff and rollups older than rollupCutoff. Rollup runs
// before pruning so every pruned bucket keeps a final accurate aggregate.
func (s *Store) CompactRollups(rawCutoff, rollupCutoff int64) error {
	if _, err := s.db.Exec(`INSERT OR REPLACE INTO metrics_rollup_5m
		(server_id, bucket, cpu, mem, disk, net_in, net_out)
		SELECT server_id, (ts/?)*? AS bucket,
			AVG(cpu), AVG(mem), AVG(disk), AVG(net_in), AVG(net_out)
		FROM metrics_samples GROUP BY server_id, bucket`,
		rollupBucket, rollupBucket); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM metrics_samples WHERE ts < ?`, rawCutoff); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM metrics_rollup_5m WHERE bucket < ?`, rollupCutoff)
	return err
}
