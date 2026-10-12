package network

import "testing"

func TestCloudMetadataDenied_RejectsMetadataEndpoints(t *testing.T) {
	denied := []string{
		"169.254.169.254",          // AWS / GCP / Azure IMDS
		"169.254.169.254:80",       // with port
		"metadata.google.internal", // GCP metadata hostname (→ 169.254.169.254)
		"metadata.azure.internal",  // Azure metadata hostname
		"169.254.170.5",            // AWS ECS metadata (169.254.170.0/23 range)
		"168.63.129.16",            // Azure WireServer (metadata proxy)
		"fe80::a9fe:a9fe",          // AWS IMDS (IPv6 link-local)
		"fd00:ec2::254",            // AWS ECS (IPv6)
	}
	for _, h := range denied {
		if !cloudMetadataDenied(h) {
			t.Errorf("cloudMetadataDenied(%q) = false, want true", h)
		}
	}
}

func TestCloudMetadataDenied_AWSEcsRangeBoundary(t *testing.T) {
	if !cloudMetadataDenied("169.254.170.5") {
		t.Error("cloudMetadataDenied(169.254.170.5) = false, want true (inside 169.254.170.0/23)")
	}
	if cloudMetadataDenied("169.254.172.1") {
		t.Error("cloudMetadataDenied(169.254.172.1) = true, want false (outside 169.254.170.0/23)")
	}
}

func TestCloudMetadataDenied_AllowsNormalDestinations(t *testing.T) {
	allowed := []string{
		"example.com",
		"api.example.com:443",
		"8.8.8.8",
		"127.0.0.1",
		"localhost",
		"metadata.example.internal", // lookalike but not a provider endpoint
		"169.254.172.1",             // AWS-adjacent but outside the ECS /23
		"[::1]:8080",
	}
	for _, h := range allowed {
		if cloudMetadataDenied(h) {
			t.Errorf("cloudMetadataDenied(%q) = true, want false", h)
		}
	}
}

// TestMetadataDenyScopeCurrentBehavior documents the SSRF posture of the client
// metadata gate: it covers ONLY cloud metadata endpoints. RFC1918/loopback
// private targets are NOT denied by this layer (the client has no private-range
// deny). This test locks the current behavior so the intent is explicit.
func TestMetadataDenyScopeCurrentBehavior(t *testing.T) {
	allowed := []string{
		"10.0.0.1",
		"10.255.255.255",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.1.1",
		"192.168.0.100",
		"127.0.0.1",
		"localhost",
		"[::1]",
	}
	for _, h := range allowed {
		if cloudMetadataDenied(h) {
			t.Errorf("cloudMetadataDenied(%q) = true; %q is a private target, not cloud metadata", h, h)
		}
	}
}
