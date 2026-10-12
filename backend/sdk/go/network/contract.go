// Package network defines the license-clean (Apache-2.0) LIMENORA client
// network boundary and the FP-1 federated-presentation carriage contract.
package network

import "context"

// AuthzContext carries authorization info through the gateway.
type AuthzContext struct {
	TenantID    string
	WorkspaceID string
	ActorID     string
	TraceID     string
}

// GatewayRequest is a proxied outbound request.
type GatewayRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
	Ctx     AuthzContext
}

// GatewayResult is the response from a gateway.
type GatewayResult struct {
	Status  int
	Body    []byte
	Headers map[string]string
}

// NetworkGateway proxies outbound calls.
type NetworkGateway interface {
	Do(ctx context.Context, req GatewayRequest) (*GatewayResult, error)
}

// SecretInjector injects secrets into outbound requests.
type SecretInjector interface {
	Inject(ctx context.Context, req *GatewayRequest) error
}
