# Pi Keyboard 脚本使用说明

## 脚本列表

- `setup.sh` - 基本 USB gadget 设置
- `cleanup.sh` - 清理 USB gadget 配置
- `create_hid_device.sh` - 创建 HID 设备节点
- `setup_complete.sh` - 完整设置脚本（推荐使用）

## 快速开始

### 方法一：使用完整设置脚本（推荐）

```bash
# 给脚本添加执行权限
chmod +x *.sh

# 运行完整设置
sudo ./setup_complete.sh
```

### 方法二：手动分步设置

```bash
# 1. 清理旧配置
sudo ./cleanup.sh

# 2. 设置 USB gadget
sudo ./setup.sh

# 3. 等待设备稳定
sleep 3

# 4. 创建 HID 设备节点
sudo ./create_hid_device.sh
```

## 故障排除

### 问题：设备节点未创建

如果 `/dev/hidg0` 不存在，手动创建：

```bash
# 1. 查找设备号
cat /sys/kernel/config/usb_gadget/pi_keyboard/functions/hid.usb0/dev

# 2. 创建设备节点（替换主次设备号）
sudo mknod /dev/hidg0 c <主设备号> <次设备号>

# 3. 设置权限
sudo chmod 666 /dev/hidg0
```

### 问题：UDC 绑定失败

检查 USB OTG 是否启用：

```bash
# 检查 UDC 目录
ls /sys/class/udc

# 如果没有 UDC，启用 USB OTG
echo "dtoverlay=dwc2" | sudo tee -a /boot/config.txt
echo "dtoverlay=dwc2,dr_mode=peripheral" | sudo tee -a /boot/config.txt
sudo reboot
```

### 问题：权限不足

确保以 root 权限运行：

```bash
sudo -i
./setup_complete.sh
```

## 验证设置

设置完成后，检查以下项目：

```bash
# 1. 检查 USB gadget
ls -la /sys/kernel/config/usb_gadget/pi_keyboard/

# 2. 检查 HID 设备节点
ls -la /dev/hidg0

# 3. 检查设备权限
ls -la /dev/hidg0

# 4. 测试设备写入
echo -ne '\x00\x00\x04\x00\x00\x00\x00\x00' > /dev/hidg0
```

## 启动服务

设置完成后，启动 Pi Keyboard 服务：

```bash
./main
```

访问 http://localhost:8081 使用 Web 界面。 