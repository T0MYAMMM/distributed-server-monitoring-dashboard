// Package masking centralizes the IP-masking rule in exactly one place so the
// policy is consistent across REST responses and WebSocket pushes, and so it
// can be evolved in a single edit.
//
// Policy: anonymous viewers never see a node's real network addresses — both
// the public IP and the Tailscale IP are masked. The OS hostname is not masked
// (it only identifies a node within the tailnet). Admin (valid JWT) viewers see
// real values, in REST responses and in live WebSocket frames alike.
package masking

import "github.com/thomasstefen/server-monitor/internal/domain"

// MaskedIP is the placeholder shown to unauthenticated viewers.
const MaskedIP = "***.***.***.**"

// One masks the sensitive address fields of a single server in place.
func One(sv *domain.Server) {
	sv.IPAddress = MaskedIP
	sv.TailscaleIP = MaskedIP
}

// All masks every server in the slice in place and returns it for convenience.
func All(servers []domain.Server) []domain.Server {
	for i := range servers {
		One(&servers[i])
	}
	return servers
}
