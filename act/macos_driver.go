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

// Press 按下并释放按键，支持真实的持续时间
func (d *MacOSDriver) Press(key string, duration time.Duration) error {
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}

	// 对于字符按键，使用瞬时输入（不需要持续时间）
	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		return d.pressAppleScript(key)
	}

	// 对于特殊按键，实现真正的按下-等待-释放
	return d.pressWithDuration(key, duration)
}

// pressWithDuration 实现真正的按下-等待-释放逻辑
func (d *MacOSDriver) pressWithDuration(key string, duration time.Duration) error {
	// 步骤1: 按下按键
	if err := d.keyDown(key); err != nil {
		return fmt.Errorf("按键按下失败: %v", err)
	}

	// 步骤2: 等待持续时间
	time.Sleep(duration)

	// 步骤3: 释放按键
	if err := d.keyUp(key); err != nil {
		return fmt.Errorf("按键释放失败: %v", err)
	}

	return nil
}

// keyDown 按下按键（不释放）
func (d *MacOSDriver) keyDown(key string) error {
	macKey := d.translateKey(key)
	keyCode := d.getKeyCode(macKey)

	// 使用 AppleScript 发送按键按下事件
	script := fmt.Sprintf(`tell application "System Events" to key down %s`, keyCode)
	cmd := exec.Command("osascript", "-e", script)

	err := cmd.Run()
	if err != nil {
		log.Printf("[MACOS] 按键按下失败 - 按键: %s, 错误: %v", key, err)
		return fmt.Errorf("按键按下失败: %v", err)
	}

	return nil
}

// keyUp 释放按键
func (d *MacOSDriver) keyUp(key string) error {
	macKey := d.translateKey(key)
	keyCode := d.getKeyCode(macKey)

	// 使用 AppleScript 发送按键释放事件
	script := fmt.Sprintf(`tell application "System Events" to key up %s`, keyCode)
	cmd := exec.Command("osascript", "-e", script)

	err := cmd.Run()
	if err != nil {
		log.Printf("[MACOS] 按键释放失败 - 按键: %s, 错误: %v", key, err)
		return fmt.Errorf("按键释放失败: %v", err)
	}

	return nil
}

// pressAppleScript 使用 cliclick 或 AppleScript
func (d *MacOSDriver) pressAppleScript(key string) error {
	// 策略1: 尝试使用 cliclick（更快）
	if err := d.pressWithCliclick(key); err == nil {
		log.Printf("[MACOS] 使用驱动: cliclick - 按键: %s", key)
		return nil
	}

	// 策略2: 回退到 AppleScript
	log.Printf("[MACOS] 使用驱动: AppleScript - 按键: %s", key)
	return d.pressWithAppleScript(key)
}

// pressWithCliclick 使用 cliclick 工具
func (d *MacOSDriver) pressWithCliclick(key string) error {
	var cmd *exec.Cmd

	// 字符和数字使用文本输入
	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		cmd = exec.Command("cliclick", fmt.Sprintf("t:%s", key))
	} else {
		// 特殊按键映射
		cliclickKey := d.mapToCliclickKey(key)
		if cliclickKey == "" {
			return fmt.Errorf("cliclick 不支持按键: %s", key)
		}
		cmd = exec.Command("cliclick", fmt.Sprintf("kp:%s", cliclickKey))
	}

	err := cmd.Run()
	if err != nil {
		log.Printf("[MACOS] cliclick 执行失败 - 按键: %s, 错误: %v, 回退到AppleScript", key, err)
		return fmt.Errorf("cliclick 执行失败: %v", err)
	}

	return nil
}

// pressWithAppleScript 使用传统的 AppleScript
func (d *MacOSDriver) pressWithAppleScript(key string) error {
	var cmd *exec.Cmd

	// 优化：对字符使用预编译脚本
	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		// 使用预编译的脚本
		cmd = exec.Command("osascript", "scripts/keystroke.scpt", key)
	} else {
		// 特殊按键：使用 key code（更快）
		macKey := d.translateKey(key)
		keyCode := d.getKeyCode(macKey)
		script := fmt.Sprintf(`tell application "System Events" to key code %s`, keyCode)
		cmd = exec.Command("osascript", "-e", script)
	}

	// 关键优化：使用 Run() 而不是 Start()，确保按键真正执行
	// 但通过异步队列已经实现了非阻塞，所以这里可以等待完成
	err := cmd.Run()
	if err != nil {
		log.Printf("[MACOS] AppleScript 执行失败 - 按键: %s, 错误: %v", key, err)
		return fmt.Errorf("执行 applescript 失败: %v", err)
	}

	return nil
}

// mapToCliclickKey 将按键名映射到 cliclick 支持的按键名
func (d *MacOSDriver) mapToCliclickKey(key string) string {
	keyMap := map[string]string{
		"enter":     "return",
		"esc":       "esc",
		"backspace": "delete",
		"space":     "space",
		"tab":       "tab",
		"up":        "arrow-up",
		"down":      "arrow-down",
		"left":      "arrow-left",
		"right":     "arrow-right",
	}

	if cliclickKey, ok := keyMap[key]; ok {
		return cliclickKey
	}

	return "" // 不支持的按键
}

// Type 输入字符串，极致优化版本
func (d *MacOSDriver) Type(text string) error {
	// 转义特殊字符
	escapedText := strings.ReplaceAll(text, `"`, `\"`)
	escapedText = strings.ReplaceAll(escapedText, `\`, `\\`)

	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escapedText)

	// 优化：使用 Start() 立即返回，不等待执行完成
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Start()

	if err != nil {
		log.Printf("[MACOS] 文本输入启动失败 - 文本: %q, 错误: %v", text, err)
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
