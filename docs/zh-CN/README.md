# LIMENORA — 企业级 Agent 集成网关

**治理自治系统与外部企业之间边界的开源产品。**

LIMENORA 是一个受治理的南北向（north–south）集成边界，用于把 Agent、工作流、机器人、数字孪生连接到外部 API、MCP Server、事件、Webhook 与合作伙伴生态，同时不把协议、凭据和信任假设泄漏进自治运行时。

## 三个仓库

LIMENORA 由一个核心运行时和两个配套仓库组成：

| 仓库 | License | 用途 |
|---|---|---|
| [limenora](https://github.com/axisvela/limenora) | AGPL-3.0 | **核心运行时**：Edge Gateway / Enterprise Gateway 数据平面、TLS MITM、策略执行点、凭据注入、证据。 |
| [limenora-open](https://github.com/axisvela/limenora-open)（本仓库） | Apache-2.0 | **对外入口**：SDK、示例、API 规范、二进制发布说明、文档。 |
| [limenora-ee](https://github.com/axisvela/limenora-ee) | 企业 License | **企业版**：HA 集群、联邦、高级凭据代理、证据留存、合规包、Admin Console。 |

## 部署角色

- **LIMENORA Edge Gateway** — 部署在 Agent 主机 / 设备 / 机器人上，本地出口、TLS MITM、短 TTL 决策缓存、离线/降级模式。
- **LIMENORA Enterprise Gateway** — 集中式入站/出站数据平面，负责合作伙伴边界、企业 API 入口、Connector 执行。
- **LIMENORA Control Plane** — 管理 Route、Connector、Protocol、Tenant 与部署（规划中）。

## 核心能力

- 统一的 API / MCP / Event / Webhook / WebSocket 外部边界
- 受治理的交换：Identity + Capability + Grant + Tenant + Credential + Route
- 可解释、可撤销、可证明的证据与联邦能力
- Secretless Agent：凭据只在边界瞬时物化

## 快速开始

- [中文快速开始](zh-CN/quickstart.md) | [English quickstart](../quickstart.md)
- [架构概览](zh-CN/architecture.md) | [Architecture](../architecture.md)
- [路线图](zh-CN/roadmap.md) | [Roadmap](../roadmap.md)

## SDK 与示例

- [Go SDK](../backend/sdk/go)
- [示例](examples.md)（Edge / Enterprise / Connector / MCP / Webhook）

## 贡献

请阅读 [贡献指南](zh-CN/contributing.md) | [Contributing](../contributing.md)。

## License

本仓库为 Apache-2.0。核心运行时见 [limenora](https://github.com/axisvela/limenora) 的 AGPL-3.0。
