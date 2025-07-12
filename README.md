# Pi Keyboard - 远程键盘控制

Pi Keyboard 是一个高性能的远程键盘控制工具，支持通过 Web 界面进行远程输入操作。

## 主要功能
- 🎯 多平台支持：Linux OTG、macOS 自动化
- 🌐 Web界面：响应式虚拟键盘
- 📱 移动端适配
- ⚡ 并发处理：所有请求直接并发处理，无队列等待
- 📊 实时统计与调试日志

## 快速开始

```bash
# 构建项目
go build -o pi-keyboard .
# 运行服务
./pi-keyboard
```

### 常用参数
- `-port`：服务端口 (默认: 8080)
- `-driver`：驱动类型 (linux_otg, macos_automation)
- `-output`：Linux OTG 输出文件路径

## Web界面
启动后访问 `http://localhost:8080` 使用虚拟键盘和文本输入。

## API接口

### 按键
```http
GET /press?key=a&duration=50
```

### 按键按下
```http
GET /keydown?key=a
```

### 按键抬起
```http
GET /keyup?key=a
```

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
返回：
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

## 支持的按键
- 字母：a-z
- 数字：0-9
- 功能键：enter, esc, backspace, tab, space
- 修饰键：shift, ctrl, alt, cmd
- 方向键：up, down, left, right

## 平台支持
- Linux (USB OTG HID Gadget)
- macOS (AppleScript)

## 项目结构
```
pi-keyboard/
├── main.go           # 主程序入口
├── act/              # 核心功能包
├── web/              # Web界面文件
└── test/             # 测试文件
```

## 核心接口
```go
type KeyboardDriver interface {
    Press(key string, duration time.Duration) error
    Type(text string) error
    IsKeySupported(key string) bool
    Close() error
    GetDriverType() string
}
```

## 错误处理
- 参数错误：按键不支持、参数缺失 (HTTP 400)
- 驱动错误：系统调用失败 (HTTP 500)
- 超时错误：同步接口超时 (HTTP 504)

## 调试与监控
- 实时统计与性能监控
- 前端调试日志窗口
- 错误提示

## 安全与部署
- 默认本地访问
- macOS 需辅助功能权限
- Linux 需设备文件权限
- 输入参数校验

## 贡献
欢迎提交 Issue 和 Pull Request！