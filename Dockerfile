# 构建go项目
FROM golang:1.25-alpine AS builder

WORKDIR /app
# 安装git等构建依赖
RUN apk add --no-cache git

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cfnb .

# 复制构建好的二进制文件
COPY --from=builder /app/cfnb /usr/local/bin/cfnb
# 设置工作目录
WORKDIR /app
# 暴露端口
EXPOSE 8080
# 启动应用
CMD ["cfnb"]
