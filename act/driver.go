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

// TouchpadDriver 定义触控板驱动的统一接口
type TouchpadDriver interface {
	// MoveTouchpad 移动触控板（相对移动）
	MoveTouchpad(deltaX, deltaY int) error

	// ClickTouchpad 点击触控板
	ClickTouchpad(button string, clickType string) error

	// ScrollTouchpad 滚动触控板
	ScrollTouchpad(deltaX, deltaY int) error

	// GetTouchpadPosition 获取当前光标位置
	GetTouchpadPosition() (int, int, error)
}

// InputDriver 综合输入驱动接口（键盘+鼠标）
type InputDriver interface {
	KeyboardDriver
	TouchpadDriver
}

// DriverType 驱动类型常量
const (
	DriverTypeLinuxOTG = "linux_otg"
	DriverTypeMacOS    = "macos_automation"
	DriverTypeWindows  = "windows_automation"
)
