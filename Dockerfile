FROM gocv/opencv:4.12.0

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN go build -o layerkit .

# 创建上传目录
RUN mkdir -p uploads

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./layerkit"]
