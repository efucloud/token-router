# Token Router 总体规格

## 1. 产品定位

Token Router 是面向企业内部的统一 AI 模型聚合与分发网关。它将多个模型供应商和
OpenAI 兼容上游收敛为稳定的数据面 API，并提供 OIDC 控制台、模型治理、用户/API
Token 双层用量限额、路由容错与审计能力。

架构吸收两个参考项目中适合企业场景的部分：

- new-api：简洁的渠道接入、OpenAI 兼容入口、优先级/权重路由和故障切换。
- TokenHub：控制面/数据面/运维面分层、Provider 与 Resource 分离、显式 Model
  Route、准入/预留/结算和 Route Attempt 审计。

本项目不是售卖 API 的商业计费平台，不包含价格、余额、充值、支付、订阅、兑换码、
邀请返利或公开注册。

## 2. 规格文档

| 子规格 | 说明 |
| --- | --- |
| [产品范围](specs/01-product-scope.md) | 目标用户、角色、范围与非目标 |
| [领域模型](specs/02-domain-model.md) | 模型、供应商资源、路由、Token 与日志实体 |
| [认证与安全](specs/03-auth-security.md) | OIDC、API Key、密钥存储、权限与安全边界 |
| [控制面 API](specs/04-control-plane-api.md) | `/api/v1` 管理接口和权限矩阵 |
| [数据面与路由](specs/05-data-plane-routing.md) | `/v1` 兼容协议、选路、重试、流式转发和冷却 |
| [限额与用量](specs/06-quota-usage.md) | 用户/Token 双层限额、预留结算和审计日志 |
| [管理控制台](specs/07-console.md) | Ant Design Pro 6 页面、菜单和交互要求 |
| [交付计划](specs/08-delivery-plan.md) | MVP 阶段、验收标准与后续范围 |
| [双维度 Dashboard](specs/09-dashboard.md) | 用户工作台、管理员运行看板与指标契约 |
| [企业能力路线](specs/10-enterprise-capabilities.md) | 参考 new-api/TokenHub 的能力取舍、成熟度和实施优先级 |

子规格是具体实现与验收依据；本文件只维护稳定的全局定位、原则与导航。

## 3. 全局架构原则

- 控制面：`/api/v1/*`，使用企业 OIDC 身份管理网关配置。
- 数据面：`/v1/*`，使用企业 OIDC Token 或独立 API Key 完成认证、准入、路由、转发与结算。
- 运维面：健康状态、使用日志、路由尝试和后续可观测性。
- 模型目录与上游模型库存分离；只有显式启用的模型和路由可对内发布。
- OIDC 调用使用用户级限额；API Key 调用必须同时满足用户级与 API Key 级限额。
- 用户工作台与管理员运行看板使用独立的数据范围和服务端权限校验。
- Usage Log、Route Attempt 和操作审计不记录请求/响应正文或任何明文凭据；用户主动创建
  的对话记录作为独立私有数据保存。
- SQLite 用于本地开发，同时保持 MySQL 支持。
- MVP 先支持 OpenAI 兼容上游，Provider Adapter 保留扩展边界但不过度插件化。

## 4. MVP 结果

企业管理员能够接入多套上游资源，发布统一模型并配置容错路由；OIDC 用户能够创建
直接归属于自己的、受模型范围和双层限额约束的 API Key；内部应用能够通过 OpenAI 兼容 API 调用模型，
并获得可审计、可解释的用量与路由结果。
