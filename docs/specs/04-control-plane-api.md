# 控制面 API

## 1. 通用约定

- 前缀：`/api/v1`。
- 认证：企业 OIDC Bearer Token。
- JSON 字段使用 `lowerCamelCase`，时间使用 RFC 3339。
- 列表统一支持 `page`、`pageSize`，返回 `items`、`total`、`page`、`pageSize`。
- 写接口校验资源状态和关联关系；冲突返回 409，输入错误返回 400，无权限返回 403。
- 删除有引用的对象返回 409，并提示先禁用或解除关联。

## 2. 接口目录

### 部署探针

- `GET /livez`：仅表示进程可响应，不依赖数据库或外部供应商。
- `GET /readyz`：检查数据库连接、关键表迁移状态和 32 字节网关主密钥；未就绪返回 503。
- 探针不检查外部模型供应商，避免上游故障导致网关实例被部署平台反复重启。

### 模型目录

- `GET /models`
- `POST /models`
- `GET /models/{id}`
- `PUT /models/{id}`
- `DELETE /models/{id}`
- `GET /models?channelId={id}`：按渠道返回统一模型、上游模型和对应路由状态。

### Provider 与供应商资源

- `GET/POST /providers`
- `GET/PUT/DELETE /providers/{id}`
- `GET/POST /channels`
- `GET/PUT/DELETE /channels/{id}`
- `GET /channels/{id}/test`：只测试连通性，不自动发布发现到的模型。
- `POST /channels/{id}/models/import`：选择发现到的上游模型，幂等创建默认启用的统一模型和
  渠道路由。
- `POST /channels/{id}/models/sync`：以上游模型列表为准执行全量对账；创建缺失映射，删除
  已失效的本渠道路由，并只删除不再被任何渠道引用的孤立统一模型。

渠道测试返回成功状态、HTTP 状态、延迟和最多 200 个发现模型 ID，并持久化最近检查
时间、延迟、健康状态和脱敏错误摘要。导入或同步创建的新统一模型默认为 `active`，
新路由默认为 `enabled`；已有模型和路由的人工启停状态保持不变。同步仅在
上游成功返回有效模型列表后修改数据库，渠道不可用时不得清理已有记录。

### 模型路由

- `GET/POST /routes`
- `GET/PUT/DELETE /routes/{id}`
- 列表支持按 `modelId`、`channelId`、`status` 筛选。

### 管理员模型演练

- `POST /model-diagnostics`
- 仅 `admin` 可调用；以流式上游请求测量 TTFT 和总耗时，并返回最终渠道、Token 用量及
  完整尝试链。
- 演练允许验证尚未发布但已有启用路由的聊天模型；提示词与输出只在响应中返回，不写入
  Usage Log、Route Attempt 或审计正文。

### API Key

- `GET/POST /tokens`
- `GET/PUT/DELETE /tokens/{id}`
- `POST /tokens/{id}/disable`
- 创建响应额外返回一次性字段 `token`；其余响应永不包含该字段。

### 用户对话记录

- `GET/POST /chat/conversations`
- `GET/PUT/DELETE /chat/conversations/{id}`
- `POST /chat/conversations/{id}/messages`
- 所有接口仅返回或修改当前 OIDC 用户自己的会话；首条用户消息自动生成标题。
- 对话消息是用户主动保存的业务数据，Usage Log、Route Attempt 和审计仍不记录正文。

### 用户和限额

- `GET /accounts`
- `PUT /accounts/{id}/role`
- `PUT /accounts/{id}/quota`
- 限额接口只允许修改 limit，不允许直接修改 used 累计值。

### 用量与运维

- `GET /usage`
- `GET /usage/summary`
- `GET /usage/{requestId}/attempts`
- `GET /route-attempts`
- 支持时间、用户、Token 前缀、模型、渠道、状态和错误分类筛选。

### 管理操作审计

- `GET /audit-logs`
- 所有通过 `admin` 权限校验的写操作记录操作者、动作、资源、变更字段名、结果、状态码、
  时间、来源 IP 和 Request ID。
- 审计中不保存请求正文、字段值、API Key、OIDC Token 或其他明文凭据。

### Dashboard

- `GET /dashboard/me`：当前用户工作台，任何已登录用户可访问。
- `GET /dashboard/admin`：管理员运行看板，仅 `admin` 可访问。
- 两个接口均支持 `range=24h|7d|30d`，默认 `24h`，趋势粒度由后端决定。
- `/dashboard/me` 的用户 ID 只能来自 OIDC 请求上下文，不接受客户端指定其他用户。
- `/dashboard/admin` 返回全局聚合和运行状态，不返回请求/响应正文或任何凭据。
- 完整响应字段和指标口径见[双维度 Dashboard](09-dashboard.md)。

## 3. 权限矩阵

| 资源 | admin | edit | view | none |
| --- | --- | --- | --- | --- |
| 模型/Provider/渠道/路由 | 管理 | 管理 | 只读 | 无 |
| API Key | 管理 | 自己的 Token | 自己的 Token | 自己的 Token |
| 账户角色/用户限额 | 管理 | 无 | 无 | 无 |
| 用量 | 全部 | 全部 | 全局汇总+自己明细 | 自己 |
| 用户工作台 | 自己 | 自己 | 自己 | 自己 |
| 管理员运行看板 | 查看 | 无 | 无 | 无 |
| 管理操作审计 | 查看 | 无 | 无 | 无 |
| 管理员模型演练 | 执行 | 无 | 无 | 无 |

## 4. 并发更新

配置对象响应包含单调递增的 `version`。更新请求携带已读取的 `version`，后端采用条件更新；
对象已被其他操作修改时返回 409，避免控制台覆盖并发变更。累计用量字段不参与通用
乐观锁，由限额服务单独事务更新。
