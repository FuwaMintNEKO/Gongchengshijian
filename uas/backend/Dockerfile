# UAS 后端多阶段构建（Linux 二进制）
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o uas-server .

# 运行阶段
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/uas-server .
COPY --from=builder /app/public ./public
COPY --from=builder /app/tools ./tools

RUN mkdir -p /app/uploads/avatar

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8081/api/health || exit 1

CMD ["./uas-server"]
