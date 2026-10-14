// Package sdk is the official Apache-2.0 Go client SDK for LIMENORA.
//
// It is self-contained: it holds the FP-1 federated-presentation contract and
// the core-free proxy-egress client, and imports no AGPL core package, so a
// consumer can integrate with LIMENORA without pulling the core into its module
// graph. The client lives in the network subpackage; the root package
// re-exports it for convenience.
package sdk

import "github.com/axisrobo/LIMENORA-open/sdk/go/network"

// Version is the SDK version.
const Version = "2.4.0"

// FP-1 presentation carriage contract.
type (
	Carriage     = network.Carriage
	TunnelPolicy = network.TunnelPolicy
)

const (
	CarriageRequestHeader = network.CarriageRequestHeader
	CarriageConnectHeader = network.CarriageConnectHeader
	CarriageDisabled      = network.CarriageDisabled

	TunnelDeny                = network.TunnelDeny
	TunnelAuthorityBinding    = network.TunnelAuthorityBinding
	TunnelPassthroughNonAgent = network.TunnelPassthroughNonAgent

	DefaultResourceProofHeader      = network.DefaultResourceProofHeader
	DefaultResourceProofScheme      = network.DefaultResourceProofScheme
	DefaultIntrospectionProofHeader = network.DefaultIntrospectionProofHeader
	DefaultIntrospectionProofScheme = network.DefaultIntrospectionProofScheme
)

// Client network boundary.
type (
	AuthzContext      = network.AuthzContext
	GatewayRequest    = network.GatewayRequest
	GatewayResult     = network.GatewayResult
	NetworkGateway    = network.NetworkGateway
	SecretInjector    = network.SecretInjector
	AgentTokenSource  = network.AgentTokenSource
	StaticTokenSource = network.StaticTokenSource
	ProxyEgress       = network.ProxyEgress
	ProxyOption       = network.ProxyOption
	ProofCarrier      = network.ProofCarrier
)

var (
	// NewProxyEgress builds the proxy-aware LIMENORA client egress.
	NewProxyEgress       = network.NewProxyEgress
	WithTLSConfig        = network.WithTLSConfig
	WithMaxResponseBytes = network.WithMaxResponseBytes
	WithProofCarrier     = network.WithProofCarrier
	WithCarriage         = network.WithCarriage
	NewStaticTokenSource = network.NewStaticTokenSource
	ErrNoAgentToken      = network.ErrNoAgentToken
)
