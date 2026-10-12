# Agent Binding Contract

The agent binding maps a LIMENORA-validated **agent service credential** to an
external agent identity and form. Identity authority is NOT LIMENORA:

- `agent_ref`, `agent_class`, and epochs are owned by **NOMIVELA** (ADR 0005).
- Workload authentication (PoP token) is owned by **EIDOVELA**.
- Authorization/delegation is owned by **AEGIVELA**.

LIMENORA authenticates/authorizes the **outbound service call** only.

## Modes

- `identity_mode: native` — LIMENORA holds an agent record + token bound to
  `agent_ref` and validates its own token (`authority: "limenora"`).
- `identity_mode: federated` — LIMENORA verifies an external assertion
  (EIDOVELA PoP token); `authority` is `eidovela` or `nomivela`.

## agent_class and identity_type

`agent_class` is the NOMIVELA form (ADR 0005); `identity_type` is the Agent Identity
type (`interactive | autonomous | digitalEmployee`). **Together** they select the
integration profile (gateway, policy scope, credential context, approval mode) — see the
spec's per-`(agentClass, identityType)` matrix (§5) and AEGIVELA §3.1. `identity_type`
**refines** the `agent_class` profile and never widens it. Neither changes identity,
binding, or epoch semantics.

## Schemas

- `agent-binding.schema.json` — the binding record.
- `../api/route-resolution-request.schema.json` / `../api/route-resolution-response.schema.json` —
  the pre-flight API that returns the decision + effective class profile.
