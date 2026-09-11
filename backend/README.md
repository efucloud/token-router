# Token Router Backend

企业内部统一 AI 模型聚合与分发网关后端。控制面使用 OIDC，数据面支持 OIDC Bearer
Token 和用户创建的 `tr_` API Key；API Key 直接归属用户，不引入项目、价格、充值或
兑换码体系。

开发与构建统一使用 Go 1.26.4。产品和技术边界见[总体规格](../docs/spec.md)。

## 本地配置

`config/config.yaml` 默认连接本地 MySQL `token_router`。生产环境建议使用以下环境变量
覆盖敏感配置：

- `TOKEN_ROUTER_OIDC_ISSUER`
- `TOKEN_ROUTER_OIDC_CLIENT_ID`
- `TOKEN_ROUTER_OIDC_CLIENT_SECRET`
- `TOKEN_ROUTER_OIDC_SKIP_CLIENT_ID_CHECK`（`true` 时允许同一 Issuer 的跨客户端 Token）
- `TOKEN_ROUTER_GATEWAY_SECRET_KEY`

`oidcConfig.skipClientIDCheck` 默认为 `false`，此时 Token 的 `aud` 必须包含配置的
`clientId`。需要让共享同一 OIDC 的多个系统互通 Token 时设为 `true`；Issuer、签名、
有效期和本地账户状态仍会被校验。

未配置 OIDC issuer 时，服务仍可启动数据面，但所有需要 OIDC 的控制面接口均无法通过
认证，不会自动降级为匿名管理模式。

## 本地上游引导

使用环境变量传入上游密钥，命令不会将明文写入数据库：

```shell
UPSTREAM_API_KEY='your-key' go run ./cmd/bootstrap \
  --config ./config/config.yaml \
  --base-url 'https://example.com/compatible-mode/v1' \
  --model 'internal-model-name' \
  --upstream-model 'provider-model-name'
```

命令会创建本地管理员、Provider、Channel、统一模型、显式路由和一次性测试 Token。

## 生成前端接口代码

在本目录运行：

```shell
GOFLAGS=-mod=mod go run ./generate
```

生成内容会直接写入相邻的 `frontend/src/services` 目录，并清理其中的旧文件。
