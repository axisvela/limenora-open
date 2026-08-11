# 架构概览（中文）

LIMENORA 是自治系统的受治理南北向集成边界。

## 部署角色

- **Edge Gateway**：运行在 Agent 主机 / 设备 / 机器人上。本地出口、TLS MITM、短 TTL 决策缓存、离线/降级模式。
- **Enterprise Gateway**：集中式入站/出站数据平面。合作伙伴边界、企业 API 入口、Connector 执行。
- **Control Plane**：管理 Route、Connector、Protocol、Tenant 与部署。

## 六大平面

- **控制平面**：集成目录、Route 注册表、Connector 注册表、协议配置、部署/注册。
- **入站/出站数据平面**：Proxy、MCP 网关、Webhook/Event 网关、WebSocket/gRPC、协议归一化。
- **信任执行平面**：AEGIVELA 适配器、mTLS/工作负载身份、授权/撤销校验、租户隔离。
- **凭据与转换平面**：凭据绑定、令牌交换、header/query/body 注入、OAuth 刷新、finalizer。
- **可靠性平面**：限流、重试、熔断、背压、去重、幂等、降级策略。
- **证据平面**：交换日志、脱敏摘要、追踪、审计导出、合规留存。

## 核心对象

IntegrationEndpoint、Route、ProtocolBinding、ConnectorVersion、TenantBoundary、TrustBoundary、CredentialBinding、IntegrationExchange —— 全部由 LIMENORA 拥有。

## 产品边界

- **AEGIVELA** 拥有身份、授权、策略、撤销、证明。LIMENORA 是策略执行点（PEP），不是 PDP/IAM。
- **NOETIVELA** 拥有模型选择、成本、回退。LIMENORA 只传输模型流量。
- **MODUREGIS** 拥有能力注册表。LIMENORA 的 Route 只引用能力，从不复制定义。
- **Harmovela** 拥有协调语义。LIMENORA 只做消息映射与传输。

完整产品规划见 limenora-ee 仓库。
