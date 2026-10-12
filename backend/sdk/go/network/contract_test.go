package network

import "testing"

func TestCarriageValues(t *testing.T) {
	cases := map[Carriage]string{
		CarriageRequestHeader: "request_header",
		CarriageConnectHeader: "connect_header",
		CarriageDisabled:      "disabled",
	}
	for c, want := range cases {
		if string(c) != want {
			t.Errorf("Carriage %q = %q, want %q", c, string(c), want)
		}
	}
}

func TestTunnelPolicyValues(t *testing.T) {
	cases := map[TunnelPolicy]string{
		TunnelDeny:                "deny",
		TunnelAuthorityBinding:    "authority_binding",
		TunnelPassthroughNonAgent: "passthrough_non_agent",
	}
	for p, want := range cases {
		if string(p) != want {
			t.Errorf("TunnelPolicy %q = %q, want %q", p, string(p), want)
		}
	}
}

func TestProofDefaults(t *testing.T) {
	if DefaultResourceProofHeader != "DPoP" {
		t.Errorf("DefaultResourceProofHeader = %q, want DPoP", DefaultResourceProofHeader)
	}
	if DefaultResourceProofScheme != "DPoP" {
		t.Errorf("DefaultResourceProofScheme = %q, want DPoP", DefaultResourceProofScheme)
	}
	if DefaultIntrospectionProofHeader != "DPoP-Introspection" {
		t.Errorf("DefaultIntrospectionProofHeader = %q, want DPoP-Introspection", DefaultIntrospectionProofHeader)
	}
	if DefaultIntrospectionProofScheme != "DPoP" {
		t.Errorf("DefaultIntrospectionProofScheme = %q, want DPoP", DefaultIntrospectionProofScheme)
	}
}

func TestValidCarriage(t *testing.T) {
	for _, c := range []Carriage{CarriageRequestHeader, CarriageConnectHeader, CarriageDisabled} {
		if !ValidCarriage(c) {
			t.Errorf("ValidCarriage(%q) = false, want true", c)
		}
	}
	for _, c := range []Carriage{"", "bogus", "REQUEST_HEADER"} {
		if ValidCarriage(c) {
			t.Errorf("ValidCarriage(%q) = true, want false", c)
		}
	}
}

func TestValidTunnelPolicy(t *testing.T) {
	for _, p := range []TunnelPolicy{TunnelDeny, TunnelAuthorityBinding, TunnelPassthroughNonAgent} {
		if !ValidTunnelPolicy(p) {
			t.Errorf("ValidTunnelPolicy(%q) = false, want true", p)
		}
	}
	for _, p := range []TunnelPolicy{"", "bogus", "DENY"} {
		if ValidTunnelPolicy(p) {
			t.Errorf("ValidTunnelPolicy(%q) = true, want false", p)
		}
	}
}
