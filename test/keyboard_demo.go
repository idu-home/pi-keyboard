package main

import (
	"fmt"
	"log"
	"pi-keyboard/act"
	"runtime"
	"time"
)

func main() {
	fmt.Printf("当前操作系统: %s\n", runtime.GOOS)

	// 创建驱动工厂
	factory := act.NewDriverFactory()

	// 显示可用驱动
	available := factory.GetAvailableDrivers()
	fmt.Printf("可用驱动: %v\n", available)

	if len(available) == 0 {
		log.Fatal("当前平台没有可用的键盘驱动")
	}

	// 创建驱动
	driver, err := factory.CreateDriver()
	if err != nil {
		log.Fatalf("创建驱动失败: %v", err)
	}
	defer driver.Close()

	fmt.Printf("使用驱动: %s\n", driver.GetDriverType())

	// 测试按键支持
	testKeys := []string{"a", "1", "space", "enter", "esc", "shift"}
	fmt.Println("\n按键支持测试:")
	for _, key := range testKeys {
		supported := driver.IsKeySupported(key)
		fmt.Printf("  %s: %v\n", key, supported)
	}

	// 根据平台进行不同的测试
	switch driver.GetDriverType() {
	case act.DriverTypeMacOS:
		testMacOS(driver)
	case act.DriverTypeLinuxOTG:
		testLinuxOTG(driver)
	default:
		fmt.Printf("未知驱动类型: %s\n", driver.GetDriverType())
	}
}

func testMacOS(driver act.KeyboardDriver) {
	fmt.Println("\n=== Mac OS 驱动测试 ===")
	fmt.Println("注意：这将在你的 Mac 上模拟键盘输入")
	fmt.Println("请确保当前有文本编辑器或终端处于活动状态")

	time.Sleep(3 * time.Second)

	// 测试单个按键（原子操作）
	fmt.Println("测试单个按键（原子操作）...")
	if err := driver.Press("h", 100*time.Millisecond); err != nil {
		fmt.Printf("按键测试失败: %v\n", err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	// 测试文本输入
	fmt.Println("测试文本输入...")
	if err := driver.Type("ello world"); err != nil {
		fmt.Printf("文本输入测试失败: %v\n", err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	// 测试特殊按键
	fmt.Println("测试回车键...")
	if err := driver.Press("enter", 100*time.Millisecond); err != nil {
		fmt.Printf("回车键测试失败: %v\n", err)
		return
	}

	// 测试不同持续时间的按键
	fmt.Println("测试不同持续时间的按键...")
	keys := []struct {
		key      string
		duration time.Duration
	}{
		{"a", 50 * time.Millisecond},
		{"b", 100 * time.Millisecond},
		{"c", 200 * time.Millisecond},
	}

	for _, k := range keys {
		if err := driver.Press(k.key, k.duration); err != nil {
			fmt.Printf("按键 %s 测试失败: %v\n", k.key, err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Mac OS 驱动测试完成")
}

func testLinuxOTG(driver act.KeyboardDriver) {
	fmt.Println("\n=== Linux OTG 驱动测试 ===")
	fmt.Println("注意：这需要连接到目标设备才能看到效果")

	// 测试单个按键
	fmt.Println("测试单个按键...")
	if err := driver.Press("a", 100*time.Millisecond); err != nil {
		fmt.Printf("按键测试失败: %v\n", err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	// 测试文本输入
	fmt.Println("测试文本输入...")
	if err := driver.Type("hello"); err != nil {
		fmt.Printf("文本输入测试失败: %v\n", err)
		return
	}

	fmt.Println("Linux OTG 驱动测试完成")
}
