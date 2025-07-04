package act

import (
	"encoding/json"
	"fmt"
	"io"
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
}

func NewKeyboard(driver KeyboardDriver) *Keyboard {
	return &Keyboard{driver: driver}
}

// 移除不安全的 keydown/keyup 方法，统一使用 Press 原子操作

// /press?key=a&duration=100
func (k *Keyboard) PressHandler(w http.ResponseWriter, r *http.Request) {
	if !k.tryLock() {
		http.Error(w, "有其他请求正在执行，请稍后重试", http.StatusTooManyRequests)
		return
	}
	defer k.globalLock.Unlock()

	key := strings.ToLower(r.URL.Query().Get("key"))
	if !k.driver.IsKeySupported(key) {
		http.Error(w, "不支持的按键", 400)
		return
	}
	duration := 50 // ms
	if d := r.URL.Query().Get("duration"); d != "" {
		fmt.Sscanf(d, "%d", &duration)
	}

	if err := k.driver.Press(key, time.Duration(duration)*time.Millisecond); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
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
	if !k.tryLock() {
		http.Error(w, "有其他请求正在执行，请稍后重试", http.StatusTooManyRequests)
		return
	}
	defer k.globalLock.Unlock()

	var actions []Action
	if err := json.NewDecoder(r.Body).Decode(&actions); err != nil {
		http.Error(w, "JSON 解析失败", 400)
		return
	}
	for _, act := range actions {
		key := strings.ToLower(act.Key)
		if !k.driver.IsKeySupported(key) {
			http.Error(w, "不支持的按键: "+act.Key, 400)
			return
		}
		dur := act.Duration
		if dur <= 0 {
			dur = 50
		}

		if err := k.driver.Press(key, time.Duration(dur)*time.Millisecond); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	io.WriteString(w, "ok")
}

// /type 文本输入接口
func (k *Keyboard) TypeHandler(w http.ResponseWriter, r *http.Request) {
	if !k.tryLock() {
		http.Error(w, "有其他请求正在执行，请稍后重试", http.StatusTooManyRequests)
		return
	}
	defer k.globalLock.Unlock()

	var req TypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON 解析失败", 400)
		return
	}

	if req.Text == "" {
		http.Error(w, "文本内容不能为空", 400)
		return
	}

	if err := k.driver.Type(req.Text); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	io.WriteString(w, "ok")
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
