# LayerKit

基于 GrabCut 算法的智能图片分层 API 服务

## 功能特性

- 🎨 **智能分层**：使用 OpenCV GrabCut 算法自动分离前景和背景
- 🚀 **Redis缓存**：基于图片MD5的智能缓存，避免重复计算
- 📊 **结构化数据**：返回详细的分层参数（边界框、置信度、mask等）
- 🔍 **MD5查询**：支持通过MD5哈希值快速查询历史结果
- 📝 **Zap日志**：结构化日志记录，便于调试和监控
- 🌐 **原生JS Demo**：提供开箱即用的前端演示页面

## 技术栈

- **后端框架**: Gin
- **图像处理**: GoCV (OpenCV Go绑定)
- **缓存**: Redis
- **日志**: Zap
- **前端**: 原生 JavaScript + HTML5 Canvas

## 快速开始

### 方式1: 使用 Docker（推荐，最简单）

**无需安装 OpenCV！**

```bash
# 启动服务（包含 Redis）
docker-compose up --build

# 访问
# API: http://localhost:8080
# 前端: http://localhost:8080 (访问 static/index.html)
```

### 方式2: 本地运行（MinGW64/MSYS2）

#### 前置要求

- Go 1.21+
- Redis
- OpenCV 4.x（MinGW64）

#### 如果已安装 MSYS2/MinGW64 OpenCV

如果你已经通过 MSYS2 安装了 OpenCV：
```bash
pacman -S mingw-w64-x86_64-opencv
```

**快速启动（一键运行）**：
```cmd
# 自动配置环境变量并启动
run.cmd
```

**或使用配置脚本**：
```cmd
# 带验证的配置脚本
setup-mingw64-env.cmd
```

**手动配置**（如果脚本无法运行）：
```cmd
# 设置环境变量（替换为你的实际路径）
set CGO_ENABLED=1
set CGO_CPPFLAGS=-ID:/msys2/mingw64/include/opencv4
set CGO_LDFLAGS=-LD:/msys2/mingw64/lib -lopencv_core -lopencv_imgproc -lopencv_imgcodecs -lopencv_videoio -lopencv_highgui -lopencv_video -lopencv_features2d -lopencv_calib3d -lopencv_objdetect
set PATH=D:\msys2\mingw64\bin;%PATH%

# 运行服务
go run .
```

验证 OpenCV 配置：
```bash
# 在 MSYS2 终端中
pkg-config --cflags opencv4
pkg-config --libs opencv4
```

#### 从零安装 OpenCV (Windows)

```cmd
# 运行安装脚本（会引导你完成安装）
setup-opencv.cmd
```

或手动安装：

```bash
# 使用 Chocolatey（需要管理员权限）
choco install opencv mingw -y

# 配置环境变量（重要！）
setx CGO_CPPFLAGS "-IC:/opencv/build/install/include" /M
setx CGO_LDFLAGS "-LC:/opencv/build/install/x64/mingw/lib" /M
```

详细安装指南请查看 [OPENCV_SETUP.md](OPENCV_SETUP.md)

#### 安装 Go 依赖

```bash
go mod download
```

#### 配置文件

复制并修改 `config.yaml`：

```yaml
server:
  port: ":8080"
  mode: "debug"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  ttl: 24h

upload:
  max_size: 10485760  # 10MB
  upload_dir: "./uploads"
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "image/jpg"

grabcut:
  iterations: 5
  border_size: 10
```

#### 启动 Redis

```bash
# Windows (使用 Docker)
docker run -d -p 6379:6379 redis:7-alpine

# 或使用本地 Redis
redis-server
```

#### 运行服务

```bash
# 使用快速启动脚本（已配置环境变量）
run.cmd

# 或手动运行（需先配置环境变量）
go run .
```

访问 http://localhost:8080

服务将在 `http://localhost:8080` 启动

### 访问演示页面

打开浏览器访问: `http://localhost:8080`

## 前端使用指南

### 基础用法：上传图片获取掩码

```javascript
// 1. 上传图片并获取分层结果
async function uploadImage(file) {
  const formData = new FormData();
  formData.append('image', file);
  
  const response = await fetch('http://localhost:8080/api/v1/upload', {
    method: 'POST',
    body: formData
  });
  
  const result = await response.json();
  return result.data; // { md5, width, height, layers: [...] }
}

// 2. 解析 Base64 掩码为图片
function base64ToImage(base64) {
  return new Promise((resolve) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.src = 'data:image/png;base64,' + base64;
  });
}

// 3. 使用示例
const file = document.querySelector('input[type="file"]').files[0];
const layerData = await uploadImage(file);

// 获取前景掩码
const foregroundLayer = layerData.layers.find(l => l.type === 'foreground');
const maskImage = await base64ToImage(foregroundLayer.mask);
```

### 应用掩码：前景提取

```javascript
async function extractForeground(originalImage, layerData) {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  
  canvas.width = layerData.width;
  canvas.height = layerData.height;
  
  // 绘制原图
  ctx.drawImage(originalImage, 0, 0);
  const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
  
  // 获取前景掩码
  const foregroundLayer = layerData.layers.find(l => l.type === 'foreground');
  const maskImg = await base64ToImage(foregroundLayer.mask);
  
  // 绘制掩码到临时 canvas
  const maskCanvas = document.createElement('canvas');
  const maskCtx = maskCanvas.getContext('2d');
  maskCanvas.width = canvas.width;
  maskCanvas.height = canvas.height;
  maskCtx.drawImage(maskImg, 0, 0);
  const maskData = maskCtx.getImageData(0, 0, canvas.width, canvas.height);
  
  // 应用掩码（保留前景，背景透明）
  for (let i = 0; i < imageData.data.length; i += 4) {
    const maskValue = maskData.data[i]; // 掩码为灰度图，只需读取 R 通道
    if (maskValue < 128) {
      // 背景区域：设为透明
      imageData.data[i + 3] = 0;
    }
  }
  
  ctx.putImageData(imageData, 0, 0);
  return canvas;
}

// 使用示例
const originalImg = new Image();
originalImg.src = URL.createObjectURL(file);
await new Promise(resolve => originalImg.onload = resolve);

const foregroundCanvas = await extractForeground(originalImg, layerData);
document.body.appendChild(foregroundCanvas);
```

### 高级用法：更换背景

```javascript
async function replaceBackground(originalImage, layerData, newBackgroundColor = '#00ff00') {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  
  canvas.width = layerData.width;
  canvas.height = layerData.height;
  
  // 1. 填充新背景
  ctx.fillStyle = newBackgroundColor;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  
  // 2. 获取前景掩码
  const foregroundLayer = layerData.layers.find(l => l.type === 'foreground');
  const maskImg = await base64ToImage(foregroundLayer.mask);
  
  // 3. 创建临时 canvas 处理原图
  const tempCanvas = document.createElement('canvas');
  const tempCtx = tempCanvas.getContext('2d');
  tempCanvas.width = canvas.width;
  tempCanvas.height = canvas.height;
  
  tempCtx.drawImage(originalImage, 0, 0);
  const imageData = tempCtx.getImageData(0, 0, canvas.width, canvas.height);
  
  // 4. 创建掩码数据
  const maskCanvas = document.createElement('canvas');
  const maskCtx = maskCanvas.getContext('2d');
  maskCanvas.width = canvas.width;
  maskCanvas.height = canvas.height;
  maskCtx.drawImage(maskImg, 0, 0);
  const maskData = maskCtx.getImageData(0, 0, canvas.width, canvas.height);
  
  // 5. 应用掩码：背景透明
  for (let i = 0; i < imageData.data.length; i += 4) {
    if (maskData.data[i] < 128) {
      imageData.data[i + 3] = 0; // 背景透明
    }
  }
  
  tempCtx.putImageData(imageData, 0, 0);
  
  // 6. 合成：新背景 + 前景
  ctx.drawImage(tempCanvas, 0, 0);
  
  return canvas;
}

// 使用示例：绿幕效果
const greenScreenCanvas = await replaceBackground(originalImg, layerData, '#00ff00');
document.body.appendChild(greenScreenCanvas);

// 使用图片作为背景
async function replaceWithImageBackground(originalImage, layerData, backgroundImage) {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  
  canvas.width = layerData.width;
  canvas.height = layerData.height;
  
  // 绘制背景图（拉伸填充）
  ctx.drawImage(backgroundImage, 0, 0, canvas.width, canvas.height);
  
  // 提取前景（透明背景）
  const foregroundCanvas = await extractForeground(originalImage, layerData);
  
  // 合成
  ctx.drawImage(foregroundCanvas, 0, 0);
  
  return canvas;
}
```

### 实用功能：裁剪前景边界框

```javascript
function cropToBoundingBox(canvas, boundingBox) {
  const { x, y, width, height } = boundingBox;
  
  const croppedCanvas = document.createElement('canvas');
  const ctx = croppedCanvas.getContext('2d');
  
  croppedCanvas.width = width;
  croppedCanvas.height = height;
  
  ctx.drawImage(canvas, x, y, width, height, 0, 0, width, height);
  
  return croppedCanvas;
}

// 使用示例：只保留前景主体（裁剪）
const foregroundLayer = layerData.layers.find(l => l.type === 'foreground');
const foregroundCanvas = await extractForeground(originalImg, layerData);
const croppedCanvas = cropToBoundingBox(foregroundCanvas, foregroundLayer.bounding_box);

document.body.appendChild(croppedCanvas);
```

## API 接口

### 1. 上传图片并分层

**POST** `/api/v1/upload`

- **Content-Type**: `multipart/form-data`
- **参数**: 
  - `image`: 图片文件 (JPEG/PNG, 最大10MB)

**响应示例**:
```json
{
  "success": true,
  "message": "处理成功",
  "data": {
    "md5": "abc123...",
    "width": 1920,
    "height": 1080,
    "timestamp": 1699401234,
    "layers": [
      {
        "id": 1,
        "type": "foreground",
        "bounding_box": {
          "x": 100,
          "y": 200,
          "width": 800,
          "height": 600
        },
        "mask": "base64编码的PNG图片...",
        "confidence": 0.85
      },
      {
        "id": 2,
        "type": "background",
        "bounding_box": {
          "x": 0,
          "y": 0,
          "width": 1920,
          "height": 1080
        },
        "mask": "base64编码的PNG图片...",
        "confidence": 0.15
      }
    ]
  }
}
```

### 2. 通过MD5查询分层结果

**GET** `/api/v1/layer/:md5`

**响应**: 与上传接口相同

## 项目结构

```
LayerKit/
├── config/              # 配置管理
│   └── config.go
├── handler/             # HTTP处理器
│   └── upload.go
├── middleware/          # 中间件
│   ├── cors.go
│   └── logger.go
├── model/               # 数据模型
│   └── layer.go
├── service/             # 业务逻辑
│   ├── grabcut.go
│   └── redis.go
├── static/              # 静态文件
│   └── index.html
├── utils/               # 工具函数
│   ├── hash.go
│   ├── id.go
│   └── logger.go
├── uploads/             # 上传文件目录
├── config.yaml          # 配置文件
├── docker-compose.yml   # Docker Compose 配置
├── Dockerfile           # Docker 镜像构建
├── go.mod
├── main.go
└── README.md
```

## 配置说明

配置文件 `config.yaml`：

```yaml
server:
  port: ":8080"              # 服务端口
  mode: "debug"              # 运行模式: debug/release
  read_timeout: 10s          # 读取超时
  write_timeout: 10s         # 写入超时

redis:
  addr: "localhost:6379"     # Redis 地址
  password: ""               # Redis 密码
  db: 0                      # Redis 数据库
  ttl: 24h                   # 缓存过期时间

upload:
  max_size: 10485760         # 最大文件大小 (10MB)
  upload_dir: "./uploads"    # 上传目录
  allowed_types:             # 允许的文件类型
    - "image/jpeg"
    - "image/png"
    - "image/jpg"

grabcut:
  iterations: 5              # GrabCut 迭代次数 (1-15)
  border_size: 10            # 边界大小 (像素)
```

## GrabCut 算法说明

GrabCut 是一种基于图割的前景提取算法，本项目使用以下策略：

1. 自动设置初始矩形区域（图片边缘向内缩进）
2. 执行指定次数的迭代优化
3. 生成前景和背景的二值mask
4. 计算边界框和置信度

**参数调优**：
- `iterations`: 5-10 适合大多数场景，更高值提升精度但增加耗时
- `border_size`: 10-30 像素，根据图片主体位置调整

## 故障排除

遇到问题？查看 **[故障排除指南](TROUBLESHOOTING.md)**

常见问题：
- OpenCV 编译错误 → [OPENCV_SETUP.md](OPENCV_SETUP.md)
- Redis 连接失败 → 使用 `docker run -d -p 6379:6379 redis:7-alpine`
- 端口被占用 → 修改 `config.yaml` 中的端口
- CORS 错误 → 检查 `middleware/cors.go`

## 开发建议

- 根据实际需求调整 GrabCut 参数（迭代次数、边界大小）
- 可扩展支持更多图像处理算法（如语义分割、实例分割）
- 添加用户认证和权限管理
- 实现图片存储服务（如 OSS、S3）集成
- 添加批量处理功能
- 支持自定义分层参数（手动标注前景区域）
- 添加 WebSocket 支持实时进度推送

## 性能优化

- 使用 Redis 缓存避免重复计算
- 通过 MD5 去重，相同图片直接返回缓存
- 支持配置 GrabCut 迭代次数平衡精度和速度
- 可选：添加图片预处理（缩放、压缩）

## License

MIT

