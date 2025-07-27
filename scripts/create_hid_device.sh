#!/bin/bash

echo "创建 HID 设备节点..."

# 检查是否以 root 权限运行
if [ "$EUID" -ne 0 ]; then
    echo "请以 root 权限运行此脚本"
    echo "使用方法: sudo $0"
    exit 1
fi

# 检查 gadget 是否存在
if [ ! -d "/sys/kernel/config/usb_gadget/pi_keyboard" ]; then
    echo "错误: pi_keyboard gadget 不存在"
    echo "请先运行 setup.sh 创建 USB gadget"
    exit 1
fi

# 检查 HID function 是否存在
if [ ! -d "/sys/kernel/config/usb_gadget/pi_keyboard/functions/hid.usb0" ]; then
    echo "错误: HID function 不存在"
    exit 1
fi

# 检查设备号文件
DEV_FILE="/sys/kernel/config/usb_gadget/pi_keyboard/functions/hid.usb0/dev"
if [ ! -f "$DEV_FILE" ]; then
    echo "错误: 设备号文件不存在: $DEV_FILE"
    echo "请确保 USB gadget 已正确激活"
    exit 1
fi

# 读取设备号
DEV_NUMBERS=$(cat "$DEV_FILE")
if [ -z "$DEV_NUMBERS" ]; then
    echo "错误: 无法读取设备号"
    exit 1
fi

echo "找到设备号: $DEV_NUMBERS"

# 解析主次设备号
MAJOR=$(echo $DEV_NUMBERS | cut -d: -f1)
MINOR=$(echo $DEV_NUMBERS | cut -d: -f2)

echo "主设备号: $MAJOR"
echo "次设备号: $MINOR"

# 删除已存在的设备节点（如果有）
if [ -e "/dev/hidg0" ]; then
    echo "删除已存在的设备节点..."
    rm -f /dev/hidg0
fi

# 创建设备节点
echo "创建设备节点 /dev/hidg0..."
mknod /dev/hidg0 c $MAJOR $MINOR

if [ $? -eq 0 ]; then
    echo "✓ 设备节点创建成功"
    
    # 设置权限
    chmod 666 /dev/hidg0
    echo "✓ 权限设置完成"
    
    # 显示设备信息
    echo "设备节点信息:"
    ls -la /dev/hidg0
    
    echo "HID 设备节点创建完成！"
else
    echo "✗ 设备节点创建失败"
    exit 1
fi 