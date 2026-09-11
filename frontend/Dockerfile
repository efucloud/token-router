# 构建阶段
FROM registry.cn-shenzhen.aliyuncs.com/efucloud-public/node:22-bookworm AS builder

WORKDIR /app

# 1. 先只复制依赖文件（触发缓存）
COPY package.json package-lock.json .npmrc ./

# 2. 安装依赖（含 devDependencies，因为 Umi 需要）
# 使用 npm ci 确保依赖与 package-lock.json 一致
RUN npm ci

# 3. 再复制源码（这一步变动频繁，放最后）
COPY . .

# 4. 执行构建
RUN npm run build


# 运行阶段
FROM registry.cn-shenzhen.aliyuncs.com/efucloud-public/nginx:1.29.4-alpine

ARG GIT_COMMIT
ARG BUILD_DATE

LABEL org.opencontainers.image.revision=${GIT_COMMIT}
LABEL org.opencontainers.image.created=${BUILD_DATE}
LABEL com.efucloud.build.commit=${GIT_COMMIT}
LABEL com.efucloud.build.date=${BUILD_DATE}

# 从 builder 复制构建产物
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
