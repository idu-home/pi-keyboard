# pi-keyboard

## 项目简介
pi-keyboard 是一个基于 Go 语言开发的跨平台键盘模拟 Web 服务，支持多种平台和技术方案：
- **Linux (树莓派等)**：通过 USB OTG 方式模拟标准 USB 键盘
- **Mac OS**：通过系统自动化工具（AppleScript）模拟键盘输入
- **Windows**：计划支持（暂未实现）

## 需求与应用场景
- **远程控制**：通过 Web API 远程控制目标设备的键盘输入
- **自动化测试**：模拟用户键盘操作进行自动化测试
- **辅助输入**：为特殊需求提供键盘输入解决方案
- **跨平台支持**：同一套 API 在不同平台上工作

## 技术方案

### Linux (树莓派) - USB OTG 模式
- 利用 Linux 的 HID Gadget（g_hid）功能，将设备模拟成 USB 键盘
- 生成 `/dev/hidg0` 设备文件，写入标准 HID 报文
- 目标设备通过 USB 连接，识别为真实键盘硬件
- 支持 Windows 和 Mac 作为目标主机

### Mac OS - 系统自动化模式
- 使用 AppleScript 的 System Events 进行键盘模拟
- 直接在本机模拟键盘输入，无需额外硬件
- 支持字符输入和特殊按键
- 需要系统辅助功能权限

## 接口设计

### 1. 单键按压接口
```http
GET /press?key=<key>&duration=<ms>
```
- `key`: 按键名称（如 `a`, `1`, `space`, `enter` 等）
- `duration`: 按键持续时间（毫秒，默认 50ms）

### 2. 批量操作接口
```http
POST /actions
Content-Type: application/json

[
  {"key": "a", "duration": 100},
  {"key": "b", "duration": 50}
]
```

### 3. 文本输入接口
```http
POST /type
Content-Type: application/json

{"text": "Hello World"}
```

### 支持的按键
- **字母**: `a-z`
- **数字**: `0-9`
- **特殊键**: `space`, `enter`, `esc`, `tab`, `backspace`, `shift`
- **Mac OS 扩展**: `cmd`, `ctrl`, `alt`, 方向键等

## 部署与使用

### 环境变量配置
- `PIKBD_PORT`: 服务端口（默认 8080）
- `PIKBD_DRIVER`: 强制指定驱动类型（`linux_otg`, `macos_automation`）
- `PIKBD_OUTPUT`: Linux OTG 输出文件路径（默认 `/dev/hidg0`）

### Linux (树莓派) 部署
1. 安装并配置 HID Gadget，确保 `/dev/hidg0` 存在且有写权限
2. 编译并运行程序：
   ```bash
   go build
   sudo ./pi-keyboard
   ```
3. 用 OTG 线将树莓派连接到目标电脑
4. 通过 HTTP 请求调用接口

### Mac OS 部署
1. 编译并运行程序：
   ```bash
   go build
   ./pi-keyboard
   ```
2. 首次运行时，系统会提示授予"辅助功能"权限
3. 在"系统偏好设置" > "安全性与隐私" > "隐私" > "辅助功能"中添加终端或应用
4. 重新启动程序

### 使用示例
```bash
# 单键按压
curl "http://localhost:8080/press?key=a&duration=100"

# 批量操作
curl -X POST http://localhost:8080/actions \
  -H "Content-Type: application/json" \
  -d '[{"key":"h","duration":50},{"key":"i","duration":50}]'

# 文本输入
curl -X POST http://localhost:8080/type \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello World"}'
```

## 平台差异与限制

### Linux OTG 模式
- ✅ 支持原子的按键操作（Press）
- ✅ 目标设备识别为硬件键盘
- ✅ 无需目标设备安装软件
- ✅ 安全的按键控制，避免卡键风险
- ❌ 需要支持 OTG 的硬件
- ❌ 需要 root 权限

### Mac OS 自动化模式
- ✅ 无需额外硬件
- ✅ 支持丰富的按键类型
- ✅ 支持快速文本输入
- ✅ 原子按键操作，确保安全
- ❌ 需要辅助功能权限
- ❌ 只能控制本机

## 架构设计

### 分层架构
```
HTTP API 层      ← 对外接口 (/press, /actions, /type)
业务逻辑层       ← 请求处理、并发控制 (Keyboard)
驱动抽象层       ← 统一接口定义 (KeyboardDriver)
平台实现层       ← 具体实现 (LinuxOTG/MacOS)
系统层          ← 操作系统接口 (HID Gadget/AppleScript)
```

### 核心接口
```go
type KeyboardDriver interface {
    Press(key string, duration time.Duration) error  // 唯一按键方法
    Type(text string) error                          // 文本输入
    IsKeySupported(key string) bool                  // 按键支持检查
    Close() error                                    // 资源清理
    GetDriverType() string                           // 驱动类型
}
```

## 安全设计

### 原子按键操作
- **移除了不安全的 KeyDown/KeyUp 接口**：避免按键卡死风险
- **统一使用 Press 原子操作**：确保每次按键都有对应的释放
- **异常处理机制**：按键操作失败时自动清理状态
- **资源清理**：程序退出时确保释放所有按键

### 风险控制原理
```go
// ❌ 危险的分离操作
KeyDown("a")  // 如果这里程序崩溃...
KeyUp("a")    // 这里永远不会执行

// ✅ 安全的原子操作  
Press("a", 100*time.Millisecond)  // 自动完成按下-释放周期
```

### 并发安全
- **全局锁**：防止多个请求同时操作键盘
- **状态锁**：保护内部按键状态的原子性
- **非阻塞检测**：避免请求堆积

## 扩展开发

### 新平台支持
1. 实现 `KeyboardDriver` 接口
2. 在 `DriverFactory` 中注册平台检测逻辑
3. 添加平台特定的按键映射

### 新功能扩展
- **组合键支持**: 扩展 Press 方法支持修饰键组合
- **宏操作**: 支持复杂的按键序列和延迟
- **配置管理**: 支持按键映射和驱动参数自定义

## 注意事项
- **Linux**: 需要 root 权限或相应的 udev 规则以写入 `/dev/hidg0`
- **Mac OS**: 需要在系统设置中授予"辅助功能"权限
- **安全**: 本工具可以模拟键盘输入，请谨慎使用，避免被恶意利用
- **兼容性**: 不同平台支持的按键可能有差异
- **原子性**: 所有按键操作都是原子的，确保不会出现卡键情况