#!/bin/bash

echo "快速修复 HID 设备权限..."

# 检查是否以 root 权限运行
if [ "$EUID" -ne 0 ]; then
    echo "请以 root 权限运行此脚本"
    echo "使用方法: sudo $0"
    exit 1
fi

# 检查设备是否存在
if [ ! -e "/dev/hidg0" ]; then
    echo "错误: /dev/hidg0 不存在"
    echo "请先运行 setup.sh 创建设备"
    exit 1
fi

# 设置设备权限
echo "设置设备权限..."
chmod 666 /dev/hidg0

# 检查权限设置结果
echo "当前设备权限:"
ls -la /dev/hidg0

# 测试设备访问
echo "测试设备访问..."
if [ -r "/dev/hidg0" ] && [ -w "/dev/hidg0" ]; then
    echo "✓ 权限设置成功"
    
    # 简单测试写入
    echo -ne '\x00\x00\x00\x00\x00\x00\x00\x00' > /dev/hidg0 2>/dev/null
    if [ $? -eq 0 ]; then
        echo "✓ 设备写入测试成功"
    else
        echo "✗ 设备写入测试失败"
    fi
else
    echo "✗ 权限设置失败"
    exit 1
fi

echo "权限修复完成！现在可以运行 ./main 了。" 