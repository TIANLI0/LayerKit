# LayerKit

基于 GrabCut 算法的智能图片分层 API 服务

## 功能特性

- 🎨 **智能分层**：使用 OpenCV GrabCut 算法自动分离前景和背景
- 🚀 **Redis缓存**：基于图片MD5的智能缓存，避免重复计算
- 📊 **结构化数据**：返回详细的分层参数（边界框、置信度、mask等）
- 🔍 **MD5查询**：支持通过MD5哈希值快速查询历史结果
- 📝 **Zap日志**：结构化日志记录，便于调试和监控
- 🌐 **原生JS Demo**：提供开箱即用的前端演示页面
- 🤖 **自动构建**：GitHub Actions 自动构建 Docker 镜像

## 技术栈

- **后端框架**: Gin
- **图像处理**: GoCV (OpenCV Go绑定)
- **缓存**: Redis
- **日志**: Zap
- **前端**: 原生 JavaScript + HTML5 Canvas
- **CI/CD**: GitHub Actions + Docker

## 快速开始

### 方式1: 使用 Docker（推荐，最简单）

**无需安装 OpenCV！**

#### 使用预构建镜像（推荐）

```bash
# 1. 拉取最新镜像
docker pull crpi-rd21818prkp9226g.cn-shanghai.personal.cr.aliyuncs.com/hongmoai/layerkit:latest

# 2. 使用 docker-compose 启动（包含 Redis）
docker-compose up -d

# 访问
# API: http://localhost:8080
# 前端: http://localhost:8080
# 健康检查: http://localhost:8080/health
# 版本信息: http://localhost:8080/version
```

#### 本地构建镜像

```bash
# 启动服务（包含 Redis）
docker-compose up --build

# 访问
# API: http://localhost:8080
# 前端: http://localhost:8080 (访问 static/index.html)
```

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

## License

MIT


