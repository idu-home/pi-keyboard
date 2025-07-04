package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"pi-keyboard/act"
	"pi-keyboard/logger"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// 命令行参数定义
	var (
		port       = flag.String("port", "8081", "服务端口")
		driverType = flag.String("driver", "", "强制指定驱动类型 (linux_otg, macos_automation)")
		outputFile = flag.String("output", "", "Linux OTG 输出文件路径")

		// 日志配置
		enableHTTPLog   = flag.Bool("log", true, "是否启用HTTP日志")
		logOutput       = flag.String("log-output", "stdout", "日志输出目标 (stdout/file/both)")
		logFile         = flag.String("log-file", "pi-keyboard.log", "日志文件路径")
		logRequestBody  = flag.Bool("log-req-body", false, "是否记录请求体")
		logResponseBody = flag.Bool("log-resp-body", true, "是否记录响应体")

		showHelp = flag.Bool("help", false, "显示帮助信息")
	)

	flag.Parse()

	// 显示帮助信息
	if *showHelp {
		fmt.Println("Pi Keyboard - 远程键盘控制服务")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Printf("  %s [选项]\n", os.Args[0])
		fmt.Println()
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("示例:")
		fmt.Printf("  %s -port 8081 -log-output file -log-file ./logs/api.log\n", os.Args[0])
		fmt.Printf("  %s -driver macos_automation -log-req-body\n", os.Args[0])
		fmt.Printf("  %s -log false\n", os.Args[0])
		return
	}

	log.Printf("=== Pi Keyboard 启动 ===")

	// 创建驱动工厂
	factory := act.NewDriverFactory()
	log.Printf("驱动工厂创建成功")

	// 获取配置选项
	var options []act.DriverOption
	if *outputFile != "" {
		options = append(options, act.WithOutputFile(*outputFile))
		log.Printf("配置输出文件: %s", *outputFile)
	}
	if *driverType != "" {
		options = append(options, act.WithDriverType(*driverType))
		log.Printf("强制指定驱动类型: %s", *driverType)
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

	// 创建日志配置 - 硬编码需要记录日志的API
	logConfig := &logger.LogConfig{
		EnableHTTPLog:   *enableHTTPLog,
		Output:          *logOutput,
		LogFile:         *logFile,
		LogRequestBody:  *logRequestBody,
		LogResponseBody: *logResponseBody,
	}

	// 创建HTTP日志记录器
	httpLogger, err := logger.NewHTTPLogger(logConfig)
	if err != nil {
		log.Fatalf("创建HTTP日志记录器失败: %v", err)
	}
	defer httpLogger.Close()

	// 输出日志配置信息
	log.Printf("HTTP日志配置: 启用=%v, 输出=%s, 文件=%s",
		logConfig.EnableHTTPLog, logConfig.Output, logConfig.LogFile)
	log.Printf("记录的API: /press, /press-sync, /actions, /type")

	// API 接口注册 - 有选择性地使用日志中间件
	// 核心功能API - 记录日志
	http.Handle("/press", httpLogger.Middleware(http.HandlerFunc(keyboard.PressHandler)))
	http.Handle("/press-sync", httpLogger.Middleware(http.HandlerFunc(keyboard.PressHandlerSync)))
	http.Handle("/actions", httpLogger.Middleware(http.HandlerFunc(keyboard.ActionsHandler)))
	http.Handle("/type", httpLogger.Middleware(http.HandlerFunc(keyboard.TypeHandler)))

	// 统计接口 - 不记录日志（避免过多日志）
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

	log.Printf("=== 服务启动信息 ===")
	log.Printf("监听端口: %s", *port)
	log.Printf("Web界面: http://localhost:%s", *port)
	log.Printf("API统计: http://localhost:%s/stats", *port)
	log.Printf("启动时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("======================")

	log.Printf("pi-keyboard Web 服务已启动，监听 %s 端口...", *port)
	log.Printf("访问 http://localhost:%s 使用键盘界面", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
