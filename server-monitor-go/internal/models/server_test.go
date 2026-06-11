// Characterization test (Phase 0) pinning the JSON wire contract of the Server
// DTO. This shape is shared by the frozen agent ingest payload
// (POST /api/servers/update) and every REST/WebSocket response. Deployed agents
// cannot be updated atomically, so a renamed or dropped tag is a breaking
// change. If the Phase 1 move of this type changes the key set, this test must
// fail.
package models

import (
	"encoding/json"
	"testing"
)

func TestServerJSONContract(t *testing.T) {
	// The exact set of JSON keys deployed agents and the frontend depend on.
	want := []string{
		"id", "name", "type", "location", "ip_address", "hostname",
		"tailscale_ip", "status", "uptime", "network_in", "network_out",
		"cpu", "memory", "disk", "os_type", "cpu_info", "total_memory",
		"total_disk", "order_index", "first_seen", "last_update",
	}

	b, err := json.Marshal(Server{})
	if err != nil {
		t.Fatalf("marshal Server: %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal Server: %v", err)
	}

	if len(got) != len(want) {
		t.Errorf("Server has %d JSON keys, want %d: %v", len(got), len(want), keys(got))
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("Server JSON missing required key %q (wire contract break)", k)
		}
	}
	for k := range got {
		if !contains(want, k) {
			t.Errorf("Server JSON has unexpected key %q (verify it is intended)", k)
		}
	}
}

func TestClientJSONContract(t *testing.T) {
	b, _ := json.Marshal(Client{})
	var got map[string]json.RawMessage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal Client: %v", err)
	}
	for _, k := range []string{"name", "created_at"} {
		if _, ok := got[k]; !ok {
			t.Errorf("Client JSON missing required key %q", k)
		}
	}
}

func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
