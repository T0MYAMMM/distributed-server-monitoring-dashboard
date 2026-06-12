// Package logtail tails configured log files on a monitored machine and ships
// new lines to the hub in the log-geulis format (TS | LEVEL | MODULE | MESSAGE).
// It is additive: the agent only runs it when --logs is set, so existing agents
// are unaffected.
package logtail

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Tailer follows a set of files and forwards appended lines to the hub.
type Tailer struct {
	endpoint string // hub base URL, e.g. http://hub:5000
	name     string // this node's registered name
	paths    []string
	offsets  map[string]int64
	client   *http.Client
}

// New builds a Tailer for the given hub base URL, node name, and file paths.
func New(hubBase, name string, paths []string) *Tailer {
	return &Tailer{
		endpoint: strings.TrimRight(hubBase, "/"),
		name:     name,
		paths:    paths,
		offsets:  make(map[string]int64),
		client:   &http.Client{Timeout: 8 * time.Second},
	}
}

// Run primes each file's offset to its current end (so only new lines ship),
// then polls every interval, batching and shipping appended lines until ctx is
// cancelled.
func (t *Tailer) Run(ctx context.Context, interval time.Duration) {
	for _, p := range t.paths {
		if fi, err := os.Stat(p); err == nil {
			t.offsets[p] = fi.Size()
		}
	}
	log.Printf("log tailer: shipping %d path(s) to %s/api/v1/logs", len(t.paths), t.endpoint)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		var batch []domain.LogLine
		for _, p := range t.paths {
			batch = append(batch, t.readNew(p)...)
		}
		if len(batch) > 0 {
			t.ship(batch)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// readNew returns log lines appended to path since the last read, advancing the
// stored offset. A file that shrank (truncated or rotated) is re-read from the
// start.
func (t *Tailer) readNew(path string) []domain.LogLine {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil
	}
	size := fi.Size()
	off := t.offsets[path]
	if size < off {
		off = 0 // rotated or truncated
	}
	if size == off {
		return nil
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil
	}

	r := bufio.NewReader(f)
	var lines []domain.LogLine
	var consumed int64
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			break // partial trailing line (no newline yet): leave for next read
		}
		consumed += int64(len(s))
		line := strings.TrimRight(s, "\r\n")
		if line == "" {
			continue
		}
		lines = append(lines, parseLine(line, path))
	}
	t.offsets[path] = off + consumed
	return lines
}

// parseLine maps a raw line to the structured format. Lines already in the
// log-geulis format (TS | LEVEL | MODULE | MESSAGE) are split; anything else is
// kept verbatim as the message at INFO, tagged with the file's base name.
func parseLine(line, path string) domain.LogLine {
	parts := strings.SplitN(line, " | ", 4)
	if len(parts) == 4 {
		return domain.LogLine{
			Ts:         strings.TrimSpace(parts[0]),
			Level:      strings.ToUpper(strings.TrimSpace(parts[1])),
			Module:     strings.TrimSpace(parts[2]),
			Message:    parts[3],
			SourceFile: path,
		}
	}
	return domain.LogLine{
		Ts:         time.Now().UTC().Format(time.RFC3339),
		Level:      domain.LogInfo,
		Module:     filepath.Base(path),
		Message:    line,
		SourceFile: path,
	}
}

func (t *Tailer) ship(lines []domain.LogLine) {
	body, err := json.Marshal(map[string]any{"server": t.name, "lines": lines})
	if err != nil {
		return
	}
	resp, err := t.client.Post(t.endpoint+"/api/v1/logs", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("log tailer: ship failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("log tailer: ship status %d", resp.StatusCode)
	}
}
