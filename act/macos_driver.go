package act

import (
	"fmt"
	"log"
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
	log.Printf("[MACOS] 初始化 macOS 键盘驱动")
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

// Press 按下并释放按键，优化版本
func (d *MacOSDriver) Press(key string, duration time.Duration) error {
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}

	macKey := d.translateKey(key)
	var script string

	// 对于字符按键，使用 keystroke
	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		script = fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, key)
	} else {
		// 对于特殊按键，使用 key code
		keyCode := d.getKeyCode(macKey)
		script = fmt.Sprintf(`tell application "System Events" to key code %s`, keyCode)
	}

	// 执行AppleScript（移除超时处理以提高性能）
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()

	if err != nil {
		log.Printf("[MACOS] AppleScript 执行失败 - 按键: %s, 错误: %v", key, err)
		return fmt.Errorf("执行 AppleScript 失败: %v", err)
	}

	// 不等待持续时间，让系统自然处理按键释放
	return nil
}

// Type 输入字符串，优化版本
func (d *MacOSDriver) Type(text string) error {
	// 转义特殊字符
	escapedText := strings.ReplaceAll(text, `"`, `\"`)
	escapedText = strings.ReplaceAll(escapedText, `\`, `\\`)

	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escapedText)

	// 直接执行，不等待
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()

	if err != nil {
		log.Printf("[MACOS] 文本输入失败 - 文本: %q, 错误: %v", text, err)
		return fmt.Errorf("输入文本失败: %v", err)
	}

	return nil
}

// IsKeySupported 检查是否支持指定按键（简化版）
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
	log.Printf("[MACOS] 关闭 macOS 键盘驱动")
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
		"shift":       "56",
		"control":     "59",
		"option":      "58",
		"command":     "55",
	}

	if code, ok := keyCodeMap[key]; ok {
		return code
	}

	// 对于未知按键，返回空格的 key code
	return "49"
}
