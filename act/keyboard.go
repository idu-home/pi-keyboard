package act

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
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

type Keyboard struct {
	driver     KeyboardDriver
	globalLock sync.Mutex
	stats      *KeyboardStats
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
	CurrentlyProcessing bool
	latencySum          time.Duration
	latencyCount        int64
}

func NewKeyboard(driver KeyboardDriver) *Keyboard {
	return &Keyboard{
		driver: driver,
		stats:  &KeyboardStats{},
	}
}

// GetStats 获取统计信息
func (k *Keyboard) GetStats() *KeyboardStats {
	k.stats.mu.RLock()
	defer k.stats.mu.RUnlock()

	stats := *k.stats
	return &stats
}

// updateStats 更新统计信息
func (k *Keyboard) updateStats(success bool, latency time.Duration, rejected bool) {
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
	k.stats.latencySum += latency
	k.stats.latencyCount++
	k.stats.AverageLatency = k.stats.latencySum / time.Duration(k.stats.latencyCount)
}

// setProcessing 设置处理状态
func (k *Keyboard) setProcessing(processing bool) {
	k.stats.mu.Lock()
	defer k.stats.mu.Unlock()
	k.stats.CurrentlyProcessing = processing
}

// /press?key=a&duration=100
func (k *Keyboard) PressHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	key := strings.ToLower(r.URL.Query().Get("key"))
	durationStr := r.URL.Query().Get("duration")
	clientIP := r.RemoteAddr

	log.Printf("[PRESS] 开始处理请求 - 客户端: %s, 按键: %s, 持续时间: %s", clientIP, key, durationStr)

	if !k.tryLock() {
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		log.Printf("[PRESS] 请求被拒绝 - 客户端: %s, 按键: %s, 原因: 有其他请求正在执行, 延迟: %v", clientIP, key, latency)
		http.Error(w, "有其他请求正在执行，请稍后重试", http.StatusTooManyRequests)
		return
	}
	defer k.globalLock.Unlock()

	k.setProcessing(true)
	defer k.setProcessing(false)

	if key == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		log.Printf("[PRESS] 请求失败 - 客户端: %s, 原因: 按键参数为空, 延迟: %v", clientIP, latency)
		http.Error(w, "按键参数不能为空", 400)
		return
	}

	if !k.driver.IsKeySupported(key) {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		log.Printf("[PRESS] 请求失败 - 客户端: %s, 按键: %s, 原因: 不支持的按键, 延迟: %v", clientIP, key, latency)
		http.Error(w, "不支持的按键: "+key, 400)
		return
	}

	duration := 50 // ms
	if durationStr != "" {
		if _, err := fmt.Sscanf(durationStr, "%d", &duration); err != nil {
			log.Printf("[PRESS] 警告 - 客户端: %s, 按键: %s, 持续时间解析失败: %v, 使用默认值: %d", clientIP, key, err, duration)
		}
	}

	log.Printf("[PRESS] 执行按键操作 - 客户端: %s, 按键: %s, 持续时间: %dms", clientIP, key, duration)

	pressStart := time.Now()
	err := k.driver.Press(key, time.Duration(duration)*time.Millisecond)
	pressLatency := time.Since(pressStart)
	totalLatency := time.Since(startTime)

	if err != nil {
		k.updateStats(false, totalLatency, false)
		log.Printf("[PRESS] 按键执行失败 - 客户端: %s, 按键: %s, 错误: %v, 按键延迟: %v, 总延迟: %v", clientIP, key, err, pressLatency, totalLatency)
		http.Error(w, err.Error(), 500)
		return
	}

	k.updateStats(true, totalLatency, false)
	log.Printf("[PRESS] 按键执行成功 - 客户端: %s, 按键: %s, 按键延迟: %v, 总延迟: %v", clientIP, key, pressLatency, totalLatency)

	io.WriteString(w, "ok")
}

type Action struct {
	Key      string `json:"key"`
	Duration int    `json:"duration,omitempty"`
}

// TypeRequest 文本输入请求
type TypeRequest struct {
	Text string `json:"text"`
}

// /actions 批量指令接口，每个元素只需 key/duration
func (k *Keyboard) ActionsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := r.RemoteAddr

	log.Printf("[ACTIONS] 开始处理批量操作请求 - 客户端: %s", clientIP)

	if !k.tryLock() {
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		log.Printf("[ACTIONS] 请求被拒绝 - 客户端: %s, 原因: 有其他请求正在执行, 延迟: %v", clientIP, latency)
		http.Error(w, "有其他请求正在执行，请稍后重试", http.StatusTooManyRequests)
		return
	}
	defer k.globalLock.Unlock()

	k.setProcessing(true)
	defer k.setProcessing(false)

	var actions []Action
	if err := json.NewDecoder(r.Body).Decode(&actions); err != nil {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		log.Printf("[ACTIONS] JSON解析失败 - 客户端: %s, 错误: %v, 延迟: %v", clientIP, err, latency)
		http.Error(w, "JSON 解析失败", 400)
		return
	}

	log.Printf("[ACTIONS] 解析到 %d 个操作 - 客户端: %s", len(actions), clientIP)

	for i, act := range actions {
		key := strings.ToLower(act.Key)
		if !k.driver.IsKeySupported(key) {
			latency := time.Since(startTime)
			k.updateStats(false, latency, false)
			log.Printf("[ACTIONS] 操作失败 - 客户端: %s, 第%d个操作, 按键: %s, 原因: 不支持的按键, 延迟: %v", clientIP, i+1, act.Key, latency)
			http.Error(w, "不支持的按键: "+act.Key, 400)
			return
		}

		dur := act.Duration
		if dur <= 0 {
			dur = 50
		}

		log.Printf("[ACTIONS] 执行第%d个操作 - 客户端: %s, 按键: %s, 持续时间: %dms", i+1, clientIP, key, dur)

		pressStart := time.Now()
		if err := k.driver.Press(key, time.Duration(dur)*time.Millisecond); err != nil {
			latency := time.Since(startTime)
			pressLatency := time.Since(pressStart)
			k.updateStats(false, latency, false)
			log.Printf("[ACTIONS] 第%d个操作执行失败 - 客户端: %s, 按键: %s, 错误: %v, 按键延迟: %v, 总延迟: %v", i+1, clientIP, key, err, pressLatency, latency)
			http.Error(w, err.Error(), 500)
			return
		}

		pressLatency := time.Since(pressStart)
		log.Printf("[ACTIONS] 第%d个操作执行成功 - 客户端: %s, 按键: %s, 按键延迟: %v", i+1, clientIP, key, pressLatency)
	}

	totalLatency := time.Since(startTime)
	k.updateStats(true, totalLatency, false)
	log.Printf("[ACTIONS] 批量操作完成 - 客户端: %s, 总操作数: %d, 总延迟: %v", clientIP, len(actions), totalLatency)

	io.WriteString(w, "ok")
}

// /type 文本输入接口
func (k *Keyboard) TypeHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := r.RemoteAddr

	log.Printf("[TYPE] 开始处理文本输入请求 - 客户端: %s", clientIP)

	if !k.tryLock() {
		latency := time.Since(startTime)
		k.updateStats(false, latency, true)
		log.Printf("[TYPE] 请求被拒绝 - 客户端: %s, 原因: 有其他请求正在执行, 延迟: %v", clientIP, latency)
		http.Error(w, "有其他请求正在执行，请稍后重试", http.StatusTooManyRequests)
		return
	}
	defer k.globalLock.Unlock()

	k.setProcessing(true)
	defer k.setProcessing(false)

	var req TypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		log.Printf("[TYPE] JSON解析失败 - 客户端: %s, 错误: %v, 延迟: %v", clientIP, err, latency)
		http.Error(w, "JSON 解析失败", 400)
		return
	}

	if req.Text == "" {
		latency := time.Since(startTime)
		k.updateStats(false, latency, false)
		log.Printf("[TYPE] 请求失败 - 客户端: %s, 原因: 文本内容为空, 延迟: %v", clientIP, latency)
		http.Error(w, "文本内容不能为空", 400)
		return
	}

	log.Printf("[TYPE] 开始输入文本 - 客户端: %s, 文本长度: %d, 内容: %q", clientIP, len(req.Text), req.Text)

	typeStart := time.Now()
	err := k.driver.Type(req.Text)
	typeLatency := time.Since(typeStart)
	totalLatency := time.Since(startTime)

	if err != nil {
		k.updateStats(false, totalLatency, false)
		log.Printf("[TYPE] 文本输入失败 - 客户端: %s, 文本: %q, 错误: %v, 输入延迟: %v, 总延迟: %v", clientIP, req.Text, err, typeLatency, totalLatency)
		http.Error(w, err.Error(), 500)
		return
	}

	k.updateStats(true, totalLatency, false)
	log.Printf("[TYPE] 文本输入成功 - 客户端: %s, 文本: %q, 输入延迟: %v, 总延迟: %v", clientIP, req.Text, typeLatency, totalLatency)

	io.WriteString(w, "ok")
}

// StatsHandler 统计信息接口
func (k *Keyboard) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := k.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_requests":       stats.TotalRequests,
		"success_requests":     stats.SuccessRequests,
		"failed_requests":      stats.FailedRequests,
		"rejected_requests":    stats.RejectedRequests,
		"average_latency_ms":   stats.AverageLatency.Milliseconds(),
		"last_request_time":    stats.LastRequestTime.Format(time.RFC3339),
		"currently_processing": stats.CurrentlyProcessing,
		"success_rate":         float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100,
	})
}

// 非阻塞尝试加锁
func (k *Keyboard) tryLock() bool {
	locked := make(chan struct{}, 1)
	go func() {
		k.globalLock.Lock()
		locked <- struct{}{}
	}()
	select {
	case <-locked:
		return true
	case <-time.After(10 * time.Millisecond):
		return false
	}
}
