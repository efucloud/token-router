# 领域模型

## 1. 实体关系

```text
Account ── owns / quota subject ── API Key
    └──── owns ── Chat Conversation ── Chat Message

AI Model ── Model Route ── Provider Resource ── Provider
    │              │
    └──── Usage Log ┴── Route Attempt
```

所有核心实体使用数据库 ID 作为内部关联键，并包含创建、更新时间。被日志引用或仍有
关联对象的配置应拒绝删除，运维场景优先使用状态禁用。

## 2. AIModel

模型目录是对内发布能力的唯一来源，上游发现结果不能自动成为公开模型。

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `name` | string, unique | 对内稳定模型名 |
| `display_name` | string | 展示名称 |
| `modality` | enum | `chat` / `embedding` |
| `context_window` | int | 能力描述，非价格字段 |
| `capabilities` | JSON string array | streaming、tools、vision、reasoning 等 |
| `status` | enum | `active` / `disabled` |

只有 `active` 且至少存在一条可用路由的模型出现在 `/v1/models`。

## 3. Provider

Provider 描述适配器类型和供应商公共信息，不保存具体资源密钥。

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `name` | string, unique | 供应商名称 |
| `type` | string | MVP 为 `openai-compatible` |
| `status` | enum | `enabled` / `disabled` |
| `config` | JSON object | 非敏感公共适配配置 |

## 4. Channel（Provider Resource）

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `provider_id` | FK | 所属 Provider |
| `name` | string | 资源名称 |
| `base_url` | URL | 上游基础地址 |
| `encrypted_api_key` | bytes/text | AES-GCM 密文 |
| `priority` | int | 值越大越优先 |
| `weight` | int > 0 | 同优先级加权选择 |
| `status` | enum | `enabled` / `disabled` |
| `timeout_seconds` | int | 上游超时 |
| `max_concurrency` | int | `0` 表示不限制 |
| `health_status` | enum | `unknown` / `healthy` / `cooldown` |
| `cooldown_until` | nullable time | 冷却结束时间 |

管理 API 永不返回 `encrypted_api_key`；只返回 `credential_configured`。更新时未提供新
密钥表示保留旧值，显式清除密钥不在 MVP 范围。

## 5. ModelRoute

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `model_id` | FK | 统一模型 |
| `channel_id` | FK | 供应商资源 |
| `upstream_model` | string | 上游请求中的模型名 |
| `priority` | int | 路由级优先级，可覆盖资源默认值 |
| `weight` | int > 0 | 路由级权重 |
| `status` | enum | `enabled` / `disabled` |

`model_id + channel_id + upstream_model` 唯一。

## 6. APIToken

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `account_id` | FK | 创建者和限额归属用户 |
| `name` | string | Token 名称 |
| `key_hash` | 32-byte/hex, unique | SHA-256 摘要 |
| `key_prefix` | string | 列表识别前缀 |
| `status` | enum | `active` / `disabled` |
| `expires_at` | nullable time | 过期时间 |
| `allowed_models` | JSON string array | 模型白名单 |
| `token_limit` / `used_tokens` | int64 | token 限额与累计 |
| `request_limit` / `used_requests` | int64 | 成功请求次数限额与累计 |
| `ip_allowlist` | JSON string array | 后续启用的契约字段 |
| `rpm` / `tpm` / `max_concurrency` | int | 后续启用的窗口限制字段 |

API Key 格式为 `tr_` 加安全随机串。明文只在创建响应中返回一次。

## 7. UsageLog 与 RouteAttempt

UsageLog 每个数据面请求一条，包含：请求 ID、用户、Token、统一模型、最终渠道、
接口、输入/输出/总 token、预留 token、HTTP 状态、耗时、错误分类、是否估算和时间。

RouteAttempt 每次上游尝试一条，包含：请求 ID、路由、渠道、序号、HTTP 状态、耗时、
错误分类和时间。日志不得包含请求/响应正文或任何凭据。

## 8. Account 限额扩展

现有 Account 增加：`token_limit`、`request_limit`、`used_tokens`、`used_requests`。
`0` 表示该维度不限制。累计值只能由限额事务服务更新，不允许通用账户更新接口写入。

## 9. ChatConversation 与 ChatMessage

ChatConversation 按 `account_id` 归属当前 OIDC 用户，保存标题、当前统一模型和更新时间。
ChatMessage 保存会话内顺序、角色、正文和该轮 Token 数。首条用户消息自动生成会话标题；
删除会话时同时删除其消息。所有查询、更新和删除必须同时校验会话 ID 与当前账户 ID。

对话正文属于用户主动保存的私有业务数据，不进入 Usage Log、Route Attempt 或操作审计。
