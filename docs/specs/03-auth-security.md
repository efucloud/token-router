# 认证与安全

## 1. 两类身份

| 平面 | 凭据 | 用途 |
| --- | --- | --- |
| 控制面 `/api/v1` | 企业 OIDC Bearer Token | 人员登录、角色和资源管理 |
| 数据面 `/v1` | 企业 OIDC Bearer Token 或 `tr_` API Key | 内部应用调用模型 |

OIDC Token 可以访问数据面；API Key 不能访问管理接口。

## 2. OIDC 控制台认证

- 使用 Authorization Code Flow，由企业 IdP 完成认证。
- 前端登录前生成并保存 `state`，回调时强制校验。
- 前端检查 token 过期时间，为管理请求注入 Bearer Token，收到 401 后重新登录。
- 后端始终校验 issuer、签名和过期时间，并将 `eAuthId`、`id` 或标准 `sub` 映射到 Account。
- `oidcConfig.skipClientIDCheck=false` 时校验 audience/client ID；设为 `true` 时允许同一
  Issuer 为其他客户端签发的 Token，以支持共享同一 OIDC 的内部系统互调。
- 管理员邮箱只用于首次引导；实际接口授权以数据库 Account 角色为准。
- 服务端请求上下文应保存账户 ID，权限过滤器按 ID 查询账户并校验角色。

## 3. API Key 生命周期

- 由 `crypto/rand` 生成至少 256 bit 随机数据，编码为 URL-safe 字符并添加 `tr_` 前缀。
- 创建响应仅一次返回明文；持久化 SHA-256 摘要和不可用于认证的短前缀。
- 认证时对输入计算 SHA-256，通过唯一索引查找，并检查 Token 状态、过期时间和用户状态。
- 列表、日志和错误信息仅展示短前缀。
- 删除或禁用后立即拒绝新请求；进行中的请求按开始时的准入结果完成结算。
- 每个 Token 可独立配置 IP/CIDR 白名单、RPM、TPM 和最大并发；空白名单和数值 `0`
  分别表示不限制。

## 4. 上游密钥

- 部署配置 `gateway.secretKey` 提供主密钥，不写入数据库或返回前端。
- 主密钥解码后必须为 32 bytes，使用 AES-256-GCM。
- 每次加密使用新的随机 nonce，密文中保存可版本化格式和 nonce。
- 只有发起上游请求的适配器能够在内存中解密；日志与错误禁止输出密钥。
- 主密钥为空时允许读取不依赖渠道的控制面能力，但创建/更新密钥及数据面调用应返回明确配置错误。

## 5. 权限与数据范围

- `admin` 可以管理所有对象和账户限额。
- `edit` 可以管理运行配置和自己的 Token，但不能修改用户角色或用户级限额。
- `view` 只读运行配置，仅能管理自己的 Token。
- `none` 仅访问自己的 Token、限额和用量。
- 对象归属必须由服务端从认证上下文推导，禁止信任客户端传入的 `account_id`。

## 6. 网络与日志安全

- `base_url` 仅允许 `http`/`https`；禁止 URL userinfo，并规范化尾部 `/`。
- 生产部署应通过出站网络策略限制 SSRF；应用层拒绝明显非法地址但不替代网络隔离。
- 上游重定向默认禁用，避免 Authorization 被带到未知主机。
- 日志禁止记录 Authorization、Cookie、请求正文、响应正文、OIDC Token、API Key 和上游密钥。
- 错误响应使用稳定错误码和请求 ID，不透传可能包含敏感信息的上游正文。
- IP 白名单默认只匹配服务端连接看到的 `RemoteAddr`，不信任客户端可伪造的
  `X-Forwarded-For`。部署在可信反向代理之后时，应由入口代理清洗来源地址并采用受控的
  real-IP 配置，应用再显式支持可信代理网段。
