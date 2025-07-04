package act

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MacOSDriver Mac OS 键盘驱动实现
type MacOSDriver struct {
	// Mac OS 特殊按键映射
	macKeyMap map[string]string
}

// NewMacOSDriver 创建 Mac OS 驱动实例
func NewMacOSDriver() *MacOSDriver {
	return &MacOSDriver{
		macKeyMap: map[string]string{
			"enter":     "return",
			"esc":       "escape",
			"backspace": "delete",
			"space":     "space",
			"tab":       "tab",
			"shift":     "shift",
			"cmd":       "command",
			"ctrl":      "control",
			"alt":       "option",
			"up":        "up arrow",
			"down":      "down arrow",
			"left":      "left arrow",
			"right":     "right arrow",
		},
	}
}

// Press 按下并释放按键，持续指定时间
func (d *MacOSDriver) Press(key string, duration time.Duration) error {
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}

	macKey := d.translateKey(key)
	script := fmt.Sprintf(`tell application "System Events" to key code %s`, d.getKeyCode(macKey))

	// 对于字符按键，使用 keystroke
	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		script = fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, key)
	}

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("执行 AppleScript 失败: %v", err)
	}

	// 模拟按键持续时间
	if duration > time.Millisecond {
		time.Sleep(duration)
	}

	return nil
}

// Type 输入字符串
func (d *MacOSDriver) Type(text string) error {
	// 转义特殊字符
	escapedText := strings.ReplaceAll(text, `"`, `\"`)
	escapedText = strings.ReplaceAll(escapedText, `\`, `\\`)

	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escapedText)

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("输入文本失败: %v", err)
	}

	return nil
}

// IsKeySupported 检查是否支持指定按键
func (d *MacOSDriver) IsKeySupported(key string) bool {
	key = strings.ToLower(key)

	// 检查是否为单个字符
	if len(key) == 1 {
		return (key >= "a" && key <= "z") || (key >= "0" && key <= "9")
	}

	// 检查是否为特殊按键
	_, ok := d.macKeyMap[key]
	if ok {
		return true
	}

	// 检查原始键盘映射
	_, ok = keyMap[key]
	return ok
}

// Close 关闭驱动，释放资源
func (d *MacOSDriver) Close() error {
	// Mac OS 驱动无需特殊清理
	return nil
}

// GetDriverType 获取驱动类型
func (d *MacOSDriver) GetDriverType() string {
	return DriverTypeMacOS
}

// translateKey 将通用按键名转换为 Mac OS 按键名
func (d *MacOSDriver) translateKey(key string) string {
	key = strings.ToLower(key)
	if macKey, ok := d.macKeyMap[key]; ok {
		return macKey
	}
	return key
}

// getKeyCode 获取按键的 key code（用于特殊按键）
func (d *MacOSDriver) getKeyCode(key string) string {
	keyCodeMap := map[string]string{
		"return":      "36",
		"escape":      "53",
		"delete":      "51",
		"space":       "49",
		"tab":         "48",
		"up arrow":    "126",
		"down arrow":  "125",
		"left arrow":  "123",
		"right arrow": "124",
	}

	if code, ok := keyCodeMap[key]; ok {
		return code
	}

	// 对于未知按键，返回空格的 key code
	return "49"
}
