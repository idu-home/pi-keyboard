#!/bin/bash

echo "=== Pi Keyboard 完整设置脚本 ==="

# 检查是否以 root 权限运行
if [ "$EUID" -ne 0 ]; then
    echo "请以 root 权限运行此脚本"
    echo "使用方法: sudo $0"
    exit 1
fi

# 1. 清理之前的配置
echo "步骤 1: 清理之前的配置..."
./cleanup.sh

# 2. 运行基本设置
echo "步骤 2: 运行基本 USB gadget 设置..."
./setup.sh

# 3. 等待设备稳定
echo "步骤 3: 等待设备稳定..."
sleep 3

# 4. 创建 HID 设备节点
echo "步骤 4: 创建 HID 设备节点..."
./create_hid_device.sh

# 4.5. 设置设备权限
echo "步骤 4.5: 设置设备权限..."
if [ -e "/dev/hidg0" ]; then
    # 设置权限让所有用户都能访问
    chmod 666 /dev/hidg0
    echo "✓ 设备权限设置完成"
    ls -la /dev/hidg0
else
    echo "✗ 设备节点不存在，无法设置权限"
fi

# 4.6. 安装 udev 规则
echo "步骤 4.6: 安装 udev 规则..."
UDEV_RULES_DIR="/etc/udev/rules.d"
UDEV_RULES_FILE="$UDEV_RULES_DIR/99-hidg.rules"

if [ -f "99-hidg.rules" ]; then
    cp 99-hidg.rules "$UDEV_RULES_FILE"
    echo "✓ udev 规则已安装到 $UDEV_RULES_FILE"
    
    # 重新加载 udev 规则
    udevadm control --reload-rules
    udevadm trigger
    echo "✓ udev 规则已重新加载"
else
    echo "⚠ udev 规则文件不存在，跳过安装"
fi

# 5. 最终检查
echo "步骤 5: 最终检查..."
if [ -e "/dev/hidg0" ]; then
    echo "✓ 所有设置完成！"
    echo "✓ HID 设备节点: /dev/hidg0"
    echo "✓ 可以运行 ./main 启动服务"
else
    echo "✗ HID 设备节点创建失败"
    echo "请检查错误信息并手动创建设备节点"
    exit 1
fi

echo "=== 设置完成 ===" 