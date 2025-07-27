#!/bin/bash

# 0. 清理之前的配置（如果存在）
cleanup_gadget() {
    echo "清理之前的 USB gadget 配置..."
    cd /sys/kernel/config/usb_gadget/
    if [ -d "pi_keyboard" ]; then
        # 先解除 UDC 绑定
        if [ -f "pi_keyboard/UDC" ]; then
            echo "" > pi_keyboard/UDC 2>/dev/null || true
        fi
        
        # 删除符号链接
        if [ -L "pi_keyboard/configs/c.1/hid.usb0" ]; then
            rm pi_keyboard/configs/c.1/hid.usb0
        fi
        
        # 删除整个 gadget 目录
        rm -rf pi_keyboard
        echo "已清理旧的 pi_keyboard gadget"
    fi
}

# 执行清理
cleanup_gadget

# 检查 UDC 可用性
check_udc() {
    echo "检查可用的 UDC..."
    if [ ! -d "/sys/class/udc" ]; then
        echo "错误: /sys/class/udc 目录不存在"
        exit 1
    fi
    
    UDC_COUNT=$(ls /sys/class/udc | wc -l)
    if [ "$UDC_COUNT" -eq 0 ]; then
        echo "错误: 没有可用的 UDC"
        echo "请确保启用了 USB OTG 功能"
        exit 1
    fi
    
    echo "找到 $UDC_COUNT 个可用的 UDC:"
    ls /sys/class/udc
}

check_udc

# 1. 加载模块
modprobe libcomposite

# 2. 创建 gadget
cd /sys/kernel/config/usb_gadget/
mkdir -p pi_keyboard
cd pi_keyboard

echo 0x1d6b > idVendor  # Linux Foundation
echo 0x0104 > idProduct # Multifunction Composite Gadget
echo 0x0100 > bcdDevice
echo 0x0200 > bcdUSB

# 3. 设备描述
mkdir -p strings/0x409
echo "deadbeef1234" > strings/0x409/serialnumber
echo "Raspberry Pi" > strings/0x409/manufacturer
echo "Pi Keyboard" > strings/0x409/product

# 4. 配置
mkdir -p configs/c.1/strings/0x409
echo "Config 1: HID Keyboard" > configs/c.1/strings/0x409/configuration
echo 120 > configs/c.1/MaxPower

# 5. 创建 HID function
mkdir -p functions/hid.usb0
echo 1 > functions/hid.usb0/protocol
echo 1 > functions/hid.usb0/subclass
echo 8 > functions/hid.usb0/report_length
echo -ne '\x05\x01\x09\x06\xa1\x01\x05\x07\x19\xe0\x29\xe7\x15\x00\x25\x01\x75\x01\x95\x08\x81\x02\x95\x01\x75\x08\x81\x01\x95\x05\x75\x01\x05\x08\x19\x01\x29\x05\x91\x02\x95\x01\x75\x03\x91\x01\x95\x06\x75\x08\x15\x00\x25\x65\x05\x07\x19\x00\x29\x65\x81\x00\xc0' > functions/hid.usb0/report_desc

# 6. 绑定 function 到 config
ln -s functions/hid.usb0 configs/c.1/

# 7. 绑定 UDC
echo "绑定 UDC..."
UDC_NAME=$(ls /sys/class/udc | head -1)
if [ -z "$UDC_NAME" ]; then
    echo "错误: 没有找到可用的 UDC"
    exit 1
fi

echo "$UDC_NAME" > UDC
if [ $? -eq 0 ]; then
    echo "成功绑定 UDC: $UDC_NAME"
    echo "USB HID Keyboard gadget 已激活"
else
    echo "错误: 绑定 UDC 失败"
    exit 1
fi

# 8. 检查设备节点
echo "检查 HID 设备节点..."
sleep 2  # 等待设备节点创建

if [ -e "/dev/hidg0" ]; then
    echo "✓ HID 设备节点 /dev/hidg0 已创建"
    ls -la /dev/hidg0
else
    echo "⚠ HID 设备节点 /dev/hidg0 未找到"
    echo "尝试手动创建设备节点..."
    
    # 查找 HID function 的主次设备号
    if [ -e "/sys/kernel/config/usb_gadget/pi_keyboard/functions/hid.usb0/dev" ]; then
        DEV_NUMBERS=$(cat /sys/kernel/config/usb_gadget/pi_keyboard/functions/hid.usb0/dev)
        if [ ! -z "$DEV_NUMBERS" ]; then
            echo "找到设备号: $DEV_NUMBERS"
            # 创建设备节点
            mknod /dev/hidg0 c $DEV_NUMBERS 2>/dev/null || echo "创建设备节点失败，可能需要手动创建"
        fi
    fi
    
    echo "如果设备节点仍未创建，请手动执行："
    echo "1. 查找设备号: cat /sys/kernel/config/usb_gadget/pi_keyboard/functions/hid.usb0/dev"
    echo "2. 创建设备节点: sudo mknod /dev/hidg0 c <主设备号> <次设备号>"
    echo "3. 设置权限: sudo chmod 666 /dev/hidg0"
fi

echo "设置完成！"
