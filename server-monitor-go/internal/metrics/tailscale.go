package metrics

import (
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// hostname returns the machine's OS hostname, or "unknown" if unavailable.
func hostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown"
}

// tailscaleIP returns the host's Tailscale IPv4. It first asks the `tailscale`
// CLI (authoritative when installed), then falls back to scanning local
// interfaces for an address in the Tailscale CGNAT range (100.64.0.0/10).
// Returns "" when the host is not on a tailnet.
func tailscaleIP() string {
	if ip := tailscaleIPFromCLI(); ip != "" {
		return ip
	}
	return tailscaleIPFromInterfaces()
}

func tailscaleIPFromCLI() string {
	path, err := exec.LookPath("tailscale")
	if err != nil {
		return ""
	}
	cmd := exec.Command(path, "ip", "-4")
	// Guard against a hung CLI.
	done := make(chan struct{})
	var out []byte
	go func() { out, _ = cmd.Output(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if ip := strings.TrimSpace(line); isTailscaleIP(ip) {
			return ip
		}
	}
	return ""
}

func tailscaleIPFromInterfaces() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip4 := ip.To4(); ip4 != nil && isTailscaleIP(ip4.String()) {
				return ip4.String()
			}
		}
	}
	return ""
}

// tailscaleCGNAT is the 100.64.0.0/10 range Tailscale assigns node IPs from.
var tailscaleCGNAT = func() *net.IPNet {
	_, n, _ := net.ParseCIDR("100.64.0.0/10")
	return n
}()

func isTailscaleIP(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && tailscaleCGNAT.Contains(ip)
}
