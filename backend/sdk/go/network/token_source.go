package network

import (
	"context"
	"errors"
)

// ErrNoAgentToken marks a missing credential (callers must fail closed).
var ErrNoAgentToken = errors.New("network: no agent token configured")

// AgentTokenSource yields the credential presented to the gateway.
type AgentTokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticTokenSource returns a pre-provisioned token.
type StaticTokenSource struct{ token string }

// NewStaticTokenSource returns a source for a fixed token.
func NewStaticTokenSource(token string) *StaticTokenSource { return &StaticTokenSource{token: token} }

// Token returns the configured token or ErrNoAgentToken when unset.
func (s *StaticTokenSource) Token(context.Context) (string, error) {
	if s.token == "" {
		return "", ErrNoAgentToken
	}
	return s.token, nil
}
