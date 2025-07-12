package act

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HID 键盘 keycode 映射表（部分，后续可扩展）
var keyMap = map[string]byte{
	// 字母
	"a": 0x04, "b": 0x05, "c": 0x06, "d": 0x07, "e": 0x08,
	"f": 0x09, "g": 0x0a, "h": 0x0b, "i": 0x0c, "j": 0x0d,
	"k": 0x0e, "l": 0x0f, "m": 0x10, "n": 0x11, "o": 0x12,
	"p": 0x13, "q": 0x14, "r": 0x15, "s": 0x16, "t": 0x17,
	"u": 0x18, "v": 0x19, "w": 0x1a, "x": 0x1b, "y": 0x1c, "z": 0x1d,
	// 数字
	"1": 0x1e, "2": 0x1f, "3": 0x20, "4": 0x21, "5": 0x22,
	"6": 0x23, "7": 0x24, "8": 0x25, "9": 0x26, "0": 0x27,
	// 控制键
	"enter": 0x28, "esc": 0x29, "backspace": 0x2a, "tab": 0x2b, "space": 0x2c,
	"-": 0x2d, "=": 0x2e, "[": 0x2f, "]": 0x30, "\\": 0x31,
	"nonus#": 0x32, // 非美式#和~，部分键盘
	";": 0x33, "'": 0x34, "`": 0x35, ",": 0x36, ".": 0x37, "/": 0x38,
	"capslock": 0x39,
	// 功能键
	"f1": 0x3a, "f2": 0x3b, "f3": 0x3c, "f4": 0x3d, "f5": 0x3e, "f6": 0x3f,
	"f7": 0x40, "f8": 0x41, "f9": 0x42, "f10": 0x43, "f11": 0x44, "f12": 0x45,
	"printscreen": 0x46, "scrolllock": 0x47, "pause": 0x48, "insert": 0x49,
	"home": 0x4a, "pageup": 0x4b, "delete": 0x4c, "end": 0x4d, "pagedown": 0x4e,
	"right": 0x4f, "left": 0x50, "down": 0x51, "up": 0x52,
	// 小键盘
	"numlock": 0x53, "kp/": 0x54, "kp*": 0x55, "kp-": 0x56, "kp+": 0x57,
	"kpenter": 0x58, "kp1": 0x59, "kp2": 0x5a, "kp3": 0x5b, "kp4": 0x5c,
	"kp5": 0x5d, "kp6": 0x5e, "kp7": 0x5f, "kp8": 0x60, "kp9": 0x61, "kp0": 0x62,
	"kp.": 0x63, "nonus\\": 0x64,
	// 修饰键
	"application": 0x65, "power": 0x66, "kpequal": 0x67,
	"f13": 0x68, "f14": 0x69, "f15": 0x6a, "f16": 0x6b, "f17": 0x6c, "f18": 0x6d,
	"f19": 0x6e, "f20": 0x6f, "f21": 0x70, "f22": 0x71, "f23": 0x72, "f24": 0x73,
	// 控制/系统
	"control": 0xe0, "shift": 0xe1, "alt": 0xe2, "gui": 0xe3, // 左侧
	"rcontrol": 0xe4, "rshift": 0xe5, "ralt": 0xe6, "rgui": 0xe7, // 右侧
	// 媒体/系统控制（部分键盘支持）
	"mute": 0x7f, "volumeup": 0x80, "volumedown": 0x81,
	// 国际键（部分键盘支持）
	"intl1": 0x87, "intl2": 0x88, "intl3": 0x89, "intl4": 0x8a, "intl5": 0x8b,
	"intl6": 0x8c, "intl7": 0x8d, "intl8": 0x8e, "intl9": 0x8f,
	// 其它常见键
	"pausebreak": 0x48, "win": 0xe3, "menu": 0x65,
}

// KeyRequest 按键请求（简化版，去掉Response通道）
type KeyRequest struct {
	Key         string
	Duration    time.Duration
	ClientIP    string
	RequestTime time.Time
}

// KeyResponse 按键响应
type KeyResponse struct {
	Success bool
	Error   error
	Latency time.Duration
}

// TypeRequest 文本输入请求
type TypeRequest struct {
	Text string `json:"text"`
}

// Action 批量操作
type Action struct {
	Key      string `json:"key"`
	Duration int    `json:"duration,omitempty"`
}

type Keyboard struct {
	driver KeyboardDriver
	stats  *KeyboardStats
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// 移除 requestChan，改为直接并发处理
}

// KeyboardStats 统计信息
type KeyboardStats struct {
	mu                  sync.RWMutex
	TotalRequests       int64
	SuccessRequests     int64
	FailedRequests      int64
	RejectedRequests    int64
	AverageLatency      time.Duration
	LastRequestTime     time.Time
	CurrentlyProcessing int64 // 使用原子操作
	latencySum          time.Duration
	latencyCount        int64

	// 分阶段延迟统计
	ProcessLatency time.Duration   // 实际处理时间
	NetworkLatency time.Duration   // 网络传输时间
	LatencyHistory []LatencyRecord // 最近的延迟记录
}

// LatencyRecord 延迟记录
type LatencyRecord struct {
	Timestamp      time.Time     `json:"timestamp"`
	TotalLatency   time.Duration `json:"total_latency"`
	ProcessLatency time.Duration `json:"process_latency"`
	NetworkLatency time.Duration `json:"network_latency"`
}

func NewKeyboard(driver KeyboardDriver) *Keyboard {
	ctx, cancel := context.WithCancel(context.Background())
	k := &Keyboard{
		driver: driver,
		stats:  &KeyboardStats{},
		ctx:    ctx,
		cancel: cancel,
	}

	log.Printf("[KEYBOARD] 并发键盘处理器启动 - 直接并发处理，无队列")
	return k
}

// handleSingleRequest 处理单个按键请求（现在直接并发执行）
func (k *Keyboard) handleSingleRequest(req KeyRequest) {
	atomic.AddInt64(&k.stats.CurrentlyProcessing, 1)
	defer atomic.AddInt64(&k.stats.CurrentlyProcessing, -1)

	// 执行按键操作
	driverStartTime := time.Now()
	err := k.driver.Press(req.Key, req.Duration)
	processLatency := time.Since(driverStartTime)

	// 计算总延迟
	totalLatency := time.Since(req.RequestTime)
	networkLatency := totalLatency - processLatency

	// 更新统计信息
	k.updateStatsWithDetails(err == nil, totalLatency, processLatency, networkLatency, false)

	if err != nil {
		log.Printf("[KEYBOARD] 按键失败: %s (%v) - %s | 总延迟:%v 处理:%v",
			req.Key, err, req.ClientIP, totalLatency, processLatency)
	} else {
		log.Printf("[KEYBOARD] 按键成功: %s - %s | 总延迟:%v 处理:%v",
			req.Key, req.ClientIP, totalLatency, processLatency)
	}
}

// GetStats 获取统计信息
func (k *Keyboard) GetStats() *KeyboardStats {
	k.stats.mu.RLock()
	defer k.stats.mu.RUnlock()

	// 复制延迟历史
	historyLen := len(k.stats.LatencyHistory)
	latencyHistory := make([]LatencyRecord, historyLen)
	copy(latencyHistory, k.stats.LatencyHistory)

	return &KeyboardStats{
		TotalRequests:       k.stats.TotalRequests,
		SuccessRequests:     k.stats.SuccessRequests,
		FailedRequests:      k.stats.FailedRequests,
		RejectedRequests:    k.stats.RejectedRequests,
		AverageLatency:      k.stats.AverageLatency,
		LastRequestTime:     k.stats.LastRequestTime,
		CurrentlyProcessing: atomic.LoadInt64(&k.stats.CurrentlyProcessing),
		latencySum:          k.stats.latencySum,
		latencyCount:        k.stats.latencyCount,

		// 分阶段延迟统计
		ProcessLatency: k.stats.ProcessLatency,
		NetworkLatency: k.stats.NetworkLatency,
		LatencyHistory: latencyHistory,
	}
}

// updateStats 更新统计信息
func (k *Keyboard) updateStats(success bool, latency time.Duration, rejected bool) {
	k.updateStatsWithDetails(success, latency, latency, 0, rejected)
}

// updateStatsWithDetails 更新详细统计信息
func (k *Keyboard) updateStatsWithDetails(success bool, totalLatency, processLatency, networkLatency time.Duration, rejected bool) {
	k.stats.mu.Lock()
	defer k.stats.mu.Unlock()

	k.stats.TotalRequests++
	k.stats.LastRequestTime = time.Now()

	if rejected {
		k.stats.RejectedRequests++
		return
	}

	if success {
		k.stats.SuccessRequests++
	} else {
		k.stats.FailedRequests++
	}

	// 更新平均延迟
	k.stats.latencySum += totalLatency
	k.stats.latencyCount++
	k.stats.AverageLatency = k.stats.latencySum / time.Duration(k.stats.latencyCount)

	// 更新分阶段延迟（简单平均）
	if k.stats.latencyCount == 1 {
		k.stats.ProcessLatency = processLatency
		k.stats.NetworkLatency = networkLatency
	} else {
		k.stats.ProcessLatency = (k.stats.ProcessLatency + processLatency) / 2
		k.stats.NetworkLatency = (k.stats.NetworkLatency + networkLatency) / 2
	}

	// 添加到历史记录（保持最近50条记录）
	record := LatencyRecord{
		Timestamp:      time.Now(),
		TotalLatency:   totalLatency,
		ProcessLatency: processLatency,
		NetworkLatency: networkLatency,
	}

	k.stats.LatencyHistory = append(k.stats.LatencyHistory, record)
	if len(k.stats.LatencyHistory) > 50 {
		k.stats.LatencyHistory = k.stats.LatencyHistory[1:]
	}
}

// PressHandler 按键处理接口（现在直接并发处理）
func (k *Keyboard) PressHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	key := strings.ToLower(r.URL.Query().Get("key"))
	durationStr := r.URL.Query().Get("duration")
	clientIP := r.RemoteAddr

	// 快速参数验证
	if key == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "按键参数不能为空", 400)
		return
	}

	if !k.driver.IsKeySupported(key) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "不支持的按键: "+key, 400)
		return
	}

	// 解析持续时间
	duration := 50 * time.Millisecond // 默认50ms
	if durationStr != "" {
		var durationMs int
		if _, err := fmt.Sscanf(durationStr, "%d", &durationMs); err == nil {
			duration = time.Duration(durationMs) * time.Millisecond
		}
	}

	// 创建请求
	req := KeyRequest{
		Key:         key,
		Duration:    duration,
		ClientIP:    clientIP,
		RequestTime: time.Now(),
	}

	// 直接启动goroutine并发处理，不使用队列
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		k.handleSingleRequest(req)
	}()

	// 立即返回，不等待处理完成
	io.WriteString(w, "processing")
}

// PressHandlerSync 同步按键处理接口（直接处理，不使用队列）
func (k *Keyboard) PressHandlerSync(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	key := strings.ToLower(r.URL.Query().Get("key"))
	durationStr := r.URL.Query().Get("duration")
	clientIP := r.RemoteAddr

	// 快速参数验证
	if key == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "按键参数不能为空", 400)
		return
	}

	if !k.driver.IsKeySupported(key) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "不支持的按键: "+key, 400)
		return
	}

	// 解析持续时间
	duration := 50 * time.Millisecond
	if durationStr != "" {
		var durationMs int
		if _, err := fmt.Sscanf(durationStr, "%d", &durationMs); err == nil {
			duration = time.Duration(durationMs) * time.Millisecond
		}
	}

	// 创建请求
	req := KeyRequest{
		Key:         key,
		Duration:    duration,
		ClientIP:    clientIP,
		RequestTime: time.Now(),
	}

	// 直接同步处理
	k.handleSingleRequest(req)

	// 根据最后的统计信息判断是否成功
	if k.stats.FailedRequests > 0 {
		http.Error(w, "按键处理失败", 500)
	} else {
		io.WriteString(w, "ok")
	}
}

// ActionsHandler 批量操作处理（并发处理版）
func (k *Keyboard) ActionsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := r.RemoteAddr

	var actions []Action
	if err := json.NewDecoder(r.Body).Decode(&actions); err != nil {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "JSON 解析失败", 400)
		return
	}

	if len(actions) == 0 {
		http.Error(w, "操作列表为空", 400)
		return
	}

	// 并发处理所有操作
	for _, act := range actions {
		key := strings.ToLower(act.Key)
		if !k.driver.IsKeySupported(key) {
			http.Error(w, "不支持的按键: "+act.Key, 400)
			return
		}

		duration := 50 * time.Millisecond
		if act.Duration > 0 {
			duration = time.Duration(act.Duration) * time.Millisecond
		}

		req := KeyRequest{
			Key:         key,
			Duration:    duration,
			ClientIP:    clientIP,
			RequestTime: time.Now(),
		}

		// 每个操作都启动独立的goroutine
		k.wg.Add(1)
		go func(r KeyRequest) {
			defer k.wg.Done()
			k.handleSingleRequest(r)
		}(req)
	}

	io.WriteString(w, "processing")
	log.Printf("[ACTIONS] 批量操作并发处理: %d个操作 - %s", len(actions), clientIP)
}

// TypeHandler 文本输入处理（并发处理版）
func (k *Keyboard) TypeHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := r.RemoteAddr

	var req TypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "JSON 解析失败", 400)
		return
	}

	if req.Text == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "文本内容不能为空", 400)
		return
	}

	// 将文本转换为按键序列，并发处理
	for _, char := range req.Text {
		key := strings.ToLower(string(char))
		if k.driver.IsKeySupported(key) {
			keyReq := KeyRequest{
				Key:         key,
				Duration:    50 * time.Millisecond,
				ClientIP:    clientIP,
				RequestTime: time.Now(),
			}

			// 每个字符都启动独立的goroutine
			k.wg.Add(1)
			go func(r KeyRequest) {
				defer k.wg.Done()
				k.handleSingleRequest(r)
			}(keyReq)
		}
	}

	io.WriteString(w, "processing")
	log.Printf("[TYPE] 文本输入并发处理 - 客户端: %s, 字符数: %d", clientIP, len(req.Text))
}

// KeyDownHandler 按键按下接口
func (k *Keyboard) KeyDownHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	key := strings.ToLower(r.URL.Query().Get("key"))

	if key == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "按键参数不能为空", 400)
		return
	}

	if !k.driver.IsKeySupported(key) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "不支持的按键: "+key, 400)
		return
	}

	err := k.driver.KeyDown(key)
	latency := time.Since(startTime)
	k.updateStats(err == nil, latency, false)
	if err != nil {
		http.Error(w, "按键按下失败: "+err.Error(), 500)
		return
	}
	io.WriteString(w, "ok")
}

// KeyUpHandler 按键释放接口
func (k *Keyboard) KeyUpHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	key := strings.ToLower(r.URL.Query().Get("key"))

	if key == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "按键参数不能为空", 400)
		return
	}

	if !k.driver.IsKeySupported(key) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		http.Error(w, "不支持的按键: "+key, 400)
		return
	}

	err := k.driver.KeyUp(key)
	latency := time.Since(startTime)
	k.updateStats(err == nil, latency, false)
	if err != nil {
		http.Error(w, "按键释放失败: "+err.Error(), 500)
		return
	}
	io.WriteString(w, "ok")
}

// StatsHandler 统计信息接口
func (k *Keyboard) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := k.GetStats()

	// 计算成功率，避免除零错误
	var successRate float64
	if stats.TotalRequests > 0 {
		successRate = float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100
	}

	// 格式化最后请求时间
	var lastRequestTime string
	if !stats.LastRequestTime.IsZero() {
		lastRequestTime = stats.LastRequestTime.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"total_requests":       stats.TotalRequests,
		"success_requests":     stats.SuccessRequests,
		"failed_requests":      stats.FailedRequests,
		"rejected_requests":    stats.RejectedRequests,
		"average_latency_ms":   stats.AverageLatency.Milliseconds(),
		"last_request_time":    lastRequestTime,
		"currently_processing": stats.CurrentlyProcessing,
		"success_rate":         successRate,

		// 分阶段延迟统计
		"latency_breakdown": map[string]interface{}{
			"process_ms": stats.ProcessLatency.Milliseconds(),
			"network_ms": stats.NetworkLatency.Milliseconds(),
		},
		"latency_history": stats.LatencyHistory,
	})

	if err != nil {
		log.Printf("[STATS] JSON编码失败: %v", err)
		http.Error(w, "统计信息编码失败", 500)
		return
	}
}

// Close 关闭键盘服务
func (k *Keyboard) Close() error {
	log.Printf("[KEYBOARD] 关闭键盘服务")
	k.cancel()
	k.wg.Wait() // 等待所有并发任务完成
	return k.driver.Close()
}
