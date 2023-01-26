FROM golang:alpine

MAINTAINER leigme

# 为我们的镜像设置必要的环境变量
ENV GO111MODULE on
ENV GOPROXY "https://goproxy.cn,direct"

WORKDIR $GOPATH/src/thor

ADD . ./

RUN go build -o thor .

# 声明服务端口
EXPOSE 8080

# 启动容器时运行的命令
ENTRYPOINT  ["./thor"]