FROM gocv/opencv:4.12.0

# 构建参数
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG BUILD_ID=unknown
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译应用，注入版本信息
RUN go build -ldflags "\
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.BuildID=${BUILD_ID} \
    -X main.GitCommit=${GIT_COMMIT} \
    -X main.GitBranch=${GIT_BRANCH}" \
    -o layerkit .

# 创建上传目录
RUN mkdir -p uploads

# 添加版本信息标签
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.source="https://github.com/TIANLI0/LayerKit"
LABEL org.opencontainers.image.title="LayerKit"
LABEL org.opencontainers.image.description="基于 GrabCut 算法的智能图片分层 API 服务"

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./layerkit"]

