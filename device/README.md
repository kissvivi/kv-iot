# 设备管理服务 (Device Service)

## 服务概述

设备管理服务是kv-iot物联网平台的核心组件之一，负责管理设备信息、产品信息和通信通道配置。该服务提供了完整的设备生命周期管理功能，包括设备的创建、查询、更新和删除，以及设备与产品的关联管理。

## 核心功能

### 1. 设备管理
- 设备创建与注册
- 设备信息查询（单个/批量）
- 设备删除
- 支持子设备管理

### 2. 产品管理
- 产品创建与配置
- 产品信息查询
- 产品与通信通道关联

### 3. 通道管理
- 通信通道配置与管理
- 通道编解码脚本设置
- 通道状态监控
- 支持MQTT等多种通信协议

### 4. 物模型管理
- 属性（Property）定义与管理
- 动作（Action）定义与管理
- 事件（Event）定义与管理

## 技术架构

### 分层结构

1. **数据层 (data)**
   - 数据模型定义
   - 数据访问接口
   - 数据库初始化与连接

2. **服务层 (service)**
   - 业务逻辑实现
   - 服务依赖管理
   - 数据验证与处理

3. **接口层 (endpoint)**
   - HTTP REST API
   - gRPC接口
   - 请求处理与响应格式化

### 数据模型

#### 设备 (Devices)
```go
type Devices struct {
    gorm.Model
    Name        string    // 设备名称
    DeviceNo    string    // 设备编号
    ProductID   int64     // 所属产品ID
    Desc        string    // 设备描述
    LastTime    time.Time // 最后在线时间
    SubDevice   int16     // 是否子设备
    SubDeviceID int64     // 子设备ID
    SubDeviceNo string    // 子设备号
    ProductKey  string    // 产品标识
}
```

#### 产品 (Products)
```go
type Products struct {
    gorm.Model
    Name       string // 产品名称
    Desc       string // 产品描述
    ChannelID  string // 产品通讯通道ID
    ProductKey string // 产品标识
}
```

#### 通道 (Channels)
```go
type Channels struct {
    gorm.Model
    Name       string // 通道名称
    Desc       string // 通道描述
    Encode     string // 编码脚本
    Decode     string // 解码脚本
    ScriptType string // 编解码脚本类型
}
```

## API接口

### HTTP REST API

#### 设备相关接口

| 方法 | 路径 | 功能描述 | 请求体 | 响应 |
|------|------|----------|--------|------|
| POST | /api/v1/device | 创建设备 | `{"name": "...", "device_no": "...", "product_id": 1, ...}` | `{"code": 0, "data": {...}, "msg": "添加成功"}` |
| DELETE | /api/v1/device | 删除设备 | `{"id": 1}` | `{"code": 0, "msg": "删除成功"}` |
| POST | /api/v1/device/get | 查询设备 | `{"id": 1}` 或 `{"device_no": "..."}` | `{"code": 0, "data": [...], "msg": "查询成功"}` |
| GET | /api/v1/device/all | 获取所有设备 | N/A | `{"code": 0, "data": [...], "msg": "查询成功"}` |

### gRPC接口

服务同时提供gRPC接口，用于更高效的服务间通信。gRPC服务默认监听在配置文件指定的端口上。

## 配置说明

设备管理服务通过配置文件进行配置，主要配置项包括：

### 服务配置
- 服务版本
- HTTP服务地址与端口
- gRPC服务地址与端口

### 数据库配置
- MySQL连接信息（地址、用户名、密码、数据库名）

配置文件示例（config.json）：
```json
{
  "application": {
    "deviceServer": {
      "version": "0.0.2",
      "httpserver": {
        "host": "0.0.0.0",
        "port": 8100,
        "mode": "dev",
        "readTimeout": 5,
        "writeTimeout": 5
      }
    }
  },
  "datasource":{
    "mysql":{
      "url": "172.18.61.43:3306",
      "username": "admin",
      "password": "admin",
      "dbname": "kv-iot"
    }
  }
}
```

## 启动方式

### 开发环境启动

```bash
# 进入项目根目录
go run cmd/device/main.go run
```

### 编译运行

```bash
# 编译
go build -o device cmd/device/main.go

# 运行
./device run
```

### Docker容器化部署

参考项目根目录的Dockerfile和docker-compose.yml文件。

## 服务依赖

- **数据库**: MySQL 5.7+
- **Web框架**: Gin
- **ORM**: GORM
- **命令行工具**: Cobra
- **RPC框架**: gRPC
- **消息队列**: MQTT Broker (用于设备通信)

## 与其他服务的关系

- **数据中心服务**: 设备管理服务提供的设备信息、产品配置和通道设置被数据中心服务用于设备认证、数据处理和流转
- **认证服务**: 提供API访问的认证和授权

## 通讯通道接入流程

### 1. 通道配置
1. 创建通讯通道，设置通道名称、描述和编解码脚本
2. 配置编解码脚本用于设备数据的转换和处理
3. 保存通道配置信息到数据库

### 2. 产品与通道关联
1. 创建产品时指定关联的通道ID
2. 产品配置完成后，设备可以通过该通道与平台通信

### 3. 设备数据接入流程
1. 设备通过MQTT协议连接到平台
2. 数据中心服务通过订阅的主题接收设备消息
3. 根据消息主题提取产品标识和设备编号
4. 验证设备是否已认证
5. 认证通过后，对消息进行解码处理并存储到InfluxDB
6. 设备管理服务可以查询和展示设备的最新数据

## 数据显示

### 设备数据查询
通过设备管理服务的API接口，可以查询设备的最新数据：
- 查询设备基本信息和状态
- 查询设备的属性数据
- 查询设备上报的事件

### 数据流转
1. 设备通过关联的通讯通道发送数据到数据中心
2. 数据中心处理后存储到时序数据库
3. 设备管理服务从数据库读取数据并展示
4. 支持通过API获取设备的历史数据和实时数据

### 数据可视化
设备数据可以通过前端界面进行可视化展示，包括：
- 设备状态面板
- 数据趋势图表
- 事件告警列表

## 开发指南

### 代码结构

```
device/
├── data/              # 数据模型和仓库
│   ├── repo/          # 数据访问实现
│   ├── channels.go    # 通道模型
│   ├── devices.go     # 设备模型
│   ├── products.go    # 产品模型
│   ├── kv_action.go   # 动作模型
│   ├── kv_event.go    # 事件模型
│   ├── kv_propery.go  # 属性模型
│   └── init_all_db.go # 数据库初始化
├── endpoint/          # 接口层
│   ├── http/          # HTTP接口
│   └── grpc/          # gRPC接口
└── service/           # 业务逻辑层
    ├── channels_service.go    # 通道服务
    ├── devices_service.go     # 设备服务
    ├── products_service.go    # 产品服务
    ├── kv_action_service.go   # 动作服务
    ├── kv_event_service.go    # 事件服务
    ├── kv_property_service.go # 属性服务
    └── base_service.go        # 服务依赖管理
```

### 添加新功能

1. 在data包中定义新的数据模型
2. 在repo包中实现数据访问接口
3. 在service包中实现业务逻辑
4. 在endpoint包中暴露API接口

## 故障排查

### 常见问题

1. **服务启动失败**
   - 检查配置文件是否正确
   - 确认数据库连接是否正常
   - 查看端口是否被占用

2. **API调用失败**
   - 检查请求参数是否正确
   - 确认服务是否正常运行
   - 查看日志中的错误信息

## 日志

服务启动时会输出基本配置信息，包括服务版本、监听地址、数据库连接信息等（敏感信息会被隐藏）。

## 许可证

Apache License

---

© 2022 kv-iot Team