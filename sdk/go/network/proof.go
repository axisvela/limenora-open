package network

// ProofCarrier supplies the FP-1 federated-presentation proof headers for one
// outbound request. Implementations must fail closed: a proof that cannot be
// produced returns an error so the request is not sent unauthenticated.
//
// The carriage (where the proof is attached) and the proof header/scheme names
// are the carrier's own operator-selected configuration and MUST match the
// gateway's GATEWAY_* profile, or the bound target/header will not verify.
type ProofCarrier interface {
	// ProofHeaders returns the header name→value pairs to attach to the request
	// bound to (method, target). target is the full request URI for
	// request_header carriage, or the scheme://host:port authority for
	// connect_header carriage. The returned headers must not include
	// Proxy-Authorization (that is the client credential, set separately).
	ProofHeaders(method, target, token string) (map[string]string, error)
}
