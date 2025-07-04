# Pi Keyboard - 远程键盘控制

Pi Keyboard 是一个远程键盘控制工具，支持通过Web界面进行远程输入操作。

## 功能特性

- 🎯 **多平台支持**: 支持 Linux OTG、macOS 自动化
- 🌐 **Web界面**: 提供响应式的虚拟键盘界面
- 📱 **移动端适配**: 优化的移动端触摸体验
- ⚡ **实时响应**: 低延迟的按键响应
- 🔒 **安全控制**: 防止并发操作的安全机制

## 快速开始

### 构建和运行

```bash
# 构建项目
go build -o pi-keyboard .

# 运行服务
./pi-keyboard
```

### 环境变量配置

- `PIKBD_PORT`: 服务端口 (默认: 8080)
- `PIKBD_DRIVER`: 强制指定驱动类型 (linux_otg, macos_automation)
- `PIKBD_OUTPUT`: Linux OTG 输出文件路径 (默认: /dev/hidg0)

### 使用示例

```bash
# 使用默认端口启动
./pi-keyboard

# 使用自定义端口
PIKBD_PORT=8081 ./pi-keyboard

# 强制使用特定驱动
PIKBD_DRIVER=macos_automation ./pi-keyboard
```

## Web界面使用

启动服务后，打开浏览器访问 `http://localhost:8080` 即可使用Web界面。

### 主要功能

1. **文本输入**: 在文本框中输入内容，点击"发送文本"按钮
2. **虚拟键盘**: 点击屏幕键盘上的按键进行输入
3. **状态显示**: 实时显示操作状态和结果

### 移动端使用

- 界面已针对移动设备优化
- 支持触摸操作
- 防止意外缩放和选择
- 响应式布局适配不同屏幕尺寸

## API接口

### 单键按压

```http
GET /press?key=a&duration=100
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

## 支持的按键

- **字母**: a-z
- **数字**: 0-9
- **功能键**: enter, esc, backspace, tab, space
- **修饰键**: shift, ctrl, alt, cmd
- **方向键**: up, down, left, right

## 平台支持

### Linux (树莓派)

需要配置 USB OTG 功能：

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

## 开发

### 项目结构

```
pi-keyboard/
├── main.go              # 主程序入口
├── act/                 # 核心功能包
│   ├── driver.go        # 驱动接口定义
│   ├── factory.go       # 驱动工厂
│   ├── keyboard.go      # 键盘服务
│   ├── linux_otg_driver.go    # Linux OTG驱动
│   └── macos_driver.go  # macOS驱动
├── web/                 # Web界面文件
│   ├── index.html       # 主页面
│   ├── style.css        # 样式文件
│   └── script.js        # JavaScript逻辑
└── test/                # 测试文件
```

### 添加新驱动

1. 实现 `KeyboardDriver` 接口
2. 在 `factory.go` 中注册新驱动
3. 添加相应的配置选项

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！