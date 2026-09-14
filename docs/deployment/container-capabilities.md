# 容器内 Skills 与 MCP

官方后端镜像直接支持进程内置工具、只读 Skills、Streamable HTTP MCP 和 Node/Python stdio
MCP。服务监听 `9006`，以 UID/GID `10001` 运行，不需要也不应给容器提权。

## 挂载契约

| 容器路径 | 用途 | 建议卷类型 |
| --- | --- | --- |
| `/efucloud/config/config.yaml` | Token Router 主配置 | ConfigMap 或 Secret，只读 |
| `/efucloud/skills` | `<skill>/SKILL.md` Skill 树 | ConfigMap、镜像或只读卷 |
| `/efucloud/mcp` | stdio MCP 程序及已安装依赖 | 镜像或只读卷 |
| `/efucloud/workspaces` | 所有登录用户个人工作区的共同根目录 | PVC（推荐）或受控目录挂载 |
| `/efucloud/.cache`、`/efucloud/.home` | 非 root Node/Python 运行时缓存 | `emptyDir` 或普通卷 |

后端不使用环境变量覆盖运行配置。容器配置应在 `chat.skillDirectories` 中填写 Skill 目录，
例如 `[/efucloud/skills]`；需要扫描多个目录时直接增加数组项。

## 进程内置工具

开启 `chat.builtinTools.enabled` 后，后端进程直接发布 `read_file`、`list_files`、
`search_files`、`write_file`、`edit_file`、`apply_patch` 和 `discover_commands`。开启
`chat.builtinTools.commandEnabled` 后还会发布 `command`。这些工具无需部署独立 MCP Server；
在能力接口和会话中以 `builtin` 工具提供者出现。
个人文件工具对所有已认证账号发布，服务端在 `workspaceDirectory` 下使用账号 ID 的稳定哈希
子目录强制隔离。`chat.builtinTools.commandEnabled=true` 时，`command` 对所有已认证用户发布。

服务启动时扫描自身 `PATH` 中的可执行程序。`command` 的模型说明会列出检测到的常用工具，
模型也可调用 `discover_commands` 按名称查询。因此派生镜像只要安装 `kubectl`、`helm`、
`git` 等 CLI，模型就能发现并通过 `command` 使用它们，无需给 Token Router 增加对应业务
工具代码。容器配置应将 `chat.builtinTools.workspaceDirectory` 设为
`/efucloud/workspaces`。单文件上传默认限制为 32 MiB，可通过同一配置块中的
`maxUploadBytes` 调整。

`command` 不是安全沙箱，它拥有 Token Router 进程的文件、网络和 Kubernetes ServiceAccount
权限。生产环境必须使用非 root 用户、只读根文件系统、专用可写工作卷、最小化 RBAC，并仅在
可信用户可访问的部署中启用。工具自身提供工作目录边界、超时和输出大小限制，但 Shell 命令
仍可能通过绝对路径或网络访问工作目录之外的资源。

MCP 服务直接写入主配置文件的 `chat.mcpServers`：

```yaml
chat:
  mcpServers:
    - name: workspace
      type: stdio
      enabled: true
      command: /usr/bin/node
      args: [/efucloud/mcp/workspace/dist/index.js]
      workingDir: /efucloud/mcp/workspace
      timeoutSeconds: 30
```

stdio 的 `command` 以及非空 `workingDir` 必须使用绝对路径。HTTP MCP 的鉴权请求头、stdio
环境变量等敏感值应随主配置放在 Secret 中，不应提交到仓库或写入镜像层。

`logConfig.filename` 留空时后端只写 stdout，适合只读根文件系统；确需文件日志时将其设置为
`/efucloud/log/token-router.log` 并挂载可写日志卷。

## 部署示例

- [Docker Compose 示例](../../deploy/examples/capabilities/docker-compose.yaml)
- [Kubernetes 示例](../../deploy/examples/capabilities/kubernetes.yaml)

Compose 示例从示例目录挂载 `config.yaml`、`skills` 和 `mcp`。Kubernetes 示例假定已经存在
包含完整 `config.yaml` 的 Secret `token-router-config`，以及 stdio 程序 PVC
`token-router-mcp-programs`；只使用 HTTP MCP 时可删除该 PVC 和挂载。

建议优先将 MCP 部署为同网络的 Streamable HTTP sidecar 或独立服务。内置 Node.js/npm、
Python 3/pip 只提供基础运行时；生产环境应在构建阶段固定 MCP 版本并安装依赖。需要浏览器、
JVM 或系统库时，构建派生镜像，不要在启动脚本中联网安装。
