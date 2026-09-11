# 容器内 Skills 与 MCP

官方后端镜像直接支持只读 Skills、Streamable HTTP MCP 和 Node/Python stdio MCP。服务监听
`9006`，以 UID/GID `10001` 运行，不需要也不应给容器提权。

## 挂载契约

| 容器路径 | 用途 | 建议卷类型 |
| --- | --- | --- |
| `/efucloud/config/config.yaml` | Token Router 主配置 | ConfigMap 或 Secret，只读 |
| `/efucloud/config/mcp.yaml` | 独立 MCP 配置 | Secret，只读 |
| `/efucloud/skills` | `<skill>/SKILL.md` Skill 树 | ConfigMap、镜像或只读卷 |
| `/efucloud/mcp` | stdio MCP 程序及已安装依赖 | 镜像或只读卷 |
| `/efucloud/.cache`、`/efucloud/.home` | 非 root Node/Python 运行时缓存 | `emptyDir` 或普通卷 |

镜像默认设置 `TOKEN_ROUTER_CHAT_SKILL_DIRECTORIES=/efucloud/skills`。需要扫描多个目录时，
使用 Linux 路径列表格式，例如 `/company/skills:/team/skills`。该变量一旦设置，会覆盖主配置
中的 `chat.skillDirectories`。

设置 `TOKEN_ROUTER_CHAT_MCP_CONFIG_FILE=/efucloud/config/mcp.yaml` 后，独立文件中的
`mcpServers` 会覆盖主配置的同名列表。文件采用以下结构：

```yaml
mcpServers:
  - name: workspace
    type: stdio
    enabled: true
    command: /usr/bin/node
    args: [/efucloud/mcp/workspace/dist/index.js]
    workingDir: /efucloud/mcp/workspace
    timeoutSeconds: 30
```

配置文件不可读、字段拼错或包含多个 YAML document 时，进程会在启动阶段直接失败。stdio
的 `command` 以及非空 `workingDir` 必须使用绝对路径。HTTP MCP 的鉴权请求头、stdio 环境
变量等敏感值应放在 Secret 中，不应提交到仓库或写入镜像层。

`logConfig.filename` 留空时后端只写 stdout，适合只读根文件系统；确需文件日志时将其设置为
`/efucloud/log/token-router.log` 并挂载可写日志卷。

## 部署示例

- [Docker Compose 示例](../../deploy/examples/capabilities/docker-compose.yaml)
- [Kubernetes 示例](../../deploy/examples/capabilities/kubernetes.yaml)
- [独立 MCP 配置示例](../../deploy/examples/capabilities/mcp.yaml)

Compose 示例要求通过环境变量传入本地主配置、Skill 和 MCP 程序目录。Kubernetes 示例假定
已经存在包含 `config.yaml` 的 `token-router-config`、运行时环境变量 Secret
`token-router-environment`，以及 stdio 程序 PVC `token-router-mcp-programs`；只使用 HTTP MCP
时可删除该 PVC 和挂载。

建议优先将 MCP 部署为同网络的 Streamable HTTP sidecar 或独立服务。内置 Node.js/npm、
Python 3/pip 只提供基础运行时；生产环境应在构建阶段固定 MCP 版本并安装依赖。需要浏览器、
JVM 或系统库时，构建派生镜像，不要在启动脚本中联网安装。
