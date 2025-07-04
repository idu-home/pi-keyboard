package main

import (
	"log"
	"net/http"
	"os"
	"pi-keyboard/act"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("=== Pi Keyboard 启动 ===")

	// 创建驱动工厂
	factory := act.NewDriverFactory()
	log.Printf("驱动工厂创建成功")

	// 获取配置选项
	var options []act.DriverOption
	if outputFile := os.Getenv("PIKBD_OUTPUT"); outputFile != "" {
		options = append(options, act.WithOutputFile(outputFile))
		log.Printf("配置输出文件: %s", outputFile)
	}
	if driverType := os.Getenv("PIKBD_DRIVER"); driverType != "" {
		options = append(options, act.WithDriverType(driverType))
		log.Printf("强制指定驱动类型: %s", driverType)
	}

	// 创建驱动
	log.Printf("正在创建键盘驱动...")
	driver, err := factory.CreateDriver(options...)
	if err != nil {
		log.Fatalf("创建键盘驱动失败: %v", err)
	}
	defer driver.Close()

	// 创建键盘服务
	keyboard := act.NewKeyboard(driver)
	log.Printf("键盘服务创建成功")

	// 输出驱动信息
	log.Printf("使用键盘驱动: %s", driver.GetDriverType())
	log.Printf("可用驱动列表: %v", factory.GetAvailableDrivers())

	// API 接口
	http.HandleFunc("/press", keyboard.PressHandler)
	http.HandleFunc("/actions", keyboard.ActionsHandler)
	http.HandleFunc("/type", keyboard.TypeHandler)
	http.HandleFunc("/stats", keyboard.StatsHandler)
	log.Printf("API 接口注册完成")

	// 静态文件服务器
	fs := http.FileServer(http.Dir("./web/"))
	http.Handle("/web/", http.StripPrefix("/web/", fs))
	log.Printf("静态文件服务器配置完成")

	// 根路径重定向到键盘界面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/web/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// 获取端口配置
	port := os.Getenv("PIKBD_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("=== 服务启动信息 ===")
	log.Printf("监听端口: %s", port)
	log.Printf("Web界面: http://localhost:%s", port)
	log.Printf("API统计: http://localhost:%s/stats", port)
	log.Printf("启动时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("======================")

	log.Printf("pi-keyboard Web 服务已启动，监听 %s 端口...", port)
	log.Printf("访问 http://localhost:%s 使用键盘界面", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
