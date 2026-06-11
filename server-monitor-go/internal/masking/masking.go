// Package masking centralizes the IP-masking rule in exactly one place so the
// policy is consistent across REST responses and WebSocket pushes, and so it
// can be evolved in a single edit.
//
// Current policy (as-built): anonymous viewers see a masked public IP address;
// hostname and Tailscale IP are not masked. Admin (valid JWT) viewers see real
// values.
package masking

import "github.com/thomasstefen/server-monitor/internal/domain"

// MaskedIP is the placeholder shown to unauthenticated viewers.
const MaskedIP = "***.***.***.**"

// One masks the sensitive fields of a single server in place.
func One(sv *domain.Server) {
	sv.IPAddress = MaskedIP
}

// All masks every server in the slice in place and returns it for convenience.
func All(servers []domain.Server) []domain.Server {
	for i := range servers {
		One(&servers[i])
	}
	return servers
}
