package network

import (
	"context"
	"errors"
	"testing"
)

func TestStaticTokenSource_Token(t *testing.T) {
	src := NewStaticTokenSource("agent-token-123")
	tok, err := src.Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "agent-token-123" {
		t.Errorf("token = %q, want agent-token-123", tok)
	}
}

func TestStaticTokenSource_MissingFailsClosed(t *testing.T) {
	src := NewStaticTokenSource("")
	_, err := src.Token(context.Background())
	if !errors.Is(err, ErrNoAgentToken) {
		t.Fatalf("err = %v, want ErrNoAgentToken", err)
	}
}
