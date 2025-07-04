# Pi Keyboard - 远程键盘控制

Pi Keyboard 是一个高性能的远程键盘控制工具，支持通过Web界面进行远程输入操作。

## 功能特性

- 🎯 **多平台支持**: 支持 Linux OTG、macOS 自动化
- 🌐 **Web界面**: 提供响应式的虚拟键盘界面
- 📱 **移动端适配**: 优化的移动端触摸体验
- ⚡ **高性能异步处理**: 低延迟的按键响应（17-50ms）
- 🔄 **并发处理**: 所有请求直接并发处理，无队列等待
- 📊 **实时统计**: 完整的性能监控和统计功能
- 🔍 **调试支持**: 详细的日志记录和调试功能

## 快速开始

### 构建和运行

```bash
# 构建项目
go build -o pi-keyboard .

# 运行服务
./pi-keyboard
```

### 命令行参数

#### 基本参数
- `-port`: 服务端口 (默认: 8080)
- `-driver`: 强制指定驱动类型 (linux_otg, macos_automation)
- `-output`: Linux OTG 输出文件路径
- `-help`: 显示帮助信息

#### 日志参数
- `-log`: 是否启用HTTP日志 (默认: true)
- `-log-output`: 日志输出目标 (stdout/file/both, 默认: stdout)
- `-log-file`: 日志文件路径 (默认: pi-keyboard.log)
- `-log-req-body`: 是否记录请求体 (默认: false)
- `-log-resp-body`: 是否记录响应体 (默认: true)

> **注意**: 日志记录的API已硬编码为核心功能接口: `/press`, `/press-sync`, `/actions`, `/type`  
> `/stats` 接口不记录日志以避免过多日志输出

### 使用示例

#### 基本使用
```bash
# 使用默认配置启动
./pi-keyboard

# 显示帮助信息
./pi-keyboard -help

# 使用自定义端口
./pi-keyboard -port 8081

# 强制使用特定驱动
./pi-keyboard -driver macos_automation

# Linux OTG配置
./pi-keyboard -driver linux_otg -output /dev/hidg0
```

#### 日志配置示例
```bash
# 将日志输出到文件
./pi-keyboard -log-output file -log-file ./logs/api.log

# 同时输出到控制台和文件
./pi-keyboard -log-output both -log-file ./logs/api.log

# 关闭HTTP日志
./pi-keyboard -log=false

# 详细调试模式
./pi-keyboard -log-req-body -log-resp-body

# 组合配置
./pi-keyboard -port 8081 -log-output file -log-file ./logs/api.log -driver macos_automation
```

更多配置示例请运行: `./scripts/log-examples.sh`

## Web界面使用

启动服务后，打开浏览器访问 `http://localhost:8080` 即可使用Web界面。

### 主要功能

1. **文本输入**: 在文本框中输入内容，点击"发送文本"按钮
2. **虚拟键盘**: 点击屏幕键盘上的按键进行输入
3. **实时统计**: 显示请求统计、成功率、平均延迟等信息
4. **调试日志**: 详细的调试信息和网络请求日志

### 移动端使用

- 界面已针对移动设备优化
- 支持触摸操作
- 防止意外缩放和选择
- 响应式布局适配不同屏幕尺寸

## API接口

### 异步按键接口 (推荐)

```http
GET /press?key=a&duration=50
```

立即返回 `processing`，请求在后台并发处理。

### 同步按键接口

```http
GET /press-sync?key=a&duration=50
```

等待按键执行完成后返回结果。

### 批量操作

```http
POST /actions
Content-Type: application/json

[
  {"key": "ctrl", "duration": 50},
  {"key": "c", "duration": 50}
]
```

### 文本输入

```http
POST /type
Content-Type: application/json

{"text": "Hello World"}
```

### 统计信息

```http
GET /stats
```

返回实时的性能统计信息：

```json
{
  "total_requests": 1234,
  "success_requests": 1200,
  "failed_requests": 34,
  "rejected_requests": 0,
  "average_latency_ms": 25,
  "success_rate": 97.2,
  "currently_processing": 2,
  "last_request_time": "2023-12-01T10:30:00Z",
  "latency_breakdown": {
    "process_ms": 20,
    "network_ms": 5
  },
  "latency_history": [ /* ... */ ]
}
```

## 系统架构

### 架构概览

Pi Keyboard 采用分层架构设计，包含前端界面、HTTP服务器、异步处理服务、驱动层和系统层。

📊 **详细架构图**: [doc/architecture-diagram.md](doc/architecture-diagram.md)

### 核心组件

#### 1. Web前端 (浏览器)
- **文件**: `web/index.html`, `web/script.js`, `web/style.css`
- **功能**: 提供虚拟键盘界面，处理用户交互
- **特点**: 响应式设计，移动端优化，异步处理

#### 2. HTTP服务器 (main.go)
- **功能**: HTTP路由、静态文件服务、API接口
- **端口**: 默认8080，可通过 `PIKBD_PORT` 环境变量配置

#### 3. 键盘服务 (keyboard.go)
- **功能**: 并发请求处理、统计收集
- **并发**: 使用 goroutine 实现高并发处理

#### 4. 驱动层 (act/)
- **工厂模式**: 自动检测平台并创建合适的驱动
- **统一接口**: 所有驱动实现 `KeyboardDriver` 接口
- **支持平台**: macOS (AppleScript)、Linux (USB HID Gadget)

### 异步处理机制

🔄 **完整时序图**: [doc/sequence-diagram.md](doc/sequence-diagram.md)

#### 处理流程
1. **请求接收**: HTTP处理器接收请求
2. **参数验证**: 验证按键和持续时间
3. **并发处理**: 为每个请求启动一个独立的 goroutine
4. **立即响应**: 返回 `processing` 状态
5. **后台执行**: goroutine 调用驱动执行按键
6. **统计更新**: 更新性能统计信息

#### 并发安全
- **原子操作**: 使用 `sync/atomic` 处理计数器
- **互斥锁**: 使用 `sync.RWMutex` 保护共享数据
- **通道通信**: 使用 channel 进行 goroutine 间通信

### 驱动选择机制

#### 自动检测流程
1. **平台检测**: 使用 `runtime.GOOS` 检测操作系统
2. **驱动创建**: 根据平台自动创建合适的驱动
3. **可用性检查**: 验证驱动是否可用

#### macOS 驱动
- 使用 AppleScript 自动化
- 字符按键: 瞬时输入 (`keystroke`)
- 特殊按键: 按下-等待-释放 (`key down` + `key up`)

#### Linux 驱动
- 使用 USB HID Gadget 模式
- 直接写入设备文件 `/dev/hidg0`
- 支持真实的按键持续时间

## 支持的按键

- **字母**: a-z
- **数字**: 0-9
- **功能键**: enter, esc, backspace, tab, space
- **修饰键**: shift, ctrl, alt, cmd
- **方向键**: up, down, left, right

## 平台支持

### Linux (树莓派)

使用 USB OTG HID Gadget 模式，需要配置：

```bash
# 启用 HID Gadget
echo 'dtoverlay=dwc2' >> /boot/config.txt
echo 'modules-load=dwc2,libcomposite' >> /boot/cmdline.txt

# 重启后创建 HID 设备
sudo modprobe libcomposite
sudo mkdir -p /sys/kernel/config/usb_gadget/keyboard
# ... 更多配置步骤
```

### macOS

使用 AppleScript 自动化，需要：

1. 在"系统偏好设置" > "安全性与隐私" > "隐私"中
2. 为终端应用添加"辅助功能"权限

### Windows

暂未实现，计划中。

## 性能优化

### 延迟优化
- **异步处理**: 避免阻塞HTTP请求
- **队列缓冲**: 100个请求的缓冲队列
- **原子操作**: 减少锁竞争
- **预编译脚本**: macOS 使用预编译的 AppleScript

### 性能指标
- **响应时间**: 17-50ms（17倍性能提升）
- **队列容量**: 100个请求缓冲
- **成功率**: 97%+ 的成功率
- **并发能力**: 支持快速连续操作

## 开发

### 项目结构

```
pi-keyboard/
├── main.go              # 主程序入口和HTTP服务器
├── README.md            # 项目说明文档
├── doc/                 # 文档目录
│   ├── sequence-diagram.md     # 时序图
│   └── architecture-diagram.md # 架构图
├── act/                 # 核心功能包
│   ├── driver.go        # 驱动接口定义
│   ├── factory.go       # 驱动工厂（自动检测平台）
│   ├── keyboard.go      # 键盘服务（异步处理）
│   ├── linux_otg_driver.go    # Linux OTG驱动
│   └── macos_driver.go  # macOS驱动（AppleScript）
├── web/                 # Web界面文件
│   ├── index.html       # 主页面（响应式设计）
│   ├── style.css        # 样式文件（移动端优化）
│   └── script.js        # JavaScript逻辑（异步处理）
└── test/                # 测试文件
    └── keyboard_demo.go # 驱动测试程序
```

### 核心接口

```go
type KeyboardDriver interface {
    Press(key string, duration time.Duration) error
    Type(text string) error
    IsKeySupported(key string) bool
    Close() error
    GetDriverType() string
}
```

### 添加新驱动

1. 实现 `KeyboardDriver` 接口
2. 在 `factory.go` 中注册新驱动
3. 添加驱动类型常量
4. 更新平台检测逻辑

### 添加新接口

1. 在 `keyboard.go` 中添加处理器
2. 在 `main.go` 中注册路由
3. 更新前端调用逻辑

## 错误处理

### 错误类型
- **参数错误**: 按键不支持、参数缺失 (HTTP 400)
- **队列满**: 服务器繁忙 (HTTP 429)
- **驱动错误**: 系统调用失败 (HTTP 500)
- **超时错误**: 同步接口超时 (HTTP 504)

### 优雅降级
- 队列满时拒绝新请求
- 错误恢复和状态清理
- 详细的错误日志记录

## 调试和监控

### 日志记录
- **微秒级时间戳**: 精确的时间记录
- **请求跟踪**: 完整的请求生命周期
- **性能统计**: 详细的性能指标

### 前端调试
- **调试日志窗口**: 详细的网络请求日志
- **实时统计**: 性能监控界面
- **错误提示**: 用户友好的错误信息

## 安全考虑

### 访问控制
- **本地访问**: 默认绑定本地接口
- **权限要求**: macOS 需要辅助功能权限
- **设备权限**: Linux 需要设备文件访问权限

### 输入验证
- **按键验证**: 检查按键是否支持
- **参数验证**: 验证输入参数格式
- **长度限制**: 限制文本输入长度

## 部署建议

### 开发环境
- 使用默认配置快速启动
- 启用调试日志进行问题排查
- 使用统计接口监控性能

### 生产环境
- 配置适当的端口和权限
- 监控队列状态和性能指标
- 定期检查错误日志

### 移动端部署
- 确保网络连接稳定
- 使用 IP 地址访问服务
- 优化触摸体验和响应速度

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

### 开发指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 发起 Pull Request

### 报告问题

- 详细描述问题
- 提供复现步骤
- 包含系统信息和日志