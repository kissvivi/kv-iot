<h1 align="center">🎊🥂 Welcome to kv-iot 👋</h1>
<p>
  <img alt="Version" src="https://img.shields.io/badge/version-0.0.1-blue.svg?cacheSeconds=2592000" />
  <a href="https://github.com/kissvivi/kv-iot/blob/main/LICENSE" target="_blank">
    <img alt="License: Apache License" src="https://img.shields.io/badge/License-Apache License-yellow.svg" />
  </a>
  <a href="https://github.com/kissvivi/kv-iot/issues" target="_blank">
    <img alt="Issues" src="https://img.shields.io/github/issues/kissvivi/kv-iot.svg" />
  </a>
  <a href="https://github.com/kissvivi/kv-iot/network" target="_blank">
    <img alt="Forks" src="https://img.shields.io/github/forks/kissvivi/kv-iot.svg" />
  </a>
  <a href="https://github.com/kissvivi/kv-iot/stargazers" target="_blank">
    <img alt="Stars" src="https://img.shields.io/github/stars/kissvivi/kv-iot.svg" />
  </a>
</p>

> 云边一体化物联网平台 - Go语言开发的轻量级部署解决方案，既可部署到边缘设备也可扩展成企业级物联网平台。

## 📅 功能特性

### 已实现功能

1. **授权管理系统**
   - 用户认证与授权
   - 基于RBAC的权限管理
   - JWT令牌认证
2. **设备管理服务**
   - 设备信息管理
   - 产品管理
   - 通道配置管理
3. **数据中心服务**
   - MQTT数据接入与处理
   - 统一数据格式转换
   - 数据持久化（MySQL、InfluxDB）
4. **基础通讯能力**
   - MQTT客户端封装
   - TCP基础框架
5. **配置管理**
   - YAML配置文件解析
   - 统一配置管理

### 规划功能

1. **增强产品管理**
   - 完善物模型（属性，动作，事件）
   - 产品模板
2. **规则引擎**
   - 规则转发
   - 数据流转
   - 任务流编排
3. **高级通讯协议**
   - ModbusTCP
   - LoRaWAN
   - HTTP/WebSocket服务
4. **API管理**
   - RESTful API网关
   - API文档与测试
5. **可视化监控**
   - 设备状态监控
   - 数据可视化面板
6. **系统管理**
   - 日志管理
   - 系统监控
   - 告警通知

## 开发日志

* 2023/03-2023/04 开发通用通讯通道，实现MQTT通讯功能，确保设备可正常接入平台
* 2025/10 完善数据中心服务，增强MQTT连接配置管理和参数解析功能
* 2025/10 优化系统架构，改进配置信息输出显示，增强日志记录和错误处理能力
* 2025/10 完善项目文档，提供更全面的部署和使用指南
* 持续开发中，欢迎志同道合的朋友一起讨论物联网技术和贡献代码

### ✨ [Demo 体验地址 暂无](127.0.0.1)

### ✨ [kv-iot 文档地址/问题记录](http://doc.kv-iot.cn/)

### 前端开源地址

* https://github.com/kissvivi/kv-iot-web.git

### ✨ InfluxDB数据接入

## 🪄 快速开始

### 环境要求

* Go 1.22+
* MySQL 8.0+
* InfluxDB 1.x
* MQTT Broker (如Mosquitto,emqx)
* Docker (可选，用于容器化部署)

### 开发方式运行

1. **配置文件准备**

   ```sh
   # 复制并修改配置文件
   cp config.yaml.example config.yaml
   # 根据实际环境修改配置项
   ```
2. **安装依赖**

   ```sh
   go mod tidy
   go mod vendor
   ```
3. **运行认证服务**

   ```sh
   go build -o auth cmd/auth/main.go
   ./auth
   ```
4. **运行设备管理服务**

   ```sh
   go build -o device cmd/device/main.go
   ./device
   ```
5. **运行数据中心服务**

   ```sh
   go build -o datacenter cmd/datacenter/main.go
   ./datacenter run
   ```

### Docker容器化部署

1. **构建Docker镜像**

   ```sh
   go mod tidy
   go mod vendor
   make dockers
   ```
2. **使用Docker Compose运行**

   ```sh
   cd docker
   docker-compose up -d
   ```

### 配置说明

项目主要通过 `config.yaml`文件进行配置，包括：

- 数据库连接参数
- MQTT连接配置
- 服务端口和其他运行时参数

详细配置说明请参考配置文件注释或文档。

## 📝 系统架构

### 服务架构

![系统架构](https://i.imgur.com/system_architecture.png) *（示意图，请替换为实际架构图）*

### 服务划分

根据功能模块化设计，将系统分为多个独立服务：

- **认证服务（auth）**：负责用户认证、授权和权限管理
- **设备管理服务（device）**：管理设备、产品和通信通道配置
- **数据中心服务（datacenter）**：处理设备数据的接入、转换和存储
- **规则引擎服务（rule）**：实现数据处理规则和业务逻辑编排

### 分层架构

每个服务内部采用清晰的分层结构：

- **data层**：数据模型定义和数据访问操作
  - 包含数据实体、仓储模式实现
  - 数据库连接和事务管理
- **service层**：核心业务逻辑实现
  - 业务规则处理
  - 数据验证和转换
- **endpoint层**：服务接口暴露
  - REST API
  - gRPC接口
  - 消息订阅接口

### 数据流程

1. 设备通过MQTT协议连接到平台
2. 数据中心服务接收和解析设备消息
3. 统一格式转换后存储到数据库
4. 设备管理服务提供设备状态和配置管理
5. 规则引擎根据配置的规则处理数据流转

### 关于项目

* kv-iot是一个开源的云边一体化物联网平台，综合参考了阿里云物联网平台、中国移动ONE-NET、IotDc3、JetLinks、ChirpStack等优秀项目的设计理念
* 项目采用轻量级设计，可灵活部署于边缘设备或云服务器，满足不同规模的物联网应用需求
* 我们致力于打造一个简单易用、功能强大、性能稳定的物联网开发框架

### 交流与支持

* QQ交流群：442183314
* 欢迎提交Issue和Pull Request
* 欢迎分享您的使用案例和改进建议

## Author

👤 **jobs_vivi**

* Twitter: [@jobsvivi](https://twitter.com/jobsvivi)
* Github: [@kissvivi](https://github.com/kissvivi)

## Show your support

Give a ⭐️ if this project helped you!

## Thanks 感谢支持

### 感谢赞助

<a href="https://jb.gg/OpenSourceSupport">
<img  src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" width="10%">
</a>

### 感谢Trae AI

特别感谢Trae AI提供的强大开发辅助功能，在代码完善、文档撰写和系统优化方面提供了重要支持，大幅提升了开发效率和代码质量。

### 感谢社区贡献

感谢所有关注和支持本项目的开发者，欢迎提交Issue和Pull Request，共同建设更好的物联网平台。

## 📝 License

Copyright © 2022 [jobs_vivi](https://github.com/kissvivi).`<br />`
This project is [Apache License](https://github.com/kissvivi/kv-iot/blob/main/LICENSE) licensed.

---

_This README was generated with ❤️ by [readme-md-generator](https://github.com/kefranabg/readme-md-generator)_
