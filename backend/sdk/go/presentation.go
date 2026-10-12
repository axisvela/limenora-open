package sdk

// Federated presentation client contract (EIDOVELA-open FP-1), shared by the
// LIMENORA gateway and its clients so a client binds the same target the
// gateway verifies. These values mirror the gateway's `GATEWAY_*` configuration
// and MUST NOT be caller-supplied.

// Carriage selects where a resource proof is carried and which target its
// `htu` binds.
type Carriage string

const (
	// CarriageRequestHeader binds the intercepted inner request
	// scheme://host/path and carries the proof on a dedicated header (default).
	// Requires MITM interception of the route.
	CarriageRequestHeader Carriage = "request_header"
	// CarriageConnectHeader binds the CONNECT authority scheme://host:port only
	// (explicit opt-in; the path is not covered).
	CarriageConnectHeader Carriage = "connect_header"
	// CarriageDisabled carries no resource proof; the route must not require one.
	CarriageDisabled Carriage = "disabled"
)

// TunnelPolicy decides how an un-intercepted raw tunnel carrying selected agent
// traffic is treated.
type TunnelPolicy string

const (
	// TunnelDeny rejects the agent request; never NotApplicable, never downgrade.
	TunnelDeny TunnelPolicy = "deny"
	// TunnelAuthorityBinding accepts a connect_header proof bound to host:port;
	// valid only together with CarriageConnectHeader.
	TunnelAuthorityBinding TunnelPolicy = "authority_binding"
	// TunnelPassthroughNonAgent continues existing application authentication for
	// a route not selected for agent processing.
	TunnelPassthroughNonAgent TunnelPolicy = "passthrough_non_agent"
)

// Proof carriage header/scheme defaults (FP-1). RFC 9449 defines neither a
// CONNECT proof nor an introspection proof header, so distinct versioned names
// are used. Resource and introspection proof header/scheme are operator
// configurable.
const (
	DefaultResourceProofHeader      = "DPoP"
	DefaultResourceProofScheme      = "DPoP"
	DefaultIntrospectionProofHeader = "DPoP-Introspection"
	DefaultIntrospectionProofScheme = "DPoP"
)

// ValidCarriage reports whether c is a known resource-proof carriage.
func ValidCarriage(c Carriage) bool {
	switch c {
	case CarriageRequestHeader, CarriageConnectHeader, CarriageDisabled:
		return true
	default:
		return false
	}
}

// ValidTunnelPolicy reports whether p is a known raw-tunnel policy.
func ValidTunnelPolicy(p TunnelPolicy) bool {
	switch p {
	case TunnelDeny, TunnelAuthorityBinding, TunnelPassthroughNonAgent:
		return true
	default:
		return false
	}
}
