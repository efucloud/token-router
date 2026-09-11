# 数据面与路由

## 1. OpenAI 兼容入口

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/embeddings`

请求使用 `Authorization: Bearer tr_xxx`。错误采用 OpenAI 风格结构，并额外通过响应头
返回 `X-Request-Id`。未知字段原则上透传，网关拥有并可能改写 `model`、stream usage
选项和 Authorization。Responses API 普通、流式与工具调用字段透传，首版明确拒绝
`background=true`。

## 2. 请求处理管线

```text
API Key 认证
  → 用户/Token/模型范围检查
  → token 估算与双层限额预留
  → 候选路由筛选和排序
  → 上游尝试/必要时故障切换
  → 普通 JSON、Chat NDJSON 或 Responses SSE 转发
  → usage 结算
  → Usage Log + Route Attempt
```

任何阶段均携带同一个请求 ID。准入失败不会访问上游，并记录失败 Usage Log，但不增加
成功请求次数。

## 3. 模型发布与可见性

`GET /v1/models` 只返回同时满足以下条件的统一模型：

1. 模型状态为 `active`。
2. Token 允许该模型。
3. 至少存在一条路由、Provider 和 Channel 均启用，且渠道不处于 cooldown 或 half-open。

上游 `/models` 发现结果不得直接出现在数据面。

## 4. 选路算法

1. 根据统一模型查询已启用 Model Route。
2. 排除禁用的 Provider/Channel、处于 cooldown 的资源和已达并发上限的资源。
3. 选择最高有效优先级组。
4. 在同优先级组内按正整数权重随机选择，避免固定顺序热点。
5. 失败切换时排除本请求已尝试的路由，并重复上述过程。

MVP 采用进程内安全随机或并发安全伪随机选择。多实例间不要求严格全局权重一致。

## 5. 故障切换与 cooldown

- 网络连接、TLS、超时和上游 5xx 可切换到未尝试路由。
- 408、429 是否切换由错误分类决定；MVP 对上游 429 可切换，但客户端配额 429 不切换。
- 上游 4xx 默认不切换，避免重复无效请求。
- 一旦响应头或响应体开始写给客户端，不再重试。
- 每次尝试均写 Route Attempt。
- 资源连续失败达到阈值后进入配置时长的 cooldown；到期后仅允许一个健康探测进入
  half-open，探测成功才恢复生产流量，失败则重新进入 cooldown。

默认值由 `gateway` 配置提供：上游超时、最大尝试次数、失败阈值和 cooldown 秒数。

## 6. 普通响应

网关读取受大小限制的 JSON 响应，以解析 `usage` 和规范化错误。成功时保留上游主要
响应结构，将模型名改回请求中的统一模型名。超过限制或格式非法时按上游协议错误处理。

## 7. HTTP 流式响应

- Chat Completions 的 `stream=true` 响应使用 HTTP 分块 NDJSON（`application/x-ndjson`），
  每行一个完整 JSON chunk，不使用 WebSocket；兼容上游的 SSE 在网关边界转换为 NDJSON。
- 对 chat stream 请求强制向兼容上游追加 `stream_options.include_usage=true`。
- Responses stream 不追加 Chat Completions 专用参数，从 `response.completed` 事件提取 usage。
- 在写出首个响应字节前允许故障切换，开始写出后只可终止连接。
- 流式解析只提取 usage 和结束状态，不持久化内容增量。
- 上游正常结束但无 usage 时使用预留量结算并标记 `estimated=true`。
- 客户端中途断开时取消上游请求；已消耗量无法可靠获得时按预留量保守结算。

## 8. Provider Adapter 边界

MVP 的 `openai-compatible` Adapter 负责请求 URL、Authorization、模型名改写、错误分类
和 usage 提取。路由、限额、审计与 HTTP 生命周期位于 Adapter 外部。后续增加原生协议
时实现稳定接口，不引入插件市场或动态加载生命周期。
