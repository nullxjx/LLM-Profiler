# 使用包含Golang的基础镜像构建源码
FROM golang:1.20.5-bullseye as builder

ENV TZ=Asia/Shanghai

# 设置工作目录
WORKDIR /app

# 将项目根目录下的go.mod和go.sum复制到工作目录
COPY ../go.mod go.sum ./

# 下载依赖
RUN go mod download

# 将整个项目目录复制到工作目录
COPY .. .

# 编译main.go
RUN go build -o perf_tester main.go

FROM debian:bullseye-slim

# 将编译好的二进制文件复制到容器中
COPY --from=builder /app/perf_tester /workspace/perf_tester
COPY --from=builder /app/data /workspace/data
COPY --from=builder /app/config /workspace/config

RUN chmod +x /workspace/perf_tester

# 暴露端口
EXPOSE 8088

# 设置工作目录为 /workspace
WORKDIR /workspace

# 设置容器启动时运行的命令
CMD ["/workspace/perf_tester"]