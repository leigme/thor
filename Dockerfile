FROM golang:alpine

MAINTAINER leigme

# 为我们的镜像设置必要的环境变量
ENV GO111MODULE on
ENV GOPROXY "https://goproxy.cn,direct"
ENV GOPATH /app

WORKDIR /app

# 将我们的代码编译成二进制可执行文件app
RUN go install github.com/leigme/thor@latest

# 声明服务端口
EXPOSE 8080

# 启动容器时运行的命令
CMD ["/app/bin/thor"]