# 限额与用量

## 1. 限额维度

用户和 API Key 均维护：

- `tokenLimit` / `usedTokens`
- `requestLimit` / `usedRequests`

`0` 表示该维度不限制。请求必须同时满足用户级与 Token 级约束；Token 设置为无限并不
绕过用户限额。MVP 的 request 只统计完成上游调用的成功请求，准入失败和最终上游失败
不增加 `usedRequests`。

API Key 还可维护 `rpm`、`tpm` 和 `maxConcurrency`。RPM 统计进入网关的准入请求，
TPM 在调用前按预估 Token 预留并在结算时替换为真实用量；最大并发通过带过期时间的
租约实现。三者为 `0` 时不限制。

## 2. token 预估

- chat：按消息文本、工具定义和输出上限计算保守预留量。
- embeddings：按输入计算预留量，不预留输出 token。
- 无法精确分词时使用按 UTF-8 字节/字符的保守估算法，并设定合理最小值。
- 预留量必须有服务端上限，防止客户端用极大 `max_tokens` 锁死额度。

估算器是独立接口，后续可按模型选择精确 tokenizer。

## 3. 准入与预留

数据库事务中锁定 Account 和 APIToken，按固定顺序执行：

1. 校验用户/Token 状态和过期时间。
2. 检查 `used + reserved + estimate <= tokenLimit`。
3. 检查 `usedRequests + reservedRequests + 1 <= requestLimit`。
4. 锁定当前 Token 的分钟桶，检查 RPM/TPM 并写入请求数和 Token 预留。
5. 清理已过期并发租约，检查最大并发并创建本次请求租约。
6. 同时增加用户和 Token 的预留 token 与预留请求数。
7. 创建状态为 `reserved` 的用量记录并提交事务。

SQLite 使用写事务串行化；MySQL 使用行锁或原子条件更新。任何一步失败都不产生部分
预留。额度不足返回 429 和稳定错误码 `quota_exceeded`。
RPM、TPM 或并发拒绝同时返回 `Retry-After`、`X-RateLimit-Limit` 和
`X-RateLimit-Reset`。

## 4. 结算

成功响应后在事务中：

- 从用户和 Token 释放本次预留。
- 按上游 usage 增加输入、输出和总 token 累计。
- 用户和 Token 的 `usedRequests` 各增加 1。
- 更新 Usage Log 为 `succeeded`。

实际 token 可高于预留；结算仍记录真实值，之后请求会被限额拒绝。上游缺失 usage 时
按预留 token 扣减并将 `estimated=true`。

最终失败时释放预留，不增加 used token 和成功请求次数，并将 Usage Log 标为 `failed`。
进程异常可能留下预留，后台回收任务按超时和请求状态幂等释放。
结算无论成功或失败均删除本次并发租约；进程异常遗留的租约最多保留一小时并在后续
准入时清理。

## 5. 幂等与状态机

用量记录状态：`reserved → succeeded | failed | expired`。结算以请求 ID 为幂等键，只有
`reserved` 状态可以转换，重复结算不重复扣减。回收器只能把超时的 `reserved` 转为
`expired`。

## 6. 用量日志

Usage Log 保存：

- 请求 ID、时间、用户、Token ID/前缀。
- 统一模型、最终 Channel、接口和是否流式。
- 输入、输出、总 token 和预留 token。
- 状态、HTTP 状态、耗时、错误分类、是否估算。

错误摘要只存内部分类和脱敏短信息。日志不保存 payload、上游响应正文或凭据。

## 7. 汇总

控制面可按日、模型、用户和渠道聚合成功请求、失败请求与 token 用量。汇总是
运行治理数据，不包含金额或价格字段。MVP 直接查询 Usage Log；数据量增长后再引入
异步聚合表。

## 8. Dashboard 聚合

用户工作台的所有统计均以当前 OIDC Account 为强制过滤条件，包括其全部 API Key。
管理员运行看板使用全局数据范围。两个看板共享同一套 Usage Log 聚合口径：

- 请求数只区分成功、失败与准入拒绝，成功率只以到达上游后的完成请求计算。
- token 用量展示输入、输出和总量；估算值计入总量并单独返回估算占比。
- 延迟提供平均值和 P95；无数据时返回 `null`，不返回虚构的 0 延迟。
- 时间趋势统一按服务端时区边界聚合，并在响应中返回时区与时间范围。
