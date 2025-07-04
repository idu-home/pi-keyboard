# Pi Keyboard - 远程键盘控制服务

一个基于 Web 的远程键盘控制服务，支持实时按键输入和触控板操作。

## ✨ 主要功能

- ⌨️ 虚拟键盘：完整的键盘布局，支持字母、数字、符号和功能键
- 🖱️ 触控板支持：光标移动、点击、滚动
- 🎛️ DPI 调节：0.5x-5.0x 灵敏度调节，适应不同使用场景
- 🔌 **WebSocket 支持**：实时双向通信，低延迟响应
- 🔄 **智能降级**：WebSocket 不可用时自动切换到 HTTP API
- 📱 响应式设计：支持桌面和移动设备
- 📊 实时统计：请求统计、延迟分析、连接状态监控
- 🔧 调试功能：详细日志记录和性能分析

## 🚀 快速开始

### 编译运行

```bash
# 克隆项目
git clone <repository-url>
cd pi-keyboard

# 安装依赖
go mod tidy

# 编译
go build -o pi-keyboard .

# 运行（默认 WebSocket + HTTP 混合模式）
./pi-keyboard

# 自定义端口和配置
./pi-keyboard -port 8081 -websocket -keep-http
```

### 访问界面

- **Web 界面**: http://localhost:8081
- **WebSocket**: ws://localhost:8081/ws  
- **API 统计**: http://localhost:8081/stats

## 🔧 命令行选项

### 基本配置
```bash
-port string        服务端口 (默认: 8081)
-driver string      强制指定驱动类型 (linux_otg, macos_automation)
-output string      Linux OTG 输出文件路径
-help              显示帮助信息
```

### WebSocket 配置
```bash
-websocket         是否启用WebSocket支持 (默认: true)
-keep-http         是否保留HTTP API兼容性 (默认: true)
```

### 日志配置
```bash
-log               是否启用HTTP日志 (默认: true)
-log-output string 日志输出目标 (stdout/file/both, 默认: stdout)
-log-file string   日志文件路径 (默认: pi-keyboard.log)
-log-req-body      是否记录请求体 (默认: false)
-log-resp-body     是否记录响应体 (默认: true)
```

### 使用示例

```bash
# 仅启用 WebSocket
./pi-keyboard -websocket -keep-http=false

# 仅启用 HTTP API
./pi-keyboard -websocket=false

# 自定义日志配置
./pi-keyboard -log-output file -log-file ./logs/api.log

# 强制使用特定驱动
./pi-keyboard -driver macos_automation
```

## 📡 API 接口

### WebSocket API

**连接地址**: `ws://localhost:8081/ws`

**消息格式**:
```json
{
  "type": "message_type",
  "data": { /* 具体数据 */ },
  "timestamp": "2023-01-01T00:00:00Z",
  "request_id": "optional_request_id"
}
```

**支持的消息类型**:

#### 1. 触控板移动
```json
{
  "type": "touchpad_move",
  "data": {
    "deltaX": 10,
    "deltaY": 5,
    "dpi": 2.0
  }
}
```

#### 2. 触控板点击
```json
{
  "type": "touchpad_click", 
  "data": {
    "button": "left",  // left, right
    "type": "single"   // single, double
  }
}
```

#### 3. 触控板滚动
```json
{
  "type": "touchpad_scroll",
  "data": {
    "deltaX": 0,
    "deltaY": -3
  }
}
```

#### 4. 按键输入
```json
{
  "type": "key_press",
  "data": {
    "key": "a",
    "duration": 50
  }
}
```

#### 5. 文本输入
```json
{
  "type": "type_text",
  "data": {
    "text": "Hello World"
  }
}
```

#### 6. 心跳检测
```json
{
  "type": "ping"
}
```

### HTTP API (兼容性)

#### 按键控制
- `POST /press` - 异步按键
- `POST /press-sync` - 同步按键  
- `POST /actions` - 批量按键操作
- `POST /type` - 文本输入

#### 触控板控制
- `POST /touchpad/move` - 光标移动
- `POST /touchpad/click` - 点击操作
- `POST /touchpad/scroll` - 滚动操作

#### 系统信息
- `GET /stats` - 获取统计信息（包含 WebSocket 统计）

## 🎯 触控板功能

### 基本操作
- **滑动**: 移动光标
- **短按**: 左键点击  
- **长按**: 右键点击
- **双指滚动**: 页面滚动

### DPI 调节
- **范围**: 0.5x - 5.0x
- **预设**: 慢速(0.5x)、普通(1.0x)、快速(2.0x)、极速(5.0x)
- **实时调节**: 滑块控制

### 控制按钮
- **左键**: 主要点击
- **右键**: 右键菜单
- **双击**: 快速双击

## 🔌 WebSocket 特性

### 连接管理
- **自动重连**: 最多 5 次重连尝试
- **指数退避**: 重连间隔递增
- **心跳检测**: 54 秒间隔保持连接
- **优雅降级**: 自动切换到 HTTP API

### 性能优化
- **并发处理**: 多客户端同时连接
- **消息缓冲**: 256 字节发送缓冲区
- **超时控制**: 请求超时 5 秒
- **低延迟**: 触控板移动无需等待响应

### 连接状态
- 🟢 **已连接**: WebSocket 正常工作
- 🟡 **连接中**: 正在建立连接或重连
- 🔴 **已断开**: 连接失败
- 🔵 **HTTP 模式**: 使用 HTTP API

## 📊 统计信息

### 基础统计
- 总请求数、成功率、平均延迟
- 当前处理中的请求数
- 处理延迟 vs 网络延迟分析

### WebSocket 统计
- 活跃连接数、总连接数
- 接收/发送消息数
- 最后连接时间

### 延迟历史
- 实时延迟图表
- 最近 20 条请求记录
- 性能趋势分析

## 🛠️ 技术架构

### 后端技术
- **Go 1.21+**: 高性能服务器
- **Gorilla WebSocket**: WebSocket 实现
- **并发处理**: 无队列等待，直接并发
- **统计系统**: 实时性能监控

### 前端技术  
- **原生 JavaScript**: 无框架依赖
- **WebSocket API**: 实时通信
- **响应式设计**: 适配多设备
- **智能降级**: HTTP API 备份

### 驱动支持
- **macOS**: AppleScript + cliclick
- **Linux**: USB OTG HID 设备
- **扩展性**: 插件化驱动架构

## 🔧 开发调试

### 调试模式
- 点击"调试日志"按钮启用
- 实时日志显示
- 网络状态监控
- 性能指标追踪

### 日志级别
- **INFO**: 一般信息
- **SUCCESS**: 成功操作  
- **WARNING**: 警告信息
- **ERROR**: 错误信息

## 🚀 部署建议

### 生产环境
```bash
# 启用文件日志
./pi-keyboard -log-output file -log-file /var/log/pi-keyboard.log

# 限制 WebSocket 来源（修改代码）
CheckOrigin: func(r *http.Request) bool {
    return r.Header.Get("Origin") == "https://trusted-domain.com"
}
```

### 性能优化
- 使用反向代理 (nginx/apache)
- 启用 gzip 压缩
- 配置 SSL/TLS
- 限制并发连接数

## 📝 更新日志

### v2.0.0 (当前版本)
- ✅ 新增 WebSocket 支持
- ✅ 智能连接管理和降级
- ✅ 实时连接状态显示
- ✅ WebSocket 统计监控
- ✅ 连接模式切换功能
- ✅ 性能优化和错误处理

### v1.0.0
- ✅ 基础键盘功能
- ✅ 触控板支持
- ✅ DPI 调节
- ✅ HTTP API
- ✅ 统计系统

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License