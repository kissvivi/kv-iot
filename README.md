# kv-iot
云边物联网平台 go语言开发轻部署 可部署到边缘设备也可扩展成物联网平台

## 功能计划（规划）
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

## 服务划分
- 授权以及用户服务（auth）
- 设备管理服务(device)
- 数据处理/通道服务(data_)
- 规则引擎服务(rule_)

## 如何运行
### docker方式
打包docker镜像
``make all``

运行服务
``make run``

### 开发方式

`go mod tidy
go mod vendor`

`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on`

`go build -o auth cmd/auth/main.go`
