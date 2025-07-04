package main

import (
	"log"
	"net/http"
	"os"
	"pi-keyboard/act"
)

func main() {
	// 创建驱动工厂
	factory := act.NewDriverFactory()

	// 获取配置选项
	var options []act.DriverOption
	if outputFile := os.Getenv("PIKBD_OUTPUT"); outputFile != "" {
		options = append(options, act.WithOutputFile(outputFile))
	}
	if driverType := os.Getenv("PIKBD_DRIVER"); driverType != "" {
		options = append(options, act.WithDriverType(driverType))
	}

	// 创建驱动
	driver, err := factory.CreateDriver(options...)
	if err != nil {
		log.Fatalf("创建键盘驱动失败: %v", err)
	}
	defer driver.Close()

	// 创建键盘服务
	keyboard := act.NewKeyboard(driver)

	// 输出驱动信息
	log.Printf("使用键盘驱动: %s", driver.GetDriverType())
	log.Printf("可用驱动列表: %v", factory.GetAvailableDrivers())

	http.HandleFunc("/press", keyboard.PressHandler)
	http.HandleFunc("/actions", keyboard.ActionsHandler)
	http.HandleFunc("/type", keyboard.TypeHandler)

	// 获取端口配置
	port := os.Getenv("PIKBD_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("pi-keyboard Web 服务已启动，监听 %s 端口...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
