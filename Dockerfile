FROM golang as build

ENV CGO_ENABLED 0
ENV GO111MODULE on
ENV GOPROXY "https://goproxy.cn,direct"

# 下载程序
RUN go install github.com/leig/thor@latest

FROM alpine as run

MAINTAINER leigme

# 复制编译完成的程序到用户目录
COPY --from=build /go/bin/thor /usr/local/bin/thor

# 声明服务端口
EXPOSE 8080

# 启动容器时运行的命令
ENTRYPOINT  ["thor"]