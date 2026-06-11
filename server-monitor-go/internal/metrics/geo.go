package metrics

import (
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"
)

// lookupLocation resolves the host's two-letter country code via ip-api.com,
// falling back to "UN" (unknown) on any failure. Best-effort and cached.
func (c *Collector) lookupLocation() string {
	resp, err := c.http.Get("http://ip-api.com/json/")
	if err != nil {
		return "UN"
	}
	defer resp.Body.Close()

	var data struct {
		Status      string `json:"status"`
		CountryCode string `json:"countryCode"`
	}
	if json.NewDecoder(resp.Body).Decode(&data) == nil &&
		data.Status == "success" && data.CountryCode != "" {
		return data.CountryCode
	}
	return "UN"
}

// lookupIP returns the host's public IPv4 (and IPv6 when available, joined by
// "/"), falling back to the primary outbound local address, then 127.0.0.1.
func (c *Collector) lookupIP() string {
	ipv4 := c.fetch("https://api.ipify.org")
	ipv6 := c.fetch("https://api6.ipify.org")

	if ipv4 == "" {
		ipv4 = localIP()
	}
	switch {
	case ipv4 != "" && ipv6 != "" && ipv4 != ipv6:
		return ipv4 + "/" + ipv6
	case ipv4 != "":
		return ipv4
	case ipv6 != "":
		return ipv6
	default:
		return "127.0.0.1"
	}
}

func (c *Collector) fetch(url string) string {
	resp, err := c.http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// localIP discovers the primary outbound IP without sending traffic.
func localIP() string {
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 2*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
