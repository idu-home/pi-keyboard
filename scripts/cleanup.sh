#!/bin/bash

echo "清理 USB HID Keyboard gadget..."

cd /sys/kernel/config/usb_gadget/

if [ -d "pi_keyboard" ]; then
    echo "找到 pi_keyboard gadget，正在清理..."
    
    # 解除 UDC 绑定
    if [ -f "pi_keyboard/UDC" ]; then
        echo "解除 UDC 绑定..."
        echo "" > pi_keyboard/UDC 2>/dev/null || true
    fi
    
    # 删除符号链接
    if [ -L "pi_keyboard/configs/c.1/hid.usb0" ]; then
        echo "删除符号链接..."
        rm pi_keyboard/configs/c.1/hid.usb0
    fi
    
    # 删除整个 gadget 目录
    echo "删除 gadget 目录..."
    rm -rf pi_keyboard
    
    echo "清理完成"
else
    echo "没有找到 pi_keyboard gadget"
fi

echo "检查是否还有其他 gadget..."
ls -la /sys/kernel/config/usb_gadget/ 2>/dev/null || echo "USB gadget 配置目录不存在" 