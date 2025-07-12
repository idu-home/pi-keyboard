# 0. 复制脚本
# /usr/local/bin/setup_pi_keyboard_gadget.sh

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
ls /sys/class/udc > UDC