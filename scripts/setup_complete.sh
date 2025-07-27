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