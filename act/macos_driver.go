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

// Press 按下并释放按键，持续指定时间
func (d *MacOSDriver) Press(key string, duration time.Duration) error {
	log.Printf("[MACOS] 开始按键操作 - 按键: %s, 持续时间: %v", key, duration)

	if !d.IsKeySupported(key) {
		log.Printf("[MACOS] 按键操作失败 - 按键: %s, 原因: 不支持的按键", key)
		return fmt.Errorf("不支持的按键: %s", key)
	}

	macKey := d.translateKey(key)
	var script string

	// 对于字符按键，使用 keystroke
	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		script = fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, key)
		log.Printf("[MACOS] 使用字符按键模式 - 按键: %s, 脚本: %s", key, script)
	} else {
		// 对于特殊按键，使用 key code
		keyCode := d.getKeyCode(macKey)
		script = fmt.Sprintf(`tell application "System Events" to key code %s`, keyCode)
		log.Printf("[MACOS] 使用按键代码模式 - 按键: %s, macKey: %s, keyCode: %s, 脚本: %s", key, macKey, keyCode, script)
	}

	scriptStart := time.Now()
	cmd := exec.Command("osascript", "-e", script)

	// 设置超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		scriptLatency := time.Since(scriptStart)
		if err != nil {
			log.Printf("[MACOS] AppleScript 执行失败 - 按键: %s, 错误: %v, 执行时间: %v", key, err, scriptLatency)
			return fmt.Errorf("执行 AppleScript 失败: %v", err)
		}
		log.Printf("[MACOS] AppleScript 执行成功 - 按键: %s, 执行时间: %v", key, scriptLatency)

	case <-time.After(5 * time.Second):
		log.Printf("[MACOS] AppleScript 执行超时 - 按键: %s, 超时时间: 5s", key)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("AppleScript 执行超时")
	}

	// 模拟按键持续时间
	if duration > time.Millisecond {
		log.Printf("[MACOS] 等待按键持续时间 - 按键: %s, 持续时间: %v", key, duration)
		time.Sleep(duration)
	}

	log.Printf("[MACOS] 按键操作完成 - 按键: %s", key)
	return nil
}

// Type 输入字符串
func (d *MacOSDriver) Type(text string) error {
	log.Printf("[MACOS] 开始文本输入 - 文本长度: %d, 内容: %q", len(text), text)

	// 转义特殊字符
	escapedText := strings.ReplaceAll(text, `"`, `\"`)
	escapedText = strings.ReplaceAll(escapedText, `\`, `\\`)

	if escapedText != text {
		log.Printf("[MACOS] 文本已转义 - 原文: %q, 转义后: %q", text, escapedText)
	}

	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escapedText)
	log.Printf("[MACOS] 文本输入脚本: %s", script)

	scriptStart := time.Now()
	cmd := exec.Command("osascript", "-e", script)

	// 设置超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		scriptLatency := time.Since(scriptStart)
		if err != nil {
			log.Printf("[MACOS] 文本输入失败 - 文本: %q, 错误: %v, 执行时间: %v", text, err, scriptLatency)
			return fmt.Errorf("输入文本失败: %v", err)
		}
		log.Printf("[MACOS] 文本输入成功 - 文本: %q, 执行时间: %v", text, scriptLatency)

	case <-time.After(10 * time.Second):
		log.Printf("[MACOS] 文本输入超时 - 文本: %q, 超时时间: 10s", text)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("文本输入超时")
	}

	return nil
}

// IsKeySupported 检查是否支持指定按键
func (d *MacOSDriver) IsKeySupported(key string) bool {
	key = strings.ToLower(key)

	// 检查是否为单个字符
	if len(key) == 1 {
		isSupported := (key >= "a" && key <= "z") || (key >= "0" && key <= "9")
		log.Printf("[MACOS] 字符按键支持检查 - 按键: %s, 支持: %v", key, isSupported)
		return isSupported
	}

	// 检查是否为特殊按键
	_, ok := d.macKeyMap[key]
	if ok {
		log.Printf("[MACOS] 特殊按键支持检查 - 按键: %s, 支持: true (在macKeyMap中)", key)
		return true
	}

	// 检查原始键盘映射
	_, ok = keyMap[key]
	log.Printf("[MACOS] 通用按键支持检查 - 按键: %s, 支持: %v", key, ok)
	return ok
}

// Close 关闭驱动，释放资源
func (d *MacOSDriver) Close() error {
	log.Printf("[MACOS] 关闭 macOS 键盘驱动")
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
		log.Printf("[MACOS] 按键转换 - 原始: %s, 转换后: %s", key, macKey)
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
		log.Printf("[MACOS] 按键代码映射 - 按键: %s, 代码: %s", key, code)
		return code
	}

	// 对于未知按键，返回空格的 key code
	log.Printf("[MACOS] 未知按键，使用默认代码 - 按键: %s, 默认代码: 49", key)
	return "49"
}
