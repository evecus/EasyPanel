FROM scratch

WORKDIR /app/easypanel

# 复制编译好的二进制
COPY easypanel-linux-amd64-bin /app/easypanel/easypanel

# 声明挂载点（数据和配置目录）
VOLUME ["/app/easypanel/data", "/app/easypanel/config"]

# 默认端口
EXPOSE 3088

ENTRYPOINT ["/app/easypanel/easypanel"]
