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
	"a": 0x04, "b": 0x05, "c": 0x06, "d": 0x07, "e": 0x08,
	"f": 0x09, "g": 0x0a, "h": 0x0b, "i": 0x0c, "j": 0x0d,
	"k": 0x0e, "l": 0x0f, "m": 0x10, "n": 0x11, "o": 0x12,
	"p": 0x13, "q": 0x14, "r": 0x15, "s": 0x16, "t": 0x17,
	"u": 0x18, "v": 0x19, "w": 0x1a, "x": 0x1b, "y": 0x1c, "z": 0x1d,
	"1": 0x1e, "2": 0x1f, "3": 0x20, "4": 0x21, "5": 0x22,
	"6": 0x23, "7": 0x24, "8": 0x25, "9": 0x26, "0": 0x27,
	"enter": 0x28, "esc": 0x29, "backspace": 0x2a, "tab": 0x2b, "space": 0x2c,
	"shift": 0xe1, // 左 shift，右 shift 可扩展
}

// KeyRequest 按键请求
type KeyRequest struct {
	Key         string
	Duration    time.Duration
	ClientIP    string
	Response    chan KeyResponse
	RequestTime time.Time // 请求创建时间
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
	driver      KeyboardDriver
	stats       *KeyboardStats
	requestChan chan KeyRequest
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
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
	QueueLatency   time.Duration   // 队列等待时间
	ProcessLatency time.Duration   // 实际处理时间
	NetworkLatency time.Duration   // 网络传输时间
	LatencyHistory []LatencyRecord // 最近的延迟记录
}

// LatencyRecord 延迟记录
type LatencyRecord struct {
	Timestamp      time.Time     `json:"timestamp"`
	TotalLatency   time.Duration `json:"total_latency"`
	QueueLatency   time.Duration `json:"queue_latency"`
	ProcessLatency time.Duration `json:"process_latency"`
	NetworkLatency time.Duration `json:"network_latency"`
}

func NewKeyboard(driver KeyboardDriver) *Keyboard {
	ctx, cancel := context.WithCancel(context.Background())
	k := &Keyboard{
		driver:      driver,
		stats:       &KeyboardStats{},
		requestChan: make(chan KeyRequest, 100), // 缓冲队列
		ctx:         ctx,
		cancel:      cancel,
	}

	// 启动处理协程
	k.wg.Add(1)
	go k.processRequests()

	log.Printf("[KEYBOARD] 异步键盘处理器启动 - 队列大小: 100")
	return k
}

// processRequests 异步处理按键请求
func (k *Keyboard) processRequests() {
	defer k.wg.Done()
	log.Printf("[KEYBOARD] 按键处理协程启动")

	for {
		select {
		case req := <-k.requestChan:
			k.handleSingleRequest(req)
		case <-k.ctx.Done():
			log.Printf("[KEYBOARD] 按键处理协程停止")
			return
		}
	}
}

// handleSingleRequest 处理单个按键请求
func (k *Keyboard) handleSingleRequest(req KeyRequest) {
	processStartTime := time.Now()
	atomic.AddInt64(&k.stats.CurrentlyProcessing, 1)
	defer atomic.AddInt64(&k.stats.CurrentlyProcessing, -1)

	// 计算队列等待时间
	queueLatency := processStartTime.Sub(req.RequestTime)

	// 执行按键操作
	driverStartTime := time.Now()
	err := k.driver.Press(req.Key, req.Duration)
	processLatency := time.Since(driverStartTime)

	// 计算总延迟
	totalLatency := time.Since(req.RequestTime)
	networkLatency := totalLatency - queueLatency - processLatency

	// 更新统计信息
	k.updateStatsWithDetails(err == nil, totalLatency, queueLatency, processLatency, networkLatency, false)

	// 发送响应
	response := KeyResponse{
		Success: err == nil,
		Error:   err,
		Latency: totalLatency,
	}

	select {
	case req.Response <- response:
	case <-time.After(1 * time.Second):
		log.Printf("[KEYBOARD] 响应超时: %s - %s", req.Key, req.ClientIP)
	}

	if err != nil {
		log.Printf("[KEYBOARD] 按键失败: %s (%v) - %s | 总延迟:%v 队列:%v 处理:%v",
			req.Key, err, req.ClientIP, totalLatency, queueLatency, processLatency)
	} else {
		log.Printf("[KEYBOARD] 按键成功: %s - %s | 总延迟:%v 队列:%v 处理:%v",
			req.Key, req.ClientIP, totalLatency, queueLatency, processLatency)
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
		QueueLatency:   k.stats.QueueLatency,
		ProcessLatency: k.stats.ProcessLatency,
		NetworkLatency: k.stats.NetworkLatency,
		LatencyHistory: latencyHistory,
	}
}

// updateStats 更新统计信息
func (k *Keyboard) updateStats(success bool, latency time.Duration, rejected bool) {
	k.updateStatsWithDetails(success, latency, 0, latency, 0, rejected)
}

// updateStatsWithDetails 更新详细统计信息
func (k *Keyboard) updateStatsWithDetails(success bool, totalLatency, queueLatency, processLatency, networkLatency time.Duration, rejected bool) {
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
		k.stats.QueueLatency = queueLatency
		k.stats.ProcessLatency = processLatency
		k.stats.NetworkLatency = networkLatency
	} else {
		k.stats.QueueLatency = (k.stats.QueueLatency + queueLatency) / 2
		k.stats.ProcessLatency = (k.stats.ProcessLatency + processLatency) / 2
		k.stats.NetworkLatency = (k.stats.NetworkLatency + networkLatency) / 2
	}

	// 添加到历史记录（保持最近50条记录）
	record := LatencyRecord{
		Timestamp:      time.Now(),
		TotalLatency:   totalLatency,
		QueueLatency:   queueLatency,
		ProcessLatency: processLatency,
		NetworkLatency: networkLatency,
	}

	k.stats.LatencyHistory = append(k.stats.LatencyHistory, record)
	if len(k.stats.LatencyHistory) > 50 {
		k.stats.LatencyHistory = k.stats.LatencyHistory[1:]
	}
}

// PressHandler 按键处理接口（异步）
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

	// 检查队列是否满
	select {
	case k.requestChan <- KeyRequest{
		Key:         key,
		Duration:    duration,
		ClientIP:    clientIP,
		Response:    make(chan KeyResponse, 1),
		RequestTime: time.Now(),
	}:
		// 请求已加入队列，立即返回成功
		io.WriteString(w, "queued")

	default:
		// 队列已满，拒绝请求
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		log.Printf("[PRESS] 队列满，拒绝请求: %s - %s", key, clientIP)
		http.Error(w, "服务器繁忙，请稍后重试", http.StatusTooManyRequests)
	}
}

// PressHandlerSync 同步按键处理接口（用于需要确认的场景）
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

	// 创建响应通道
	responseChan := make(chan KeyResponse, 1)

	// 发送请求
	select {
	case k.requestChan <- KeyRequest{
		Key:         key,
		Duration:    duration,
		ClientIP:    clientIP,
		Response:    responseChan,
		RequestTime: time.Now(),
	}:
		// 等待响应
		select {
		case response := <-responseChan:
			if response.Success {
				io.WriteString(w, "ok")
			} else {
				http.Error(w, response.Error.Error(), 500)
			}
		case <-time.After(10 * time.Second):
			http.Error(w, "请求超时", http.StatusGatewayTimeout)
		}

	default:
		// 队列已满
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		http.Error(w, "服务器繁忙，请稍后重试", http.StatusTooManyRequests)
	}
}

// ActionsHandler 批量操作处理（优化版）
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

	// 检查队列容量
	if len(k.requestChan)+len(actions) > cap(k.requestChan) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		http.Error(w, "批量操作过多，服务器繁忙", http.StatusTooManyRequests)
		return
	}

	// 批量提交请求
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

		k.requestChan <- KeyRequest{
			Key:         key,
			Duration:    duration,
			ClientIP:    clientIP,
			Response:    make(chan KeyResponse, 1),
			RequestTime: time.Now(),
		}
	}

	io.WriteString(w, "queued")
	log.Printf("[ACTIONS] 批量操作入队: %d个操作 - %s", len(actions), clientIP)
}

// TypeHandler 文本输入处理
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

	// 将文本转换为按键序列
	var keyRequests []KeyRequest
	for _, char := range req.Text {
		key := strings.ToLower(string(char))
		if k.driver.IsKeySupported(key) {
			keyRequests = append(keyRequests, KeyRequest{
				Key:         key,
				Duration:    50 * time.Millisecond,
				ClientIP:    clientIP,
				Response:    make(chan KeyResponse, 1),
				RequestTime: time.Now(),
			})
		}
	}

	// 检查队列容量
	if len(k.requestChan)+len(keyRequests) > cap(k.requestChan) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		http.Error(w, "文本过长，服务器繁忙", http.StatusTooManyRequests)
		return
	}

	// 批量提交
	for _, keyReq := range keyRequests {
		k.requestChan <- keyReq
	}

	io.WriteString(w, "queued")
	log.Printf("[TYPE] 文本输入已入队 - 客户端: %s, 字符数: %d", clientIP, len(keyRequests))
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
		"queue_length":         len(k.requestChan),
		"queue_capacity":       cap(k.requestChan),

		// 分阶段延迟统计
		"latency_breakdown": map[string]interface{}{
			"queue_ms":   stats.QueueLatency.Milliseconds(),
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
	k.wg.Wait()
	close(k.requestChan)
	return k.driver.Close()
}
