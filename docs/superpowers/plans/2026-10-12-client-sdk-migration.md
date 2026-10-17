# LIMENORA Client SDK Migration Plan (license-clean client, no duplication)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Cross-repo (LIMENORA-open + PRAXOVELA). Do not push/tag without authorization.

**Goal:** Publish a license-clean (Apache-2.0) **LIMENORA client** in the `LIMENORA-open` SDK — the FP-1 federated-presentation contract plus a proxy-egress client — and have **PRAXOVELA consume it** instead of hand-rolling `network.ProxyEgress` and duplicating the carriage constants.

**Why:** PRAXOVELA already accesses EIDOVELA through the `EIDOVELA-open` SDK; the LIMENORA side should be symmetric (access through the `LIMENORA-open` SDK). Core's client logic is AGPL; consumers need an Apache-2.0 client surface. Today PRAXOVELA's `packages/adapters/network/proxy.go` + `eidovela.ProofCarriage`/`ResourceProofConfig` duplicate LIMENORA's `GATEWAY_*` carriage contract.

**Repos:** `LIMENORA-open` (`backend/sdk/go`, Apache-2.0) and `PRAXOVELA` (`packages/adapters`, consumes).
**Depends on:** `LIMENORA-open` FP-1 client contract (done: `backend/sdk/go/presentation.go`). `EIDOVELA-open` SDK stays the source of `SignDPoP`.

## Design

- New package `LIMENORA-open/backend/sdk/go/network` (Apache-2.0) containing the **client**:
  - Contract: `GatewayRequest{URL, Method, Headers, Body, Ctx}`, `GatewayResult{Status, Body, Headers}`, `NetworkGateway`, `AgentTokenSource`.
  - `ProxyEgress` (moved from PRAXOVELA `packages/adapters/network/proxy.go`): CONNECT + `Proxy-Authorization` (never destination `Authorization`), destination `Authorization` preserved, redirect hygiene (drop credentials cross-origin), `MaxIdleConnsPerHost: -1` (re-auth per request), bounded response (`WithMaxResponseBytes`), operator CA trust (`WithTLSConfig`); fail-closed on missing token.
  - Metadata deny (`denyMetadataDestination`) travels with the client.
  - The FP-1 carriage constants/types come from the SDK root package (`sdk.Carriage`, `sdk.TunnelPolicy`, `sdk.Default*`) — do not re-declare.
- `sdk.go` (root) re-exports the network surface for convenience.
- **PRAXOVELA adoption:** depend on `github.com/axisvela/limenora-open/sdk/go` (add `replace => ../../../LIMENORA-open`, mirroring the EIDOVELA-open replace); delete its bespoke `network.ProxyEgress`/contract and its duplicated `eidovela.ProofCarriage`/`ResourceProofConfig` (use `sdk.Carriage`/`sdk.TunnelPolicy`/`sdk.Default*`); keep `eidovela.SignDPoP`/`IntrospectionProof`.

## Tasks

### Task 1 (LIMENORA-open): client contract + `ProxyEgress`
- Create `backend/sdk/go/network/contract.go`, `network/proxy.go`, `network/metadata.go` (+ tests). Port from PRAXOVELA `packages/adapters/network` with the package renamed; keep behavior byte-for-byte (tests port with it).
- `go build ./...`, `go test ./sdk/go/network/... -count=1`.

### Task 2 (LIMENORA-open): re-export + docs/changelog
- `sdk.go`: re-export `NetworkGateway`, `GatewayRequest/Result`, `AgentTokenSource`, `NewProxyEgress`, options.
- `CHANGELOG.md` entry; `docs/api-reference.md` note the client SDK.

### Task 3 (PRAXOVELA): adopt the SDK
- Add the dependency + `replace`; replace `network.ProxyEgress` usages with `sdknetwork` (the SDK package); delete the moved files from `packages/adapters/network`; replace duplicated carriage constants with `sdk.*`.
- Keep `packages/adapters/eidovela` (SignDPoP via EIDOVELA-open SDK) and wire both proofs (resource + `DPoP-Introspection`) into the SDK `ProxyEgress`.
- `go build ./...`; `go test ./network/... ./... `; vendor-sync (adapters network/httpclient only).

### Task 4: end-to-end + release
- Real FP-1 acceptance: PRAXOVELA (SDK client, both proofs) → LIMENORA (verify resource proof + replay + introspect) → upstream; deny/no-downgrade/no-leak. Then close the FP-1 gate and version/CHANGELOG both repos.

## Open issues / dependencies
- **LIMENORA(-open) module path has no `/v2`**, so the SDK can't `require` core `v2.x` by path; the SDK uses a dev `replace`. Resolve the core module-path inconsistency (adopt `/v2` or keep major at `v1`) before publishing a real SDK release. (Tracked.)
- `agent_tunnel_policy` behavior is validated but not fully enforced in core (raw-tunnel handling is inline) — a core follow-up, tracked in the FP-1 plan.
- PRAXOVELA `packages/adapters/network` currently also holds `secretref.go` (typed `secret://` validation) and `confinement.go` — decide whether those move too (they are client-side; keep in PRAXOVELA unless needed by the SDK client).

## Self-Review
Covers the license-clean client move (T1/T2) and PRAXOVELA adoption (T3) with the shared FP-1 contract (already published). End-to-end + release is T4. Open issues are explicit, not hidden. Rust/Python/TS SDKs remain placeholders (out of scope).
