FROM alpine:3.19

ARG TARGETARCH

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

RUN mkdir -p /app/data /app/config && chmod 755 /app/data /app/config

# 根据目标架构自动选择预编译的静态二进制
COPY easypanel-linux-${TARGETARCH}-bin /app/easypanel
RUN chmod +x /app/easypanel

VOLUME ["/app/data", "/app/config"]

EXPOSE 3088

ENTRYPOINT ["/app/easypanel"]
