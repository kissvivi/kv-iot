<h1 align="center">🎊🥂 Welcome to kv-iot 👋</h1>
<p>
  <img alt="Version" src="https://img.shields.io/badge/version-0.0.1-blue.svg?cacheSeconds=2592000" />
  <a href="https://github.com/kissvivi/kv-iot/blob/main/LICENSE" target="_blank">
    <img alt="License: Apache License" src="https://img.shields.io/badge/License-Apache License-yellow.svg" />
  </a>

[//]: # (  <a href="https://twitter.com/jobsvivi" target="_blank">)

[//]: # (    <img alt="Twitter: jobsvivi" src="https://img.shields.io/twitter/follow/jobsvivi.svg?style=social" />)

[//]: # (  </a>)
</p>

> 云边物联网平台 go语言开发轻部署 可部署到边缘设备也可扩展成物联网平台

## 📅 功能计划（规划）
1. 产品管理
   1. 物模型（属性，动作，事件）
2. 设备管理
3. 通讯通道管理（数据中心）
   1. 脚本解析-js
4. Api管理
5. 可视化
6. 系统管理
7. 规则引擎
   1. 规则转发
8. 通讯工具
   1. mqtt broker
   2. tcp server client
   3. modbusTcp server client
   4. lora server
   5. http websocket

## 开发日志
* 2023/03-2023/04 着重开发通用通讯通道，即适用平台的MQTT通讯，目标可正常接入平台

### ✨ [Demo 体验地址 暂无](127.0.0.1)

### 前端开源地址
* https://github.com/kissvivi/kv-iot-web.git
### ✨ InfluxDB数据接入

## 🪄 Install 如何运行

### 开发方式运行
```sh
go mod tidy
go mod vendor

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on

go build -o auth cmd/auth/main.go
```

### docker方式运行
打包docker镜像
```sh
go mod tidy
go mod vendor
make dockers
```

运行服务
```sh
make run
```

## 📝项目结构理念
### 服务划分
根据每个大的功能划分服务
- 授权以及用户服务（auth）
- 设备管理服务(device)
- 数据处理/通道服务（数据中心）(data_)
- 规则引擎服务(rule_)


### 服务内结构划分
- data层 -> 数据操作层
- endpoint层 -> 数据暴露层
- service层 -> 业务逻辑层


### 关于我们
* 本物联网平台是综合调研阿里云物联平台/移动ONE-NET物联平台/IotDc3/JetLinks/ChirpStack等
* 以及工业物联网实际场景综合考虑设计，目前项目属于起步状态，远没有达到生产环境标准
* 希望更多人能一起交流物联网开发技术
* QQ交流群：442183314


## Author

👤 **jobs_vivi**

* Twitter: [@jobsvivi](https://twitter.com/jobsvivi)
* Github: [@kissvivi](https://github.com/kissvivi)

## Show your support

Give a ⭐️ if this project helped you!

## Thanks 感谢赞助
<a href="https://jb.gg/OpenSourceSupport">
<img  src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" width="10%">
</a>

## 📝 License

Copyright © 2022 [jobs_vivi](https://github.com/kissvivi).<br />
This project is [Apache License](https://github.com/kissvivi/kv-iot/blob/main/LICENSE) licensed.

***
_This README was generated with ❤️ by [readme-md-generator](https://github.com/kefranabg/readme-md-generator)_