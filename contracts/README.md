# LIMENORA Contracts

Machine-readable contracts shared across the LIMENORA runtime. These are the
**single source of truth** for the wire format between the Go and Rust
gateways, and for cross-process HTTP APIs (policy decisions, approvals,
webhooks).

## Layout

```text
contracts/
  api/         HTTP API request/response schemas
  agent/       agent service binding schemas (NOMIVELA agent_ref/agent_class)
  policy/      policy rule and decision schemas
  inject/      secret/credential injection rule schemas
  approval/    manual approval object schemas
  schema/      (reserved) draft/versioned schema staging
  aegivela/    (reserved) AEGIVELA DecisionService wire schemas
  harmovela/   (reserved) Harmovela envelope schemas
  moduregis/   (reserved) registry/connector schemas
```

## Conventions

- Schemas follow JSON Schema draft 2020-12.
- Each schema has a stable `$id` under `https://limenora.dev/schemas/...`.
- Both the Go (`backend/go`) and Rust (`backend/rust`) implementations derive
  their wire types from these contracts. When a contract changes, update this
  directory **first**, then both implementations.
- `additionalProperties: false` on request/response objects so a contract
  drift is caught early.

## Contract change flow

1. Edit the schema in this directory.
2. Add or update a conformance fixture under `tests/` (see
   `backend/go/tests/` and `backend/rust/tests/`).
3. Update the Go wire types in `backend/go/internal/<pkg>`.
4. Update the Rust wire types in `backend/rust/src/<module>`.
5. Run both test suites.
