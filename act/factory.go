package act

import (
	"fmt"
	"os"
	"runtime"
)

// DriverFactory 驱动工厂
type DriverFactory struct{}

// NewDriverFactory 创建驱动工厂实例
func NewDriverFactory() *DriverFactory {
	return &DriverFactory{}
}

// CreateDriver 根据平台自动创建合适的驱动
func (f *DriverFactory) CreateDriver(options ...DriverOption) (KeyboardDriver, error) {
	config := &DriverConfig{}

	// 应用配置选项
	for _, option := range options {
		option(config)
	}

	// 如果指定了驱动类型，直接创建
	if config.DriverType != "" {
		return f.createSpecificDriver(config.DriverType, config)
	}

	// 自动检测平台
	return f.createAutoDetectedDriver(config)
}

// createAutoDetectedDriver 自动检测平台并创建驱动
func (f *DriverFactory) createAutoDetectedDriver(config *DriverConfig) (KeyboardDriver, error) {
	switch runtime.GOOS {
	case "linux":
		// 检查是否有 HID Gadget 支持
		if f.hasHIDGadgetSupport(config.OutputFile) {
			return NewLinuxOTGDriver(config.OutputFile), nil
		}
		return nil, fmt.Errorf("linux 系统未检测到 hid gadget 支持，请确保 /dev/hidg0 存在")

	case "darwin": // Mac OS
		return NewMacOSDriver(), nil

	case "windows":
		return NewWindowsDriver(), nil

	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// createSpecificDriver 创建指定类型的驱动
func (f *DriverFactory) createSpecificDriver(driverType string, config *DriverConfig) (KeyboardDriver, error) {
	switch driverType {
	case DriverTypeLinuxOTG:
		return NewLinuxOTGDriver(config.OutputFile), nil

	case DriverTypeMacOS:
		return NewMacOSDriver(), nil

	case DriverTypeWindows:
		return NewWindowsDriver(), nil

	default:
		return nil, fmt.Errorf("未知的驱动类型: %s", driverType)
	}
}

// hasHIDGadgetSupport 检查是否有 HID Gadget 支持
func (f *DriverFactory) hasHIDGadgetSupport(outputFile string) bool {
	if outputFile == "" {
		outputFile = "/dev/hidg0"
	}

	// 检查设备文件是否存在
	if _, err := os.Stat(outputFile); err != nil {
		return false
	}

	// 尝试打开设备文件（读写模式）
	file, err := os.OpenFile(outputFile, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	file.Close()

	return true
}

// GetAvailableDrivers 获取当前平台可用的驱动列表
func (f *DriverFactory) GetAvailableDrivers() []string {
	var drivers []string

	switch runtime.GOOS {
	case "linux":
		if f.hasHIDGadgetSupport("") {
			drivers = append(drivers, DriverTypeLinuxOTG)
		}

	case "darwin":
		drivers = append(drivers, DriverTypeMacOS)

	case "windows":
		// Windows 暂未实现
	}

	return drivers
}

// DriverConfig 驱动配置
type DriverConfig struct {
	DriverType string // 强制指定驱动类型
	OutputFile string // Linux OTG 输出文件路径
}

// DriverOption 驱动配置选项
type DriverOption func(*DriverConfig)

// WithDriverType 指定驱动类型
func WithDriverType(driverType string) DriverOption {
	return func(config *DriverConfig) {
		config.DriverType = driverType
	}
}

// WithOutputFile 指定输出文件（仅对 Linux OTG 有效）
func WithOutputFile(outputFile string) DriverOption {
	return func(config *DriverConfig) {
		config.OutputFile = outputFile
	}
}
