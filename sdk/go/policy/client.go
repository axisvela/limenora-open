// Package policy is the Apache-2.0 LIMENORA policy-plane client: the CONNECT
// policy decision and the content revision, from the shared public contract
// (contracts/api/policy-connect-*.schema.json). It holds no AGPL core code.
package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ConnectPath is the policy-connect endpoint.
const ConnectPath = "/v1/policy/connect"

// RevisionPath is the policy content-revision endpoint.
const RevisionPath = "/v1/policy/revision"

// ConnectRequest is a CONNECT policy decision request.
type ConnectRequest struct {
	AgentToken     string            `json:"agent_token"`
	Host           string            `json:"host"`
	Port           string            `json:"port,omitempty"`
	Scheme         string            `json:"scheme,omitempty"`
	Method         string            `json:"method,omitempty"`
	Path           string            `json:"path,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	ConnectionID   string            `json:"connection_id,omitempty"`
	GatewayRole    string            `json:"gateway_role,omitempty"`
	GatewayVersion string            `json:"gateway_version,omitempty"`

	AgentRef       string `json:"agent_ref,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentClass     string `json:"agent_class,omitempty"`
	IdentityType   string `json:"identity_type,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	Audience       string `json:"audience,omitempty"`
}

// ConnectResponse is the policy decision plus the source content revision.
type ConnectResponse struct {
	Decision ConnectDecision `json:"decision"`
	Revision string          `json:"revision,omitempty"`
}

// Action is a policy rule action.
type Action struct {
	Type        string `json:"type"`
	RuleID      string `json:"rule_id,omitempty"`
	MaxRequests int    `json:"max_requests,omitempty"`
	WindowSecs  int    `json:"window_secs,omitempty"`
}

// PolicyRule is a matched policy rule.
type PolicyRule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	HostPattern string         `json:"host_pattern"`
	PathPattern string         `json:"path_pattern"`
	Method      string         `json:"method,omitempty"`
	Action      Action         `json:"action"`
	Conditions  map[string]any `json:"conditions,omitempty"`
}

// InjectionRule is a credential/secret injection rule.
type InjectionRule struct {
	ID                   string            `json:"id,omitempty"`
	Source               string            `json:"source"`
	Location             string            `json:"location"`
	Name                 string            `json:"name"`
	Value                string            `json:"value,omitempty"`
	HostPattern          string            `json:"host_pattern,omitempty"`
	PathPattern          string            `json:"path_pattern,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	RefreshToken         string            `json:"refresh_token,omitempty"`
	ExpiresAt            string            `json:"expires_at,omitempty"`
	OAuthProvider        string            `json:"oauth_provider,omitempty"`
	CredentialBindingRef string            `json:"credential_binding_ref,omitempty"`
}

// CacheScope scopes a cached decision for invalidation.
type CacheScope struct {
	TenantID       string   `json:"tenant_id,omitempty"`
	ProjectID      string   `json:"project_id,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
	AgentID        string   `json:"agent_id,omitempty"`
	Hosts          []string `json:"hosts,omitempty"`
	ProviderIDs    []string `json:"provider_ids,omitempty"`
}

// ConnectDecision is a resolved CONNECT policy decision.
type ConnectDecision struct {
	Intercept        bool            `json:"intercept"`
	InterceptReason  string          `json:"intercept_reason,omitempty"`
	TenantID         string          `json:"tenant_id,omitempty"`
	ProjectID        string          `json:"project_id,omitempty"`
	OrganizationID   string          `json:"organization_id,omitempty"`
	AgentID          string          `json:"agent_id,omitempty"`
	AgentName        string          `json:"agent_name,omitempty"`
	AgentIdentifier  string          `json:"agent_identifier,omitempty"`
	AgentRef         string          `json:"agent_ref,omitempty"`
	AgentClass       string          `json:"agent_class,omitempty"`
	IdentityType     string          `json:"identity_type,omitempty"`
	AccessRestricted bool            `json:"access_restricted,omitempty"`
	Mode             string          `json:"policy_mode,omitempty"`
	Rules            []PolicyRule    `json:"policy_rules,omitempty"`
	InjectionRules   []InjectionRule `json:"injection_rules,omitempty"`
	ProviderIDs      []string        `json:"provider_ids,omitempty"`
	AppConnectionIDs []string        `json:"app_connection_ids,omitempty"`
	BodyTransform    string          `json:"body_transform,omitempty"`
	CacheScope       CacheScope      `json:"cache_scope,omitempty"`
	CacheRevision    string          `json:"cache_revision,omitempty"`
}

// Client is the LIMENORA policy-plane client.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a policy client. baseURL is required.
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}, nil
}

// ResolveConnect requests a CONNECT policy decision.
func (c *Client) ResolveConnect(ctx context.Context, req ConnectRequest) (ConnectResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return ConnectResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ConnectPath, bytes.NewReader(data))
	if err != nil {
		return ConnectResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ConnectResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ConnectResponse{}, decodeError(resp, "policy connect failed")
	}
	var out ConnectResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ConnectResponse{}, err
	}
	return out, nil
}

// Revision fetches the content revision for the agent token.
func (c *Client) Revision(ctx context.Context, agentToken string) (string, error) {
	endpoint := c.baseURL + RevisionPath + "?agent_token=" + url.QueryEscape(agentToken)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", decodeError(resp, "policy revision failed")
	}
	var out struct {
		Revision string `json:"revision"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Revision, nil
}

func decodeError(resp *http.Response, prefix string) error {
	var errResp struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error == "" {
		errResp.Error = resp.Status
	}
	return fmt.Errorf("%s: %s", prefix, errResp.Error)
}
