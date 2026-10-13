# Changelog

## 2.4.0

- Synchronize the SDK version surfaces with the core tag `v2.4.0` (Go
  `backend/sdk/go`, Python `limenora_sdk`, Rust `backend/sdk/rust`, TypeScript
  `backend/sdk/typescript`).
- Core `v2.4.0` adds the AEGIVELA Decision Service (PEP) integration
  (`GATEWAY_AEGIVELA_*`, `pkg/limenora/pep`); no SDK wire-surface change.
- Note: federated-presentation (FP-1) remains gateway-side in core; its public
  schemas graduate here when the frozen cross-repo contract is published.

## 2.3.0

- Mirror the core `contracts/` into `contracts/`: agent binding
  (`agent/agent-binding.schema.json`), route resolution
  (`api/route-resolution-{request,response}.schema.json`), the additive
  `api/policy-connect-request.schema.json` fields (agent identity,
  project/tenant/organization scope, audience), plus the existing policy,
  inject, approval and transform contracts.
- Go SDK (`backend/sdk/go`): fix the core import path to
  `github.com/axisrobo/LIMENORA/pkg/limenora` and pin the core dependency to
  `v2.3.0`.
- Note: federated-presentation (FP-1) is gateway-side in core; its public
  schemas graduate here when the frozen cross-repo contract is published.

## 0.0.1 (Unreleased)

- Initial repository setup with SDK placeholders, examples, and public documentation.

