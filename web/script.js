class PiKeyboard {
    constructor() {
        this.statusElement = document.getElementById('status');
        this.textInput = document.getElementById('textInput');
        this.sendTextButton = document.getElementById('sendText');
        this.refreshStatsButton = document.getElementById('refreshStats');
        this.debugToggleButton = document.getElementById('toggleDebug');
        this.keys = document.querySelectorAll('.key');
        
        // 统计信息元素
        this.statsElements = {
            totalRequests: document.getElementById('totalRequests'),
            successRate: document.getElementById('successRate'),
            avgLatency: document.getElementById('avgLatency'),
            processingStatus: document.getElementById('processingStatus'),
            // 延迟分析元素
            queueLatency: document.getElementById('queueLatency'),
            processLatency: document.getElementById('processLatency'),
            networkLatency: document.getElementById('networkLatency')
        };
        
        // 延迟图表
        this.latencyChart = null;
        this.latencyHistory = [];
        
        this.apiBase = window.location.origin;
        this.statsInterval = null;
        this.requestCount = 0;
        this.debugLogVisible = false;
        
        // 添加日志系统
        this.enableDebugLog();
        
        this.init();
    }
    
    // 启用调试日志
    enableDebugLog() {
        this.log('🚀 Pi Keyboard 初始化');
        this.log(`📍 API Base URL: ${this.apiBase}`);
        this.log(`🌐 User Agent: ${navigator.userAgent}`);
        this.log(`📱 Screen Size: ${window.screen.width}x${window.screen.height}`);
        this.log(`🔗 当前页面URL: ${window.location.href}`);
        this.log(`🌍 网络状态: ${navigator.onLine ? '在线' : '离线'}`);
        this.log(`🕐 页面加载时间: ${new Date().toLocaleString()}`);
        
        // 监听网络状态
        window.addEventListener('online', () => {
            this.log('🌐 网络连接已恢复');
        });
        
        window.addEventListener('offline', () => {
            this.log('❌ 网络连接已断开');
        });
        
        // 监听页面可见性变化
        document.addEventListener('visibilitychange', () => {
            this.log(`👁️ 页面可见性变化: ${document.visibilityState}`);
        });
    }
    
    // 日志函数
    log(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const logMessage = `[${timestamp}] ${message}`;
        
        console.log(logMessage);
        
        // 在状态栏显示重要日志
        if (type === 'error' || type === 'warning') {
            this.updateStatus(message, type);
        }
        
        // 可选：在页面上显示日志（用于移动端调试）
        this.showDebugLog(logMessage, type);
    }
    
    // 在页面上显示调试日志
    showDebugLog(message, type) {
        // 创建或获取日志容器
        let logContainer = document.getElementById('debug-log');
        if (!logContainer) {
            logContainer = document.createElement('div');
            logContainer.id = 'debug-log';
            logContainer.style.cssText = `
                position: fixed;
                top: 10px;
                right: 10px;
                width: 300px;
                max-height: 200px;
                overflow-y: auto;
                background: rgba(0,0,0,0.9);
                color: white;
                font-size: 11px;
                padding: 8px;
                border-radius: 8px;
                z-index: 1000;
                display: none;
                font-family: monospace;
                border: 1px solid #333;
                box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            `;
            document.body.appendChild(logContainer);
        }
        
        const logEntry = document.createElement('div');
        logEntry.textContent = message;
        logEntry.style.marginBottom = '2px';
        
        if (type === 'error') logEntry.style.color = '#ff6b6b';
        if (type === 'warning') logEntry.style.color = '#feca57';
        if (type === 'success') logEntry.style.color = '#48dbfb';
        
        logContainer.appendChild(logEntry);
        
        // 保持最新的20条日志
        while (logContainer.children.length > 20) {
            logContainer.removeChild(logContainer.firstChild);
        }
        
        logContainer.scrollTop = logContainer.scrollHeight;
    }
    
    // 切换调试日志显示
    toggleDebugLog() {
        const logContainer = document.getElementById('debug-log');
        if (logContainer) {
            this.debugLogVisible = !this.debugLogVisible;
            logContainer.style.display = this.debugLogVisible ? 'block' : 'none';
            
            // 更新按钮状态
            if (this.debugLogVisible) {
                this.debugToggleButton.classList.add('active');
                this.debugToggleButton.textContent = '隐藏日志';
                this.log('📋 调试日志已显示');
            } else {
                this.debugToggleButton.classList.remove('active');
                this.debugToggleButton.textContent = '调试日志';
                this.log('📋 调试日志已隐藏');
            }
        }
    }
    
    init() {
        this.log('⚙️ 开始初始化组件');
        
        // 绑定键盘按键事件
        this.keys.forEach((key, index) => {
            const keyValue = key.dataset.key;
            this.log(`🔧 绑定按键 [${index}]: ${keyValue || '未定义'}`);
            
            key.addEventListener('click', (e) => {
                e.preventDefault();
                this.log(`👆 按键点击事件: ${keyValue}`);
                this.handleKeyPress(key);
            });
            
            // 添加触摸事件支持
            key.addEventListener('touchstart', (e) => {
                e.preventDefault();
                key.classList.add('pressed');
                this.log(`👆 按键触摸开始: ${keyValue}`);
            });
            
            key.addEventListener('touchend', (e) => {
                e.preventDefault();
                key.classList.remove('pressed');
                this.log(`👆 按键触摸结束: ${keyValue}`);
                // 触摸结束时也触发按键处理
                this.handleKeyPress(key);
            });
        });
        
        this.log(`⌨️ 已绑定 ${this.keys.length} 个按键事件`);
        
        // 绑定发送文本按钮事件
        this.sendTextButton.addEventListener('click', () => {
            this.sendText();
        });
        
        // 绑定刷新统计按钮事件
        this.refreshStatsButton.addEventListener('click', () => {
            this.refreshStats();
        });
        
        // 绑定调试按钮事件
        this.debugToggleButton.addEventListener('click', () => {
            this.toggleDebugLog();
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
            
            // 防止双击缩放
            if (now - lastTouchEnd <= 300) {
                e.preventDefault();
            } else {
                touchCount = 1;
            }
            lastTouchEnd = now;
        }, false);
        
        // 初始化统计信息
        this.log('📊 初始化统计信息');
        this.refreshStats();
        
        // 设置自动刷新统计信息
        this.startStatsAutoRefresh();
        
        this.updateStatus('就绪 (双击屏幕或长按此处显示调试日志)', 'success');
        this.log('✅ 初始化完成');
    }
    
    // 开始自动刷新统计信息
    startStatsAutoRefresh() {
        this.statsInterval = setInterval(() => {
            this.refreshStats(true); // 静默刷新
        }, 3000); // 每3秒刷新一次
        this.log('🔄 启动自动刷新统计 (3秒间隔)');
    }
    
    // 停止自动刷新
    stopStatsAutoRefresh() {
        if (this.statsInterval) {
            clearInterval(this.statsInterval);
            this.statsInterval = null;
            this.log('⏹️ 停止自动刷新统计');
        }
    }
    
    // 刷新统计信息
    async refreshStats(silent = false) {
        try {
            if (!silent) {
                this.refreshStatsButton.textContent = '刷新中...';
                this.refreshStatsButton.disabled = true;
                this.log('📊 手动刷新统计信息');
            }
            
            const stats = await this.getStats();
            this.updateStatsDisplay(stats);
            
            if (!silent) {
                this.log('📊 统计信息刷新成功', 'success');
            }
            
        } catch (error) {
            this.log(`❌ 获取统计信息失败: ${error.message}`, 'error');
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
        
        // 更新延迟分析
        if (stats.latency_breakdown) {
            this.statsElements.queueLatency.textContent = `${stats.latency_breakdown.queue_ms || 0}ms`;
            this.statsElements.processLatency.textContent = `${stats.latency_breakdown.process_ms || 0}ms`;
            this.statsElements.networkLatency.textContent = `${stats.latency_breakdown.network_ms || 0}ms`;
        }
        
        // 更新延迟历史图表
        if (stats.latency_history) {
            this.updateLatencyChart(stats.latency_history);
        }
    }
    
    async handleKeyPress(keyElement) {
        const keyValue = keyElement.dataset.key;
        if (!keyValue) {
            this.log('❌ 按键元素缺少 data-key 属性', 'error');
            return;
        }
        
        this.log(`🎯 [开始] 按键点击事件触发: ${keyValue}`, 'info');
        this.log(`🌍 当前网络状态: ${navigator.onLine ? '在线' : '离线'}`);
        this.log(`🔗 API基础URL: ${this.apiBase}`);
        
        this.requestCount++;
        const requestId = this.requestCount;
        
        this.log(`🔵 [${requestId}] 开始按键请求: ${keyValue.toUpperCase()}`);
        
        // 添加按键动画
        keyElement.classList.add('animate');
        setTimeout(() => {
            keyElement.classList.remove('animate');
        }, 150);
        
        const startTime = performance.now();
        
        // 异步处理，不阻塞UI
        this.pressKeyAsync(keyValue, requestId, startTime);
        
        // 立即更新状态
        this.updateStatus(`按下 ${keyValue.toUpperCase()}`, 'loading');
    }
    
    async pressKeyAsync(keyValue, requestId, startTime) {
        try {
            this.log(`📤 [${requestId}] 发送请求到: /press?key=${keyValue}&duration=50`);
            
            await this.pressKey(keyValue, 50); // 减少持续时间到50ms
            
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            this.log(`✅ [${requestId}] 按键成功: ${keyValue.toUpperCase()} (${latency}ms)`, 'success');
            this.updateStatus(`${keyValue.toUpperCase()} 已发送 (${latency}ms)`, 'success');
            
        } catch (error) {
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            this.log(`❌ [${requestId}] 按键失败: ${error.message} (${latency}ms)`, 'error');
            this.updateStatus(`按键失败: ${error.message} (${latency}ms)`, 'error');
        }
        
        // 1秒后恢复就绪状态
        setTimeout(() => {
            this.updateStatus('就绪', 'success');
        }, 1000);
    }
    
    async sendText() {
        const text = this.textInput.value.trim();
        if (!text) {
            this.log('⚠️ 文本输入为空', 'warning');
            this.updateStatus('请输入要发送的文本', 'error');
            return;
        }
        
        this.requestCount++;
        const requestId = this.requestCount;
        
        this.log(`🔵 [${requestId}] 开始文本输入请求: "${text}"`);
        
        const startTime = performance.now();
        
        // 异步处理文本输入
        this.typeTextAsync(text, requestId, startTime);
        
        // 立即更新状态和清空输入框
        this.updateStatus('正在发送文本...', 'loading');
        this.textInput.value = '';
    }
    
    async typeTextAsync(text, requestId, startTime) {
        try {
            this.log(`📤 [${requestId}] 发送POST请求到: /type`);
            
            await this.typeText(text);
            
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            this.log(`✅ [${requestId}] 文本输入成功 (${latency}ms): "${text}"`, 'success');
            this.updateStatus(`文本已发送 (${latency}ms): "${text}"`, 'success');
            
        } catch (error) {
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);
            
            this.log(`❌ [${requestId}] 文本输入失败 (${latency}ms): ${error.message}`, 'error');
            this.updateStatus(`发送失败 (${latency}ms): ${error.message}`, 'error');
        }
        
        // 2秒后恢复就绪状态
        setTimeout(() => {
            this.updateStatus('就绪', 'success');
        }, 2000);
    }
    
    async pressKey(key, duration = 50) {
        const url = `${this.apiBase}/press?key=${encodeURIComponent(key)}&duration=${duration}`;
        this.log(`🌐 准备发送GET请求: ${url}`);
        
        try {
            this.log(`📡 开始发送请求...`);
            const fetchStart = performance.now();
            
            const response = await fetch(url, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            
            const fetchEnd = performance.now();
            const fetchTime = Math.round(fetchEnd - fetchStart);
            
            this.log(`📥 收到响应: ${response.status} ${response.statusText} (${fetchTime}ms)`);
            this.log(`📥 响应头: ${JSON.stringify(Object.fromEntries(response.headers))}`);
            
            if (!response.ok) {
                const errorText = await response.text();
                this.log(`📥 错误响应内容: ${errorText}`, 'error');
                throw new Error(errorText || `HTTP ${response.status}`);
            }
            
            const result = await response.text();
            this.log(`📥 响应内容: "${result}"`);
            return result;
            
        } catch (error) {
            this.log(`💥 网络请求异常: ${error.message}`, 'error');
            this.log(`💥 错误类型: ${error.constructor.name}`, 'error');
            this.log(`💥 错误堆栈: ${error.stack}`, 'error');
            throw error;
        }
    }
    
    async typeText(text) {
        const url = `${this.apiBase}/type`;
        const body = JSON.stringify({ text });
        
        this.log(`🌐 POST ${url}`);
        this.log(`📤 请求体: ${body}`);
        
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: body
        });
        
        this.log(`📥 响应状态: ${response.status} ${response.statusText}`);
        
        if (!response.ok) {
            const errorText = await response.text();
            this.log(`📥 错误响应内容: ${errorText}`, 'error');
            throw new Error(errorText || `HTTP ${response.status}`);
        }
        
        const result = await response.text();
        this.log(`📥 响应内容: ${result}`);
        return result;
    }
    
    async getStats() {
        const url = `${this.apiBase}/stats`;
        
        const response = await fetch(url, {
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
        const url = `${this.apiBase}/actions`;
        const body = JSON.stringify(actions);
        
        this.log(`🌐 POST ${url}`);
        this.log(`📤 请求体: ${body}`);
        
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: body
        });
        
        this.log(`📥 响应状态: ${response.status} ${response.statusText}`);
        
        if (!response.ok) {
            const errorText = await response.text();
            this.log(`📥 错误响应内容: ${errorText}`, 'error');
            throw new Error(errorText || `HTTP ${response.status}`);
        }
        
        const result = await response.text();
        this.log(`📥 响应内容: ${result}`);
        return result;
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
    
    // 更新延迟图表
    updateLatencyChart(latencyHistory) {
        const canvas = document.getElementById('latencyChart');
        if (!canvas) return;
        
        const ctx = canvas.getContext('2d');
        const width = canvas.width;
        const height = canvas.height;
        
        // 清空画布
        ctx.clearRect(0, 0, width, height);
        
        if (!latencyHistory || latencyHistory.length === 0) {
            ctx.fillStyle = '#666';
            ctx.font = '14px Arial';
            ctx.textAlign = 'center';
            ctx.fillText('暂无数据', width / 2, height / 2);
            return;
        }
        
        // 设置边距
        const margin = { top: 20, right: 20, bottom: 30, left: 50 };
        const chartWidth = width - margin.left - margin.right;
        const chartHeight = height - margin.top - margin.bottom;
        
        // 获取最大延迟值用于缩放
        const maxLatency = Math.max(...latencyHistory.map(record => 
            (record.total_latency || 0) / 1000000 // 转换为毫秒
        ));
        
        if (maxLatency === 0) return;
        
        // 绘制背景网格
        ctx.strokeStyle = '#e0e0e0';
        ctx.lineWidth = 1;
        
        // 水平网格线
        for (let i = 0; i <= 5; i++) {
            const y = margin.top + (chartHeight / 5) * i;
            ctx.beginPath();
            ctx.moveTo(margin.left, y);
            ctx.lineTo(margin.left + chartWidth, y);
            ctx.stroke();
        }
        
        // 绘制面积图
        if (latencyHistory.length > 1) {
            const xStep = chartWidth / (latencyHistory.length - 1);
            
            // 绘制叠加面积图
            const colors = [
                'rgba(255, 99, 132, 0.6)',  // 网络延迟 - 红色
                'rgba(54, 162, 235, 0.6)',  // 处理延迟 - 蓝色
                'rgba(255, 206, 86, 0.6)'   // 队列延迟 - 黄色
            ];
            
            const layers = ['network_latency', 'process_latency', 'queue_latency'];
            
            layers.forEach((layer, layerIndex) => {
                ctx.fillStyle = colors[layerIndex];
                ctx.beginPath();
                ctx.moveTo(margin.left, margin.top + chartHeight);
                
                // 计算累积值
                latencyHistory.forEach((record, i) => {
                    const x = margin.left + xStep * i;
                    
                    let cumulativeValue = 0;
                    for (let j = layerIndex; j < layers.length; j++) {
                        const layerValue = (record[layers[j]] || 0) / 1000000; // 转换为毫秒
                        cumulativeValue += layerValue;
                    }
                    
                    const y = margin.top + chartHeight - (cumulativeValue / maxLatency) * chartHeight;
                    ctx.lineTo(x, Math.max(margin.top, y));
                });
                
                ctx.lineTo(margin.left + chartWidth, margin.top + chartHeight);
                ctx.closePath();
                ctx.fill();
            });
        }
        
        // 绘制坐标轴
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 2;
        
        // Y轴
        ctx.beginPath();
        ctx.moveTo(margin.left, margin.top);
        ctx.lineTo(margin.left, margin.top + chartHeight);
        ctx.stroke();
        
        // X轴
        ctx.beginPath();
        ctx.moveTo(margin.left, margin.top + chartHeight);
        ctx.lineTo(margin.left + chartWidth, margin.top + chartHeight);
        ctx.stroke();
        
        // 绘制Y轴标签
        ctx.fillStyle = '#666';
        ctx.font = '12px Arial';
        ctx.textAlign = 'right';
        for (let i = 0; i <= 5; i++) {
            const value = (maxLatency / 5) * (5 - i);
            const y = margin.top + (chartHeight / 5) * i;
            ctx.fillText(`${value.toFixed(1)}ms`, margin.left - 5, y + 4);
        }
        

    }
    
    // 页面卸载时清理资源
    cleanup() {
        this.stopStatsAutoRefresh();
        this.log('🧹 清理资源完成');
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