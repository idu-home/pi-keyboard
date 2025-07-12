package act

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
	"os"
	"io/ioutil"
)

// WindowsKeySender 定义按键注入接口
//
type WindowsKeySender interface {
	KeyDown(vk string) error
	KeyUp(vk string) error
	Press(vk string) error
}

type SenderType int

const (
	SenderPython SenderType = iota
	SenderPowerShell
)

// PythonKeySender 用 python 实现
//
type PythonKeySender struct{}

func (s *PythonKeySender) KeyDown(vk string) error {
	py := fmt.Sprintf(`import ctypes;ctypes.windll.user32.keybd_event(%s,0,0,0)`, vk)
	cmd := exec.Command("python", "-c", py)
	return cmd.Run()
}
func (s *PythonKeySender) KeyUp(vk string) error {
	py := fmt.Sprintf(`import ctypes;ctypes.windll.user32.keybd_event(%s,0,2,0)`, vk)
	cmd := exec.Command("python", "-c", py)
	return cmd.Run()
}
func (s *PythonKeySender) Press(vk string) error {
	py := fmt.Sprintf(`import ctypes;ctypes.windll.user32.keybd_event(%s,0,0,0);ctypes.windll.user32.keybd_event(%s,0,2,0)`, vk, vk)
	cmd := exec.Command("python", "-c", py)
	return cmd.Run()
}

// PowerShellKeySender 用 powershell 实现
//
type PowerShellKeySender struct{}

func (s *PowerShellKeySender) KeyDown(vk string) error {
	return runPowerShellKeyEvent(vk, true)
}
func (s *PowerShellKeySender) KeyUp(vk string) error {
	return runPowerShellKeyEvent(vk, false)
}
func (s *PowerShellKeySender) Press(vk string) error {
	if err := runPowerShellKeyEvent(vk, true); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return runPowerShellKeyEvent(vk, false)
}

// runPowerShellKeyEvent 写入临时 ps1 文件并执行
func runPowerShellKeyEvent(vk string, down bool) error {
	flag := "0"
	if !down {
		flag = "2"
	}
	psScript := fmt.Sprintf(`
$sig = '[DllImport("user32.dll")]public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);'
Add-Type -MemberDefinition $sig -Name NativeMethods -Namespace Win32
[Win32.NativeMethods]::keybd_event(%s,0,%s,[UIntPtr]::Zero)
`, vk, flag)
	tmpFile, err := ioutil.TempFile("", "sendkey-*.ps1")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(psScript)); err != nil {
		return err
	}
	tmpFile.Close()
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[POWERSHELL] 执行: powershell -File %s (vk=%s, flag=%s)", tmpFile.Name(), vk, flag)
	return cmd.Run()
}

// WindowsDriver Windows 键盘驱动实现
// 通过调用 python user32.SendInput 或 PowerShell 发送按键
// 支持基础按键和常用特殊键

type WindowsDriver struct {
	winKeyMap map[string]string
	senderType SenderType
	sender     WindowsKeySender
}

// NewWindowsDriver 创建 Windows 驱动实例
func NewWindowsDriver() *WindowsDriver {
	log.Printf("[WINDOWS] 初始化 Windows 键盘驱动")
	return &WindowsDriver{
		winKeyMap: map[string]string{
			"left": "0x25", "up": "0x26", "right": "0x27", "down": "0x28",
			"backspace": "0x08", "tab": "0x09", "enter": "0x0D", "shift": "0x10", "ctrl": "0x11", "alt": "0x12", "caps lock": "0x14", "esc": "0x1B", "space": "0x20", "page up": "0x21", "page down": "0x22", "end": "0x23", "home": "0x24", "insert": "0x2D", "delete": "0x2E",
			"0": "0x30", "1": "0x31", "2": "0x32", "3": "0x33", "4": "0x34", "5": "0x35", "6": "0x36", "7": "0x37", "8": "0x38", "9": "0x39",
			"a": "0x41", "b": "0x42", "c": "0x43", "d": "0x44", "e": "0x45", "f": "0x46", "g": "0x47", "h": "0x48", "i": "0x49", "j": "0x4A", "k": "0x4B", "l": "0x4C", "m": "0x4D", "n": "0x4E", "o": "0x4F", "p": "0x50", "q": "0x51", "r": "0x52", "s": "0x53", "t": "0x54", "u": "0x55", "v": "0x56", "w": "0x57", "x": "0x58", "y": "0x59", "z": "0x5A",
			"f1": "0x70", "f2": "0x71", "f3": "0x72", "f4": "0x73", "f5": "0x74", "f6": "0x75", "f7": "0x76", "f8": "0x77", "f9": "0x78", "f10": "0x79", "f11": "0x7A", "f12": "0x7B", "num lock": "0x90", "scroll lock": "0x91",
			";": "0xBA", "=": "0xBB", ",": "0xBC", "-": "0xBD", ".": "0xBE", "/": "0xBF", "`": "0xC0", "[": "0xDB", "\\": "0xDC", "]": "0xDD", "'": "0xDE",
		},
		senderType: SenderPython,
		sender:     &PythonKeySender{},
		// senderType: SenderPowerShell,
		// sender:     &PowerShellKeySender{},
	}
}

// SetSenderType 切换按键注入方案
func (d *WindowsDriver) SetSenderType(t SenderType) {
	d.senderType = t
	switch t {
	case SenderPython:
		d.sender = &PythonKeySender{}
	case SenderPowerShell:
		d.sender = &PowerShellKeySender{}
	}
}

// Press 按下并释放按键，支持持续时间
func (d *WindowsDriver) Press(key string, duration time.Duration) error {
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}

	if len(key) == 1 && ((key >= "a" && key <= "z") || (key >= "0" && key <= "9")) {
		return d.pressPython(key)
	}
	return d.pressWithDuration(key, duration)
}

// KeyDown 按下按键（不释放）
func (d *WindowsDriver) KeyDown(key string) error {
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	vk := d.getVKCode(key)
	if vk == "" {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	return d.sender.KeyDown(vk)
}

// KeyUp 释放按键
func (d *WindowsDriver) KeyUp(key string) error {
	if !d.IsKeySupported(key) {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	vk := d.getVKCode(key)
	if vk == "" {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	return d.sender.KeyUp(vk)
}

// pressWithDuration 按下-等待-释放
func (d *WindowsDriver) pressWithDuration(key string, duration time.Duration) error {
	if err := d.keyDown(key); err != nil {
		return fmt.Errorf("按键按下失败: %v", err)
	}
	time.Sleep(duration)
	if err := d.keyUp(key); err != nil {
		return fmt.Errorf("按键释放失败: %v", err)
	}
	return nil
}

// keyDown 按下按键
func (d *WindowsDriver) keyDown(key string) error {
	vk := d.getVKCode(key)
	if vk == "" {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	return d.sender.KeyDown(vk)
}

// keyUp 释放按键
func (d *WindowsDriver) keyUp(key string) error {
	vk := d.getVKCode(key)
	if vk == "" {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	return d.sender.KeyUp(vk)
}

// pressPython 直接输入字符
func (d *WindowsDriver) pressPython(key string) error {
	vk := d.getVKCode(key)
	if vk == "" {
		return fmt.Errorf("不支持的按键: %s", key)
	}
	return d.sender.Press(vk)
}

// Type 输入字符串
func (d *WindowsDriver) Type(text string) error {
	for _, char := range text {
		key := strings.ToLower(string(char))
		if d.IsKeySupported(key) {
			if err := d.Press(key, 50*time.Millisecond); err != nil {
				return fmt.Errorf("输入字符 %c 失败: %v", char, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}

// IsKeySupported 检查是否支持指定按键
func (d *WindowsDriver) IsKeySupported(key string) bool {
	key = strings.ToLower(key)
	_, ok := d.winKeyMap[key]
	return ok
}

// Close 关闭驱动
func (d *WindowsDriver) Close() error {
	log.Printf("[WINDOWS] 关闭 Windows 键盘驱动")
	return nil
}

// GetDriverType 获取驱动类型
func (d *WindowsDriver) GetDriverType() string {
	return DriverTypeWindows
}

// translateKey 通用按键名转 Windows VK
func (d *WindowsDriver) translateKey(key string) string {
	key = strings.ToLower(key)
	if winKey, ok := d.winKeyMap[key]; ok {
		return winKey
	}
	return fmt.Sprintf("ord('%s')", key)
}

func (d *WindowsDriver) getVKCode(key string) string {
	key = strings.ToLower(key)
	if vk, ok := d.winKeyMap[key]; ok {
		return vk
	}
	return ""
} 