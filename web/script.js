class PiKeyboard {
    constructor() {
        this.statusElement = document.getElementById('status');
        this.textInput = document.getElementById('textInput');
        this.sendTextButton = document.getElementById('sendText');
        this.refreshStatsButton = document.getElementById('refreshStats');
        this.keys = document.querySelectorAll('.key');
        
        // 统计信息元素
        this.statsElements = {
            totalRequests: document.getElementById('totalRequests'),
            successRate: document.getElementById('successRate'),
            avgLatency: document.getElementById('avgLatency'),
            processingStatus: document.getElementById('processingStatus')
        };
        
        this.isProcessing = false;
        this.apiBase = window.location.origin;
        this.statsInterval = null;
        
        this.init();
    }
    
    init() {
        // 绑定键盘按键事件
        this.keys.forEach(key => {
            key.addEventListener('click', (e) => {
                e.preventDefault();
                this.handleKeyPress(key);
            });
            
            // 添加触摸事件支持
            key.addEventListener('touchstart', (e) => {
                e.preventDefault();
                key.classList.add('pressed');
            });
            
            key.addEventListener('touchend', (e) => {
                e.preventDefault();
                key.classList.remove('pressed');
            });
        });
        
        // 绑定发送文本按钮事件
        this.sendTextButton.addEventListener('click', () => {
            this.sendText();
        });
        
        // 绑定刷新统计按钮事件
        this.refreshStatsButton.addEventListener('click', () => {
            this.refreshStats();
        });
        
        // 绑定文本输入框回车事件
        this.textInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                this.sendText();
            }
        });
        
        // 防止页面缩放
        document.addEventListener('gesturestart', (e) => {
            e.preventDefault();
        });
        
        // 防止双击缩放
        let lastTouchEnd = 0;
        document.addEventListener('touchend', (e) => {
            const now = (new Date()).getTime();
            if (now - lastTouchEnd <= 300) {
                e.preventDefault();
            }
            lastTouchEnd = now;
        }, false);
        
        // 初始化统计信息
        this.refreshStats();
        
        // 设置自动刷新统计信息
        this.startStatsAutoRefresh();
        
        this.updateStatus('就绪', 'success');
    }
    
    // 开始自动刷新统计信息
    startStatsAutoRefresh() {
        this.statsInterval = setInterval(() => {
            this.refreshStats(true); // 静默刷新
        }, 3000); // 每3秒刷新一次
    }
    
    // 停止自动刷新
    stopStatsAutoRefresh() {
        if (this.statsInterval) {
            clearInterval(this.statsInterval);
            this.statsInterval = null;
        }
    }
    
    // 刷新统计信息
    async refreshStats(silent = false) {
        try {
            if (!silent) {
                this.refreshStatsButton.textContent = '刷新中...';
                this.refreshStatsButton.disabled = true;
            }
            
            const stats = await this.getStats();
            this.updateStatsDisplay(stats);
            
        } catch (error) {
            console.error('获取统计信息失败:', error);
            if (!silent) {
                this.updateStatus('获取统计信息失败', 'error');
            }
        } finally {
            if (!silent) {
                this.refreshStatsButton.textContent = '刷新统计';
                this.refreshStatsButton.disabled = false;
            }
        }
    }
    
    // 更新统计信息显示
    updateStatsDisplay(stats) {
        this.statsElements.totalRequests.textContent = stats.total_requests || 0;
        
        const successRate = stats.success_rate || 0;
        this.statsElements.successRate.textContent = `${successRate.toFixed(1)}%`;
        this.statsElements.successRate.className = 'stat-value ' + 
            (successRate >= 90 ? 'success' : successRate >= 70 ? 'warning' : 'error');
        
        const avgLatency = stats.average_latency_ms || 0;
        this.statsElements.avgLatency.textContent = `${avgLatency}ms`;
        this.statsElements.avgLatency.className = 'stat-value ' + 
            (avgLatency <= 100 ? 'success' : avgLatency <= 500 ? 'warning' : 'error');
        
        const processing = stats.currently_processing;
        this.statsElements.processingStatus.textContent = processing ? '处理中' : '空闲';
        this.statsElements.processingStatus.className = 'stat-value ' + 
            (processing ? 'warning' : 'success');
    }
    
    async handleKeyPress(keyElement) {
        if (this.isProcessing) {
            this.updateStatus('请等待上一个操作完成', 'error');
            return;
        }
        
        const keyValue = keyElement.dataset.key;
        if (!keyValue) return;
        
        // 添加按键动画
        keyElement.classList.add('animate');
        setTimeout(() => {
            keyElement.classList.remove('animate');
        }, 150);
        
        const startTime = performance.now();
        
        try {
            this.isProcessing = true;
            this.updateStatus(`按下 ${keyValue.toUpperCase()}`, 'loading');
            
            await this.pressKey(keyValue);
            
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            this.updateStatus(`${keyValue.toUpperCase()} 按键成功 (${latency}ms)`, 'success');
            
        } catch (error) {
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            console.error('按键失败:', error);
            this.updateStatus(`按键失败: ${error.message} (${latency}ms)`, 'error');
        } finally {
            this.isProcessing = false;
            // 2秒后恢复就绪状态
            setTimeout(() => {
                if (!this.isProcessing) {
                    this.updateStatus('就绪', 'success');
                }
            }, 2000);
        }
    }
    
    async sendText() {
        const text = this.textInput.value.trim();
        if (!text) {
            this.updateStatus('请输入要发送的文本', 'error');
            return;
        }
        
        if (this.isProcessing) {
            this.updateStatus('请等待上一个操作完成', 'error');
            return;
        }
        
        const startTime = performance.now();
        
        try {
            this.isProcessing = true;
            this.updateStatus('正在发送文本...', 'loading');
            
            await this.typeText(text);
            
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            this.updateStatus(`文本发送成功 (${latency}ms): "${text}"`, 'success');
            this.textInput.value = '';
            
        } catch (error) {
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            console.error('发送文本失败:', error);
            this.updateStatus(`发送失败 (${latency}ms): ${error.message}`, 'error');
        } finally {
            this.isProcessing = false;
            // 3秒后恢复就绪状态
            setTimeout(() => {
                if (!this.isProcessing) {
                    this.updateStatus('就绪', 'success');
                }
            }, 3000);
        }
    }
    
    async pressKey(key, duration = 100) {
        const response = await fetch(`${this.apiBase}/press?key=${encodeURIComponent(key)}&duration=${duration}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP ${response.status}`);
        }
        
        return await response.text();
    }
    
    async typeText(text) {
        const response = await fetch(`${this.apiBase}/type`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ text })
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP ${response.status}`);
        }
        
        return await response.text();
    }
    
    async getStats() {
        const response = await fetch(`${this.apiBase}/stats`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP ${response.status}`);
        }
        
        return await response.json();
    }
    
    async sendActions(actions) {
        const response = await fetch(`${this.apiBase}/actions`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(actions)
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP ${response.status}`);
        }
        
        return await response.text();
    }
    
    updateStatus(message, type = 'success') {
        this.statusElement.textContent = message;
        this.statusElement.className = `status-display ${type}`;
        
        // 添加加载动画
        if (type === 'loading') {
            const spinner = document.createElement('span');
            spinner.className = 'loading-spinner';
            this.statusElement.insertBefore(spinner, this.statusElement.firstChild);
        }
    }
    
    // 页面卸载时清理资源
    cleanup() {
        this.stopStatsAutoRefresh();
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    const keyboard = new PiKeyboard();
    
    // 页面卸载时清理资源
    window.addEventListener('beforeunload', () => {
        keyboard.cleanup();
    });
});

// 添加一些实用的键盘快捷键
document.addEventListener('keydown', (e) => {
    // 阻止某些默认行为
    if (e.key === 'F5' || (e.ctrlKey && e.key === 'r')) {
        // 允许刷新
        return;
    }
    
    // 阻止其他可能干扰的快捷键
    if (e.key === 'F11' || e.key === 'F12') {
        e.preventDefault();
    }
});

// PWA 支持检测
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        // 这里可以注册 Service Worker 来支持离线使用
        console.log('支持 Service Worker，可以添加 PWA 功能');
    });
}

// 网络状态检测
window.addEventListener('online', () => {
    console.log('网络连接已恢复');
});

window.addEventListener('offline', () => {
    console.log('网络连接已断开');
}); 