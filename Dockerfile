FROM alpine:3.19

ARG TARGETARCH

WORKDIR /app

# 安装 CA 证书（HTTPS 请求需要）
RUN apk add --no-cache ca-certificates tzdata

RUN mkdir -p /app/data && chmod 755 /app/data
RUN mkdir -p /app/config && chmod 755 /app/config

# 根据目标架构自动选择二进制
COPY easypanel-linux-${TARGETARCH}-bin /app/easypanel
RUN chmod +x /app/easypanel

# 声明挂载点（数据和配置目录）
VOLUME ["/app/data", "/app/config"]

# 默认端口
EXPOSE 3088

ENTRYPOINT ["/app/easypanel"]
