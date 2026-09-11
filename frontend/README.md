# Token Router Frontend

基于 React 19、Ant Design 6 和 Ant Design Pro 6 的企业 AI 网关控制台，包含 OIDC
认证、用户工作台、管理员运行看板和模型聊天窗口。

## 开发

```shell
npm install
npm run dev
```

开发服务器将 `/api` 和 `/v1` 代理到 `http://localhost:9006`。可通过
`API_PROXY_TARGET` 修改后端地址。

聊天页要求用户输入自己的 `tr_` API Key。API Key 只保存在页面内存中，不会持久化；
OIDC Token 仅用于控制台 `/api/v1`，两种凭据不会混用。

## 生成接口代码

在项目的 `backend` 目录运行：

```shell
GOFLAGS=-mod=mod go run ./generate
```

生成器会清理并重新生成 `src/services`。
