# 双维度 Dashboard

## 1. 目标与边界

系统提供两套 Dashboard，而不是在同一个接口上依赖前端传入 scope：

| Dashboard | 路由 | API | 数据范围 |
| --- | --- | --- | --- |
| 我的工作台 | `/dashboard` | `GET /api/v1/dashboard/me` | 当前 OIDC 用户 |
| 管理员看板 | `/admin/dashboard` | `GET /api/v1/dashboard/admin` | 企业网关全局 |

管理员可以查看两套看板；其他角色只能访问自己的工作台。服务端必须分别实现权限和
查询范围，禁止 `/dashboard/me` 接受 `accountId` 覆盖当前用户。

## 2. 通用请求与元数据

两个接口接受 `range=24h|7d|30d`，默认 `24h`。响应包含：

```json
{
  "range": {
    "start": "2026-09-09T00:00:00+08:00",
    "end": "2026-09-10T00:00:00+08:00",
    "timezone": "Asia/Shanghai",
    "granularity": "hour"
  },
  "generatedAt": "2026-09-10T00:00:01+08:00"
}
```

`24h` 使用小时粒度，`7d` 和 `30d` 使用日粒度。每个区间返回连续 bucket，无数据的
bucket 补零；延迟分位数无样本时返回 `null`。

## 3. 我的工作台响应

`GET /api/v1/dashboard/me` 返回：

| 字段 | 内容 |
| --- | --- |
| `quota` | 本人用户级 limit、used、reserved、remaining 和 usageRatio |
| `tokenStatus` | active、disabled、expired、expiringSoon Token 数 |
| `summary` | 成功/失败/拒绝请求、成功率、输入/输出/总 token、估算占比、平均/P95 延迟 |
| `previousSummary` | 紧邻上一周期的同口径汇总，用于环比 |
| `trend` | 连续时间 bucket 的请求、错误、输入/输出 token 和延迟 |
| `modelUsage` | 本人按统一模型聚合的请求、token、成功率 |
| `errorBreakdown` | 本人按内部错误分类聚合的数量 |
| `recentRequests` | 最近 10 条调用，不含请求/响应正文 |
| `alerts` | 本人限额、Token 到期和 Token 状态提醒 |

限额 `0` 表示不限制，此时 `remaining` 和 `usageRatio` 返回 `null`。Token 即将在 7 天内
过期计入 `expiringSoon`。最近调用只展示请求 ID、时间、Token 前缀、模型、状态、token
用量和耗时。

## 4. 管理员看板响应

`GET /api/v1/dashboard/admin` 返回：

| 字段 | 内容 |
| --- | --- |
| `summary` / `previousSummary` | 全局请求、token、成功率、平均/P95 延迟及环比 |
| `principals` | 活跃用户、活跃 Token、接近限额用户和 Token 数 |
| `resourceHealth` | active 模型及 healthy、cooldown、disabled、misconfigured 渠道数 |
| `trend` | 全局连续时间 bucket 的请求、错误、token、延迟和故障切换数 |
| `modelUsage` | 各统一模型的请求、token、成功率和 P95 延迟 |
| `userUsage` | 各用户的请求、token、成功率和限额比例，默认前 10 |
| `channelUsage` | 各渠道的尝试、成功率、P95 延迟和 cooldown 状态 |
| `routing` | 上游尝试、发生切换的请求、切换成功请求和高失败率路由 |
| `quotaRisks` | 达到 80% 或已耗尽限额的用户/Token |
| `recentFailures` | 最近 10 条最终失败请求及 Route Attempt 摘要 |

`userUsage` 用于容量治理而非绩效排名，界面文案使用“用量分布”。管理员响应仍不得包含
邮箱之外的额外身份信息、API Key 明文、上游密钥或 payload。

## 5. 指标口径

- 活跃用户/Token：所选时间范围内至少有一次通过准入的数据面请求。
- 成功请求：Usage Log 最终状态为 `succeeded`。
- 最终失败：访问过上游但所有尝试均失败；准入拒绝单独统计。
- 成功率：`成功 / (成功 + 最终失败)`，没有分母时返回 `null`。
- 故障切换请求：同一请求存在至少两条 Route Attempt。
- 切换成功请求：发生故障切换且最终状态为 `succeeded`。
- P95：只使用访问过上游且有完整耗时的请求。
- 额度风险：有限额且 `(used + reserved) / limit >= 0.8`。

## 6. 性能与缓存

- MVP 直接聚合 Usage Log 和 Route Attempt，并确保时间、用户、模型、渠道索引可用。
- Dashboard 查询最大范围为 30 天，单接口目标 P95 小于 1 秒。
- 可使用不超过 30 秒的进程内缓存；缓存键必须包含 Dashboard 类型、用户 ID、时间范围
  和时区，管理员与用户缓存不可共享。
- 新调用完成后不要求主动失效，界面显示 `generatedAt` 并提供手动刷新。

## 7. 验收

1. 两个用户存在调用数据时，任一用户工作台均不能推断另一用户的数据。
2. `edit`、`view`、`none` 请求管理员接口均返回 403。
3. 管理员可以在全局看板和本人工作台之间切换，两者统计范围正确。
4. 无限额、无调用、全部失败、缺失 usage 和跨日时区场景展示正确。
5. 两个接口及前端页面不出现价格、余额、项目或任何明文凭据。
