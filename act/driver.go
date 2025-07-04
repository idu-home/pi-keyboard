package act

import "time"

// KeyboardDriver 定义键盘驱动的统一接口
type KeyboardDriver interface {
	// Press 按下并释放按键，持续指定时间（原子操作，安全）
	Press(key string, duration time.Duration) error

	// Type 输入字符串
	Type(text string) error

	// IsKeySupported 检查是否支持指定按键
	IsKeySupported(key string) bool

	// Close 关闭驱动，释放资源
	Close() error

	// GetDriverType 获取驱动类型
	GetDriverType() string
}

// DriverType 驱动类型常量
const (
	DriverTypeLinuxOTG = "linux_otg"
	DriverTypeMacOS    = "macos_automation"
	DriverTypeWindows  = "windows_automation"
)
