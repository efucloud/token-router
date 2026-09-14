# Token Router Backend

企业内部统一 AI 模型聚合与分发网关后端。控制面使用 OIDC，数据面支持 OIDC Bearer
Token 和用户创建的 `tr_` API Key；API Key 直接归属用户，不引入项目、价格、充值或
兑换码体系。

开发与构建统一使用 Go 1.26.4。产品和技术边界见[总体规格](../docs/spec.md)。
正式构建使用 Go `embed` 将前端静态资源编译进后端二进制。浏览器页面、API 和健康探针
均由同一进程的 `9006` 端口提供；前端路由由服务端回退到嵌入的 `index.html`。

## 本地配置

`config/config.yaml` 默认连接本地 MySQL `token_router`。后端运行配置只从 `-c`/`--config`
指定的 YAML 文件读取，环境变量不会覆盖配置值。生产环境应通过 Secret 或等价机制挂载完整
配置文件，并限制文件访问权限。

`oidcConfig.skipClientIDCheck` 默认为 `false`，此时 Token 的 `aud` 必须包含配置的
`clientId`。需要让共享同一 OIDC 的多个系统互通 Token 时设为 `true`；Issuer、签名、
有效期和本地账户状态仍会被校验。

未配置 OIDC issuer 时，服务仍可启动数据面，但所有需要 OIDC 的控制面接口均无法通过
认证，不会自动降级为匿名管理模式。

可选的 `redis` 配置块用于缓存已发布模型与网关路由；仅在 `redis.enabled: true` 时启用，
未配置 Redis 或显式关闭时使用数据库。缓存默认 TTL 为 30 秒。模型、供应商、
渠道、路由或渠道健康状态发生变化时，所有实例通过共享缓存代次立即失效旧数据；Redis
启动失败或运行中不可用时会自动回退到数据库。完整字段示例见 `config/config.yaml`。
`redis.mode` 支持 `standalone`、`sentinel` 和 `cluster`；三种模式统一使用 `addresses`，
其中 Sentinel 模式还需配置 `masterName`。

## 容器能力

后端镜像以 UID/GID `10001` 运行，监听 `9006`，并包含 Node.js/npm、Python 3/pip 供 stdio
MCP 使用。将主配置、Skills 和 stdio 程序分别只读挂载到
`/efucloud/config/config.yaml`、`/efucloud/skills` 和 `/efucloud/mcp`，MCP 服务定义直接写入
主配置的 `chat.mcpServers`。完整契约及 Compose/Kubernetes 示例见
[容器内 Skills 与 MCP](../docs/deployment/container-capabilities.md)。GitHub Actions 使用仓库
根目录作为构建上下文，依次完成前端构建、Go `embed` 编译和多架构镜像发布。

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
