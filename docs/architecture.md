# LIMENORA Architecture

LIMENORA is the governed north–south integration boundary for autonomous systems.

## Deployment roles

- **Edge Gateway**: runs on the agent host / device / robot. Local egress, TLS MITM, short-TTL decision cache, offline/degraded mode.
- **Enterprise Gateway**: central ingress/egress data plane. Partner boundary, enterprise API entry, connector execution.
- **Control Plane**: manages routes, connectors, protocols, tenants, and deployment.

## Planes

- **Control Plane**: integration catalog, route registry, connector registry, protocol profiles, deployment/enrollment.
- **Ingress/Egress Data Plane**: proxy, MCP gateway, webhook/event gateway, WebSocket/gRPC, protocol normalization.
- **Trust Enforcement Plane**: AEGIVELA adapter, mTLS/workload identity, grant/revocation validation, tenant isolation.
- **Credential & Transform Plane**: credential binding, token exchange, header/query/body injection, OAuth refresh, finalizers.
- **Reliability Plane**: rate limit, retry, circuit breaker, backpressure, dedupe, idempotency, degraded-mode policy.
- **Evidence Plane**: exchange journal, redacted request/response digests, traces, audit export, compliance retention.

## Core objects

IntegrationEndpoint, Route, ProtocolBinding, ConnectorVersion, TenantBoundary,
TrustBoundary, CredentialBinding, IntegrationExchange — all owned by LIMENORA.

## Product boundaries

- **AEGIVELA** owns identity, grant, policy, revocation, attestation. LIMENORA is a Policy Enforcement Point, not a PDP/IAM.
- **NOETIVELA** owns model selection, cost, fallback. LIMENORA transports model traffic only.
- **MODUREGIS** owns capability registry. LIMENORA routes reference capabilities, never copies definitions.
- **Harmovela** owns coordination semantics. LIMENORA maps and transports messages.

For the full internal product plan, see the LIMENORA-ee repository.
