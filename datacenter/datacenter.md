## 数据采集中心

### 设计思想
1. 可单独成为一个模块，与平台业务无关，只对接设备及数据库或其他网络接口
2. 支持命令行模式，便于部署和管理
3. 采用松耦合设计，便于扩展新的设备协议和数据处理方式
4. 支持高并发数据处理和可靠的消息传递

### 用途 
衔接设备与平台，统一数据格式，实现设备数据的采集、解析、存储和转发。

### 核心功能
- 设备连接管理（MQTT协议支持）
- 数据接收与解析
- 设备认证与授权
- 数据格式转换（统一数据模型）
- 数据存储（MySQL、InfluxDB等）
- 数据转发与路由
- 服务监控与日志记录
- 优雅启停与异常处理

### 系统架构

```
┌─────────────┐     ┌───────────────┐     ┌───────────────┐
│   设备层    │────▶│  数据采集中心  │────▶│    存储层     │
│ (MQTT设备)  │◀────│               │◀────│               │
└─────────────┘     └───────────────┘     └───────────────┘
                         │    │
                         ▼    ▼
                  ┌────────────────┐
                  │  监控与告警系统  │
                  └────────────────┘
```

## Center 核心组件

### 统一数据格式

~~~
// 结构体
// KvMsg 平台统一数据格式->对接到物模型
type KvMsg struct {
	ChannelID  string      `json:"channel_id"`  // 产品通讯通道id
	ProductKey interface{} `json:"product_key"` // 产品标识
	Property   Property    `json:"property"`
	Action     Action      `json:"action"`
	Event      Event       `json:"event"`
}

type Property struct {
	Name          string `json:"name"`            // 属性名称
	Identifier    string `json:"identifier"`      // 属性标识符
	DataType      string `json:"dataType"`        // 属性数据类型
	Unit          string `json:"unit"`            // 属性单位
	IsRw          string `json:"is_rw"`           // 是否可读写(r,w,rw)
	SubProperty   int16  `json:"sub_property"`    // 是否有子属性
	SubPropertyID string `json:"sub_property_id"` // 属性id
}

type Action struct {
	Name       string `json:"name"`       // 动作名称
	Identifier string `json:"identifier"` // 动作标识符
}

type Event struct {
	Name       string `json:"name"`       // 事件名称
	Identifier string `json:"identifier"` // 事件标识符
}
~~~

### windows打包
~~~
go build -o datacenter.exe .\main.go
~~~

### 数据监听命令
```
./datacenter.exe run
```

## 配置说明

数据中心通过 `config.yaml` 文件进行配置，主要配置项包括：

### 1. 应用配置
```yaml
application:
  appName: kv-iot-datacenter
  version: 0.0.1
```

### 2. 数据库配置
```yaml
datasource:
  mysql:
    host: localhost
    port: "3306"
    database: kv_iot
    username: root
    password: password
  influx:
    host: localhost
    port: "8086"
    database: iot_data
    username: admin
    password: admin
```

### 3. MQTT配置
```yaml
datasource:
  mqtt:
    url: 127.0.0.1
    port: "1883"
    username: ""
    password: ""
    clientId: "kv-iot-datacenter"
```

## 部署指南

### 环境要求
- Go 1.16+ 环境
- MySQL 5.7+
- MQTT Broker (如Mosquitto, EMQ X等)
- 可选：InfluxDB 用于时序数据存储

### 构建步骤

#### Windows系统
```bash
go build -o datacenter.exe ./cmd/datacenter/main.go
```

#### Linux/Mac系统
```bash
go build -o datacenter ./cmd/datacenter/main.go
```

### 运行步骤
1. 确保配置文件 `config.yaml` 已正确配置
2. 启动 MQTT Broker
3. 启动数据中心服务
   ```bash
   ./datacenter run
   ```

## 数据流程

1. 设备通过MQTT协议连接到数据中心
2. 数据中心接收设备上报的原始数据
3. 对数据进行解析和格式转换，统一为平台标准KvMsg格式
4. 进行设备认证验证
5. 将处理后的数据存储到数据库
6. 根据需要进行数据转发

## 故障排除

### 常见问题

#### 1. MQTT连接失败
- 检查MQTT服务器地址和端口是否正确
- 确认MQTT Broker是否正常运行
- 检查网络连接和防火墙设置

#### 2. 数据解析错误
- 检查设备上报的数据格式是否符合预期
- 查看日志中的错误信息，定位具体问题

#### 3. 数据库连接问题
- 验证数据库配置是否正确
- 确认数据库服务是否运行
- 检查数据库用户权限

### 日志查看
数据中心会输出详细的日志信息，可通过日志分析问题原因。关键日志包括：
- 服务启动日志
- 连接建立/断开日志
- 数据处理日志
- 错误异常日志

## 开发指南

### 扩展新协议
如需支持新的设备协议，可按照以下步骤进行：
1. 在 `datacenter/service/` 目录下创建新的协议处理模块
2. 实现数据接收和解析逻辑
3. 集成到数据中心主流程中

### 自定义数据处理
可通过修改 `Center` 组件的 `Decode` 和 `Encode` 方法来定制数据处理逻辑，满足特定业务需求。

## 安全建议

1. 始终使用强密码配置数据库和MQTT服务
2. 建议在生产环境中启用TLS/SSL加密
3. 定期更新软件版本和依赖库
4. 实施适当的访问控制策略
5. 定期备份配置和数据
