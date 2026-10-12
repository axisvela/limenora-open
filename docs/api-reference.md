# API Reference

LIMENORA exposes the same HTTP control surface from both the Go runtime and the
Rust reference implementation. Machine-readable contracts live in `contracts/`
(JSON Schema) and are the single source of truth. Current version: `v2.3.0`.

## Health

| Route | Method | Description |
| --- | --- | --- |
| `/healthz` | GET | Bare `200 OK` (liveness). |
| `/v1/health` | GET | `{"status":"ok","version":"v2.3.0"}` (readiness + version). |

## Identity

| Route | Method | Description |
| --- | --- | --- |
| `/me` | GET | Authenticated user ID (requires admin auth). |

## Vault provider API

All vault routes require admin auth (API key `Authorization: Bearer oc_...` or
session). `/v1/vault/...` and legacy `/api/vault/...` are equivalent.

| Route | Method | Description |
| --- | --- | --- |
| `/v1/vault/{provider}/pair` | POST | Pair a vault provider (Bitwarden). |
| `/v1/vault/{provider}/pair` | DELETE | Disconnect a provider. |
| `/v1/vault/{provider}/status` | GET | Pairing/connection status. |

## Cache

| Route | Method | Description |
| --- | --- | --- |
| `/v1/cache/invalidate` | POST | Invalidate cached CONNECT/app-injection decisions. Legacy alias `/api/cache/invalidate`. |

## Approvals

| Route | Method | Description |
| --- | --- | --- |
| `/v1/approvals/pending` | GET | List pending manual approvals (long-poll up to 30s with `?exclude=`). |
| `/v1/approvals/{id}/decision` | POST | Submit an approve/deny decision. |

## Proxy / CONNECT

- `CONNECT host:port` — tunnel or MITM depending on policy (SSRF-protected;
  invalid agent token → `407`; governed fail-closed → `407`/`502`).
- `GET http(s)://host/path` (absolute-form) — HTTP proxy; response headers are
  scrubbed on both the proxy and MITM paths.

## Agent bindings and credentials

Management endpoints require an authentication service **and** an explicit
`AdminRequiredRole`; responses carry `Cache-Control: no-store`. Credentials are
stored only as SHA-256 hashes; with `GATEWAY_AGENT_BINDING_TTL` expiry,
rotation and revocation.

| Route | Method | Description |
| --- | --- | --- |
| `/v1/agent/bindings` | POST | Mint a native binding credential (token returned once). |
| `/v1/agent/bindings/rotate` | POST | Rotate: mint a replacement, record `rotated_from`. |
| `/v1/agent/bindings/revoke` | POST | Revoke a credential (idempotent). |
| `/v1/agent/binding/resolve` | POST | Resolve the effective `(agent_class, identity_type)` profile. |

## Federated presentation (FP-1, gateway-side)

Configurable via `GATEWAY_*`: `GATEWAY_EIDOVELA_URL`/`_AUDIENCE`,
`GATEWAY_RESOURCE_PROOF_CARRIAGE` (`request_header` default), `GATEWAY_AGENT_TUNNEL_POLICY`
(`deny` default), `GATEWAY_RESOURCE_PROOF_HEADER`/`_SCHEME` (default `DPoP`),
`GATEWAY_INTROSPECTION_PROOF_HEADER` (default `DPoP-Introspection`). The gateway
verifies the RFC 9449 resource proof (`htm`/`htu`/`ath`/freshness/replay) and
relays the client introspection proof to EIDOVELA `includeContext`. The FP-1
release gate remains open pending real cross-component acceptance.

## Error semantics

| Code | Meaning |
| --- | --- |
| `400` | Malformed request / unknown agent credential. |
| `401` | Missing/invalid admin auth; unusable agent credential. |
| `403` | SSRF block; protocol-profile block; residency denied; wrong role. |
| `407` | Missing/invalid agent token (governed or non-governed). |
| `502` | Governed fail-closed with no usable decision; required credential unavailable. |
| `503` | Credential-management authority not configured. |

## Contracts

`contracts/api/policy-connect-{request,response}.schema.json`,
`contracts/api/route-resolution-{request,response}.schema.json`,
`contracts/agent/agent-binding.schema.json`, `contracts/policy/rule.schema.json`,
`contracts/inject/rule.schema.json`, `contracts/approval/*.schema.json`.

## SDK

`backend/sdk/go` (Apache-2.0) re-exports the public core `pkg/limenora` surface
and publishes the FP-1 federated-presentation client contract (`Carriage`,
`TunnelPolicy`, proof header/scheme defaults). A license-clean client
(proxy-egress) move is planned; see `docs/superpowers/plans/`.
