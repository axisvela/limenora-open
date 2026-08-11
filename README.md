# LIMENORA — Enterprise Agent Integration Gateway

**The governed north–south integration boundary between autonomous systems and the external enterprise.**

LIMENORA connects agents, workflows, robots and twins to external APIs, MCP
servers, events, webhooks and partner ecosystems — without leaking protocols,
credentials or trust assumptions into autonomous runtimes.

> [中文文档](./docs/zh-CN/README.md) · Chinese documentation

## The LIMENORA Product

LIMENORA is delivered as one open-core runtime plus two companion repositories:

| Repository | License | What it is |
|---|---|---|
| [**limenora**](https://github.com/axisrobo/limenora) | AGPL-3.0 | **Core runtime** — Edge / Enterprise Gateway data planes, TLS MITM, policy enforcement, credential injection, evidence. |
| [**limenora-open**](https://github.com/axisrobo/limenora-open) (this repo) | Apache-2.0 | **Public entry point** — SDKs, examples, API specifications, binary release info, documentation. |
| [**limenora-ee**](https://github.com/axisrobo/limenora-ee) | Enterprise License | **Enterprise Edition** — HA clustering, federation, advanced credential brokerage, evidence retention, compliance packs, Admin Console. |

## Deployment Roles

- **LIMENORA Edge Gateway** — runs on the agent host / device / robot. Local
  egress, TLS MITM, short-TTL decision cache, offline/degraded mode.
- **LIMENORA Enterprise Gateway** — centralized ingress/egress data plane.
  Partner boundary, enterprise API entry, connector execution.
- **LIMENORA Control Plane** — manages routes, connectors, protocols, tenants,
  and deployment (planned).

## Capabilities

- Unified external boundary for API / MCP / Event / Webhook / WebSocket
- Governed exchange: Identity + Capability + Grant + Tenant + Credential + Route
- Explainable, revocable, provable evidence and federation
- Secretless agents: credentials materialized only at the boundary

## Quick Start

- [English quickstart](./docs/quickstart.md)
- [中文快速开始](./docs/zh-CN/quickstart.md)
- [Architecture overview](./docs/architecture.md) · [架构概览](./docs/zh-CN/architecture.md)
- [Roadmap](./docs/roadmap.md) · [路线图](./docs/zh-CN/roadmap.md)

## SDKs

- [Go SDK](./backend/sdk/go)
- Python SDK (coming soon)
- TypeScript SDK (coming soon)

## Examples

- [Edge Gateway](./examples/edge-gateway)
- [Enterprise Gateway](./examples/enterprise-gateway)
- [Go Connector](./examples/connector-go)
- [MCP Client](./examples/mcp-client)
- [Webhook Receiver](./examples/webhook-receiver)

## Contributing

- [English contributing](./docs/contributing.md)
- [贡献指南](./docs/zh-CN/contributing.md)

## License

This repository is **Apache-2.0**. The core runtime is AGPL-3.0 in
[limenora](https://github.com/axisrobo/limenora). See [LICENSE](./LICENSE).
