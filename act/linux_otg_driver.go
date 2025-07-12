package act

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// LinuxOTGDriver Linux OTG 键盘驱动实现
type LinuxOTGDriver struct {
	outputFile string
	pressedKey string
	mu         sync.Mutex
}

// NewLinuxOTGDriver 创建 Linux OTG 驱动实例
func NewLinuxOTGDriver(outputFile string) *LinuxOTGDriver {
	if outputFile == "" {
		outputFile = "/dev/hidg0"
	}
	return &LinuxOTGDriver{
		outputFile: outputFile,
	}
}

// Press 按下并释放按键，持续指定时间（原子操作）
func (d *LinuxOTGDriver) Press(key string, duration time.Duration) error {
	key = strings.ToLower(key)
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}

	// 按下按键
	d.mu.Lock()
	d.pressedKey = key
	d.mu.Unlock()

	if err := d.sendHIDReport(); err != nil {
		// 如果按下失败，确保清理状态
		d.mu.Lock()
		d.pressedKey = ""
		d.mu.Unlock()
		d.sendHIDReport() // 尝试发送释放报文
		return err
	}

	// 持续指定时间
	time.Sleep(duration)

	// 释放按键
	d.mu.Lock()
	d.pressedKey = ""
	d.mu.Unlock()

	return d.sendHIDReport()
}

// KeyDown 按下按键（不释放）
func (d *LinuxOTGDriver) KeyDown(key string) error {
	key = strings.ToLower(key)
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	d.mu.Lock()
	d.pressedKey = key
	d.mu.Unlock()
	return d.sendHIDReport()
}

// KeyUp 释放按键
func (d *LinuxOTGDriver) KeyUp(key string) error {
	key = strings.ToLower(key)
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	d.mu.Lock()
	d.pressedKey = ""
	d.mu.Unlock()
	return d.sendHIDReport()
}

// Type 输入字符串
func (d *LinuxOTGDriver) Type(text string) error {
	for _, char := range text {
		key := strings.ToLower(string(char))
		if d.IsKeySupported(key) {
			if err := d.Press(key, 50*time.Millisecond); err != nil {
				return fmt.Errorf("输入字符 %c 失败: %v", char, err)
			}
			// 字符间间隔
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}

// IsKeySupported 检查是否支持指定按键
func (d *LinuxOTGDriver) IsKeySupported(key string) bool {
	_, ok := keyMap[strings.ToLower(key)]
	return ok
}

// Close 关闭驱动，释放资源
func (d *LinuxOTGDriver) Close() error {
	// 确保释放所有按键
	d.mu.Lock()
	d.pressedKey = ""
	d.mu.Unlock()
	return d.sendHIDReport()
}

// GetDriverType 获取驱动类型
func (d *LinuxOTGDriver) GetDriverType() string {
	return DriverTypeLinuxOTG
}

// sendHIDReport 发送 HID 报文到设备文件
func (d *LinuxOTGDriver) sendHIDReport() error {
	var report [8]byte

	d.mu.Lock()
	if d.pressedKey != "" {
		if keycode, ok := keyMap[d.pressedKey]; ok {
			report[2] = keycode
		}
	}
	d.mu.Unlock()

	file, err := os.OpenFile(d.outputFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("打开输出文件失败: %v", err)
	}
	defer file.Close()

	_, err = file.Write(report[:])
	if err != nil {
		return fmt.Errorf("写入 hid 报文失败: %v", err)
	}

	return nil
}
