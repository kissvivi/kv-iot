<h1 align="center">Welcome to kv-iot 👋</h1>
<p>
  <img alt="Version" src="https://img.shields.io/badge/version-0.0.1-blue.svg?cacheSeconds=2592000" />
  <a href="https://github.com/kissvivi/kv-iot/blob/main/LICENSE" target="_blank">
    <img alt="License: Apache License" src="https://img.shields.io/badge/License-Apache License-yellow.svg" />
  </a>
  <a href="https://twitter.com/jobsvivi" target="_blank">
    <img alt="Twitter: jobsvivi" src="https://img.shields.io/twitter/follow/jobsvivi.svg?style=social" />
  </a>
</p>

> 云边物联网平台 go语言开发轻部署 可部署到边缘设备也可扩展成物联网平台

## 📅 功能计划（规划）
1. 产品管理（下一步做）
   1. 物模型（属性，动作，事件）
2. 设备管理（下一步做）
3. 通讯通道管理
   1. 脚本解析-js
4. Api管理
5. 可视化
6. 系统管理 （正在做）
7. 规则引擎
   1. 规则转发

### ✨ [Demo](127.0.0.1)

## Install 如何运行
### docker方式
打包docker镜像
```sh
make all
```

运行服务
```sh
make run
```

### 开发方式
```sh
go mod tidy
go mod vendor

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on

go build -o auth cmd/auth/main.go
```

## 📝项目结构理念
### 服务划分
根据每个大的功能划分服务
- 授权以及用户服务（auth）
- 设备管理服务(device)
- 数据处理/通道服务(data_)
- 规则引擎服务(rule_)


### 服务内结构划分
- data层 -> 数据操作层
- endpoint层 -> 数据暴露层
- service层 -> 业务逻辑层

## Author

👤 **jobs_vivi**

* Twitter: [@jobsvivi](https://twitter.com/jobsvivi)
* Github: [@kissvivi](https://github.com/kissvivi)

## Show your support

Give a ⭐️ if this project helped you!

## 📝 License

Copyright © 2022 [jobs_vivi](https://github.com/kissvivi).<br />
This project is [Apache License](https://github.com/kissvivi/kv-iot/blob/main/LICENSE) licensed.

***
_This README was generated with ❤️ by [readme-md-generator](https://github.com/kefranabg/readme-md-generator)_