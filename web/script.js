class PiKeyboard {
    constructor() {
        this.statusElement = document.getElementById('status');
        this.textInput = document.getElementById('textInput');
        this.sendTextButton = document.getElementById('sendText');
        this.refreshStatsButton = document.getElementById('refreshStats');
        this.debugToggleButton = document.getElementById('toggleDebug');
        this.keys = document.querySelectorAll('.key');
        
        // 触控板相关元素
        this.touchpad = document.getElementById('touchpad');
        this.leftClickBtn = document.getElementById('leftClick');
        this.rightClickBtn = document.getElementById('rightClick');
        this.doubleClickBtn = document.getElementById('doubleClick');
        
        // 统计信息元素
        this.statsElements = {
            totalRequests: document.getElementById('totalRequests'),
            successRate: document.getElementById('successRate'),
            avgLatency: document.getElementById('avgLatency'),
            processingStatus: document.getElementById('processingStatus'),
            // 延迟分析元素 (移除队列延迟)
            processLatency: document.getElementById('processLatency'),
            networkLatency: document.getElementById('networkLatency')
        };
        
        // 延迟图表
        this.latencyChart = null;
        this.latencyHistory = [];
        
        this.apiBase = window.location.origin;
        this.statsInterval = null;
        this.requestCount = 0;
        this.debugMode = false;
        this.keyPressStartTimes = {};
        
        // WebSocket 相关
        this.useWebSocket = true; // 默认启用 WebSocket
        this.ws = null;
        this.wsReconnectAttempts = 0;
        this.wsMaxReconnectAttempts = 5;
        this.wsReconnectDelay = 1000; // 1秒
        this.wsRequestCallbacks = new Map(); // 存储请求回调
        this.wsRequestId = 0;
        
        // 触控板状态管理
        this.touchpadState = {
            isTracking: false,
            lastX: 0,
            lastY: 0,
            touchStartTime: 0,
            lastTouchTime: 0,
            longPressTimer: null,
            moveThreshold: 2,
            touchCount: 0,
            dpi: 2.0 // 默认 DPI
        };
        
        // 添加日志系统
        this.enableDebugLog();
        
        this.init();
    }
    
    // 启用调试日志
    enableDebugLog() {
        this.debugMode = true;
        this.log('🐛 调试模式已启用');
        
        // 显示调试面板
        const debugPanel = document.querySelector('.debug-panel');
        if (debugPanel) {
            debugPanel.style.display = 'block';
        }
        
        // 更新按钮文本
        const debugButton = document.getElementById('toggleDebug');
        if (debugButton) {
            debugButton.textContent = '关闭调试';
        }
        
        this.log('🚀 Pi Keyboard 初始化 - 并发处理模式');
        this.log(`📍 API Base URL: ${this.apiBase}`);
        this.log(`🌐 User Agent: ${navigator.userAgent}`);
        this.log(`📱 Screen Size: ${window.screen.width}x${window.screen.height}`);
        this.log(`🔗 当前页面URL: ${window.location.href}`);
        this.log(`🌍 网络状态: ${navigator.onLine ? '在线' : '离线'}`);
        this.log(`🕐 页面加载时间: ${new Date().toLocaleString()}`);
        this.log('⚡ 并发处理: 无队列等待，每个请求独立处理');
        
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
            this.debugMode = !this.debugMode;
            logContainer.style.display = this.debugMode ? 'block' : 'none';
            
            // 更新按钮状态
            if (this.debugMode) {
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
        
        // 初始化 WebSocket
        this.initWebSocket();

        // 绑定按键事件
        this.keys.forEach(key => {
            const keyValue = key.dataset.key;
            
            const handlePressStart = (e) => {
                e.preventDefault();
                this.handleKeyPress(key);
            };
            
            const handlePressEnd = (e) => {
                // 处理按键释放逻辑（如果需要）
            };
            
            // 鼠标事件（按键）
            key.addEventListener('mousedown', handlePressStart);
            key.addEventListener('mouseup', handlePressEnd);
            key.addEventListener('mouseleave', (e) => {
                 // 如果按着鼠标移出按钮，也算释放
                if (this.keyPressStartTimes[keyValue]) {
                    handlePressEnd(e);
                }
            });

            // 触摸事件
            key.addEventListener('touchstart', handlePressStart, { passive: false });
            key.addEventListener('touchend', handlePressEnd);
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
        
        // 绑定连接模式切换按钮事件
        const toggleConnectionBtn = document.getElementById('toggleConnectionMode');
        if (toggleConnectionBtn) {
            toggleConnectionBtn.addEventListener('click', () => {
                this.toggleConnectionMode();
            });
        }
        
        // 绑定触控板事件
        this.initTouchpad();
        
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
        
        this.updateStatus('正在初始化...', 'info');
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
        if (this.statsElements.totalRequests) {
            this.statsElements.totalRequests.textContent = stats.total_requests || 0;
        }
        
        if (this.statsElements.successRate) {
            const rate = stats.success_rate ? `${stats.success_rate.toFixed(1)}%` : '0%';
            this.statsElements.successRate.textContent = rate;
        }
        
        if (this.statsElements.avgLatency) {
            this.statsElements.avgLatency.textContent = `${stats.average_latency_ms || 0}ms`;
        }
        
        if (this.statsElements.processingStatus) {
            this.statsElements.processingStatus.textContent = stats.currently_processing || 0;
        }
        
        if (this.statsElements.processLatency) {
            this.statsElements.processLatency.textContent = `${stats.latency_breakdown?.process_ms || 0}ms`;
        }
        
        if (this.statsElements.networkLatency) {
            this.statsElements.networkLatency.textContent = `${stats.latency_breakdown?.network_ms || 0}ms`;
        }

        // 更新WebSocket统计
        this.updateWebSocketStats(stats.websocket);
        
        // 更新延迟图表
        if (stats.latency_history) {
            this.updateLatencyChart(stats.latency_history);
        }
    }
    
    // 更新WebSocket统计显示
    updateWebSocketStats(wsStats) {
        const wsActiveConnections = document.getElementById('wsActiveConnections');
        const wsMessagesReceived = document.getElementById('wsMessagesReceived');
        const wsMessagesSent = document.getElementById('wsMessagesSent');

        if (wsStats && wsStats.enabled !== false) {
            // 显示WebSocket统计
            if (wsActiveConnections) {
                wsActiveConnections.style.display = 'flex';
                wsActiveConnections.querySelector('.stat-value').textContent = wsStats.active_connections || 0;
            }
            
            if (wsMessagesReceived) {
                wsMessagesReceived.style.display = 'flex';
                wsMessagesReceived.querySelector('.stat-value').textContent = wsStats.messages_received || 0;
            }
            
            if (wsMessagesSent) {
                wsMessagesSent.style.display = 'flex';
                wsMessagesSent.querySelector('.stat-value').textContent = wsStats.messages_sent || 0;
            }
        } else {
            // 隐藏WebSocket统计
            if (wsActiveConnections) wsActiveConnections.style.display = 'none';
            if (wsMessagesReceived) wsMessagesReceived.style.display = 'none';
            if (wsMessagesSent) wsMessagesSent.style.display = 'none';
        }
    }
    
    // 修改按键处理以支持 WebSocket
    async handleKeyPress(keyElement, duration = 50) {
        const keyValue = keyElement.dataset.key;
        if (!keyValue) {
            this.log('❌ 按键元素缺少 data-key 属性', 'error');
            return;
        }
        
        this.log(`🎯 [开始] 按键点击事件触发: ${keyValue}, 时长: ${duration}ms`, 'info');
        
        this.requestCount++;
        const requestId = this.requestCount;
        
        this.log(`🔵 [${requestId}] 开始按键请求: ${keyValue.toUpperCase()}`);
        
        // 添加按键动画
        keyElement.classList.add('animate');
        setTimeout(() => {
            keyElement.classList.remove('animate');
        }, 150);
        
        const startTime = performance.now();
        
        try {
            if (this.useWebSocket && this.ws && this.ws.readyState === WebSocket.OPEN) {
                // 使用 WebSocket
                await this.sendWebSocketMessage('key_press', {
                    key: keyValue,
                    duration: duration
                }, true);
            } else {
                // 降级到 HTTP API
                await this.pressKey(keyValue, duration);
            }
            
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
    }
    
    // 修改文本发送以支持 WebSocket
    async sendText() {
        const text = this.textInput.value.trim();
        if (!text) {
            this.updateStatus('请输入文本', 'warning');
            return;
        }

        this.requestCount++;
        const requestId = this.requestCount;
        const startTime = performance.now();

        this.log(`🔵 [${requestId}] 开始文本输入请求: "${text}"`);
        this.updateStatus('正在发送文本...', 'loading');

        try {
            if (this.useWebSocket && this.ws && this.ws.readyState === WebSocket.OPEN) {
                // 使用 WebSocket
                await this.sendWebSocketMessage('type_text', {
                    text: text
                }, true);
            } else {
                // 降级到 HTTP API
                await this.typeText(text);
            }

            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);

            this.log(`✅ [${requestId}] 文本输入成功: "${text}" (${latency}ms)`, 'success');
            this.updateStatus(`文本已发送 (${latency}ms)`, 'success');
            
            // 清空输入框
            this.textInput.value = '';
        } catch (error) {
            const endTime = performance.now();
            const latency = Math.round(endTime - startTime);

            this.log(`❌ [${requestId}] 文本输入失败: ${error.message} (${latency}ms)`, 'error');
            this.updateStatus(`文本发送失败: ${error.message} (${latency}ms)`, 'error');
        }
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
    
    // 更新延迟图表 (简化版，移除队列延迟)
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
        
        // 绘制面积图 (只显示处理延迟和网络延迟)
        if (latencyHistory.length > 1) {
            const xStep = chartWidth / (latencyHistory.length - 1);
            
            // 绘制叠加面积图 (移除队列延迟)
            const colors = [
                'rgba(255, 99, 132, 0.6)',  // 网络延迟 - 红色
                'rgba(54, 162, 235, 0.6)',  // 处理延迟 - 蓝色
            ];
            
            const layers = ['network_latency', 'process_latency'];
            
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
    
    // 初始化触控板
    initTouchpad() {
        if (!this.touchpad) {
            this.log('❌ 触控板元素未找到');
            return;
        }

        this.log('🖱️ 初始化触控板功能');
        
        // 绑定触控板事件
        this.bindTouchpadEvents();
        
        // 绑定触控板按钮事件
        this.bindTouchpadButtons();
        
        // 初始化 DPI 控件
        this.initDPIControls();
    }

    // 绑定触控板事件
    bindTouchpadEvents() {
        // 鼠标事件（桌面端触控板）
        this.touchpad.addEventListener('mousedown', (e) => {
            this.handleTouchpadStart(e.clientX, e.clientY, e);
        });

        this.touchpad.addEventListener('mousemove', (e) => {
            if (this.touchpadState.isTracking) {
                this.handleTouchpadMove(e.clientX, e.clientY, e);
            }
        });

        this.touchpad.addEventListener('mouseup', (e) => {
            this.handleTouchpadEnd(e);
        });

        this.touchpad.addEventListener('mouseleave', (e) => {
            if (this.touchpadState.isTracking) {
                this.handleTouchpadEnd(e);
            }
        });

        // 触摸事件（移动端）
        this.touchpad.addEventListener('touchstart', (e) => {
            e.preventDefault();
            const touch = e.touches[0];
            this.touchpadState.touchCount = e.touches.length;
            this.handleTouchpadStart(touch.clientX, touch.clientY, e);
        }, { passive: false });

        this.touchpad.addEventListener('touchmove', (e) => {
            e.preventDefault();
            const touch = e.touches[0];
            this.touchpadState.touchCount = e.touches.length;
            
            if (this.touchpadState.isTracking) {
                this.handleTouchpadMove(touch.clientX, touch.clientY, e);
            }
        }, { passive: false });

        this.touchpad.addEventListener('touchend', (e) => {
            e.preventDefault();
            this.handleTouchpadEnd(e);
        });

        // 滚轮事件
        this.touchpad.addEventListener('wheel', (e) => {
            e.preventDefault();
            this.handleTouchpadScroll(e.deltaX, e.deltaY);
        }, { passive: false });
    }

    // 绑定触控板按钮事件
    bindTouchpadButtons() {
        if (this.leftClickBtn) {
            this.leftClickBtn.addEventListener('click', () => {
                this.sendTouchpadClick('left', 'single');
            });
        }

        if (this.rightClickBtn) {
            this.rightClickBtn.addEventListener('click', () => {
                this.sendTouchpadClick('right', 'single');
            });
        }

        if (this.doubleClickBtn) {
            this.doubleClickBtn.addEventListener('click', () => {
                this.sendTouchpadClick('left', 'double');
            });
        }
    }

    // 初始化 DPI 控件
    initDPIControls() {
        this.dpiSlider = document.getElementById('dpiSlider');
        this.dpiValue = document.getElementById('dpiValue');
        this.dpiPresetBtns = document.querySelectorAll('.dpi-preset-btn');

        if (!this.dpiSlider || !this.dpiValue) {
            this.log('⚠️ DPI 控件未找到', 'warning');
            return;
        }

        // 设置初始值
        this.updateDPIValue(this.touchpadState.dpi);
        this.updateActivePreset(this.touchpadState.dpi);

        // 绑定滑块事件
        this.dpiSlider.addEventListener('input', (e) => {
            const dpi = parseFloat(e.target.value);
            this.touchpadState.dpi = dpi;
            this.updateDPIValue(dpi);
            this.updateActivePreset(dpi);
            this.log(`🎯 DPI 调整为: ${dpi}x`);
        });

        // 绑定预设按钮事件
        this.dpiPresetBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const dpi = parseFloat(btn.dataset.dpi);
                this.touchpadState.dpi = dpi;
                this.dpiSlider.value = dpi;
                this.updateDPIValue(dpi);
                this.updateActivePreset(dpi);
                this.log(`🎯 DPI 预设: ${dpi}x`);
            });
        });

        this.log('🎯 DPI 控件初始化完成');
    }

    // 更新 DPI 显示值
    updateDPIValue(dpi) {
        if (this.dpiValue) {
            this.dpiValue.textContent = dpi.toFixed(1);
        }
    }

    // 更新激活的预设按钮
    updateActivePreset(dpi) {
        this.dpiPresetBtns.forEach(btn => {
            const presetDpi = parseFloat(btn.dataset.dpi);
            if (Math.abs(presetDpi - dpi) < 0.1) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });
    }

    // 处理触控板开始事件
    handleTouchpadStart(x, y, event) {
        this.touchpadState.isTracking = true;
        this.touchpadState.lastX = x;
        this.touchpadState.lastY = y;
        this.touchpadState.touchStartTime = Date.now();
        this.touchpadState.lastTouchTime = Date.now();

        // 设置长按定时器（右键）
        this.touchpadState.longPressTimer = setTimeout(() => {
            if (this.touchpadState.isTracking) {
                this.sendTouchpadClick('right', 'single');
                this.touchpadState.isTracking = false;
                this.log('🖱️ 长按触发右键');
            }
        }, 500);

        this.log(`🖱️ 触控板开始跟踪: (${x}, ${y})`);
    }

    // 处理触控板移动事件
    handleTouchpadMove(x, y, event) {
        if (!this.touchpadState.isTracking) return;

        const deltaX = x - this.touchpadState.lastX;
        const deltaY = y - this.touchpadState.lastY;
        
        // 计算移动距离
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        // 如果移动距离太小，忽略
        if (distance < this.touchpadState.moveThreshold) return;

        // 清除长按定时器（因为发生了移动）
        if (this.touchpadState.longPressTimer) {
            clearTimeout(this.touchpadState.longPressTimer);
            this.touchpadState.longPressTimer = null;
        }

        // 节流：限制发送频率
        const now = Date.now();
        if (now - this.touchpadState.lastTouchTime < 16) { // 约60fps
            return;
        }
        this.touchpadState.lastTouchTime = now;

        // 发送触控板移动
        this.sendTouchpadMove(Math.round(deltaX), Math.round(deltaY));
        
        this.touchpadState.lastX = x;
        this.touchpadState.lastY = y;
    }

    // 处理触控板结束事件
    handleTouchpadEnd(event) {
        if (!this.touchpadState.isTracking) return;

        const touchDuration = Date.now() - this.touchpadState.touchStartTime;
        
        // 清除长按定时器
        if (this.touchpadState.longPressTimer) {
            clearTimeout(this.touchpadState.longPressTimer);
            this.touchpadState.longPressTimer = null;
            
            // 如果是短时间的点击，发送左键点击
            if (touchDuration < 200) {
                this.sendTouchpadClick('left', 'single');
                this.log('🖱️ 短按触发左键');
            }
        }

        this.touchpadState.isTracking = false;
        this.log(`🖱️ 触控板结束跟踪，持续时间: ${touchDuration}ms`);
    }

    // 处理触控板滚动事件
    handleTouchpadScroll(deltaX, deltaY) {
        // 如果是双指滚动，发送滚动事件
        if (this.touchpadState.touchCount >= 2) {
            this.sendTouchpadScroll(Math.round(deltaX), Math.round(deltaY));
            this.log(`🖱️ 双指滚动: (${deltaX}, ${deltaY})`);
        }
    }

    // 发送触控板移动请求
    async sendTouchpadMove(deltaX, deltaY) {
        try {
            if (this.useWebSocket && this.ws && this.ws.readyState === WebSocket.OPEN) {
                // 使用 WebSocket (不等待响应以提高性能)
                await this.sendWebSocketMessage('touchpad_move', {
                    deltaX,
                    deltaY,
                    dpi: this.touchpadState.dpi
                }, false);
            } else {
                // 降级到 HTTP API
                const url = `${this.apiBase}/touchpad/move`;
                const body = JSON.stringify({ 
                    deltaX, 
                    deltaY, 
                    dpi: this.touchpadState.dpi 
                });
                
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: body
                });

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }
            }

            // 不记录每个移动的详细日志，避免日志过多
        } catch (error) {
            this.log(`❌ 触控板移动失败: ${error.message}`, 'error');
        }
    }

    // 发送触控板点击请求
    async sendTouchpadClick(button, type) {
        try {
            if (this.useWebSocket && this.ws && this.ws.readyState === WebSocket.OPEN) {
                // 使用 WebSocket
                this.log(`🌐 [WebSocket] 发送触控板点击: ${button} ${type}`);
                await this.sendWebSocketMessage('touchpad_click', {
                    button,
                    type
                }, true);
                this.log(`✅ 触控板点击成功: ${button} ${type}`, 'success');
            } else {
                // 降级到 HTTP API
                this.log(`🌐 [HTTP] 发送触控板点击: ${button} ${type}`);
                const url = `${this.apiBase}/touchpad/click`;
                const body = JSON.stringify({ button, type });
                
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: body
                });

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }

                this.log(`✅ 触控板点击成功: ${button} ${type}`, 'success');
            }
        } catch (error) {
            this.log(`❌ 触控板点击失败: ${error.message}`, 'error');
        }
    }

    // 发送触控板滚动请求
    async sendTouchpadScroll(deltaX, deltaY) {
        try {
            if (this.useWebSocket && this.ws && this.ws.readyState === WebSocket.OPEN) {
                // 使用 WebSocket
                await this.sendWebSocketMessage('touchpad_scroll', {
                    deltaX,
                    deltaY
                }, true);
            } else {
                // 降级到 HTTP API
                const url = `${this.apiBase}/touchpad/scroll`;
                const body = JSON.stringify({ deltaX, deltaY });
                
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: body
                });

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }
            }

            this.log(`🖱️ 触控板滚动: (${deltaX}, ${deltaY})`);
        } catch (error) {
            this.log(`❌ 触控板滚动失败: ${error.message}`, 'error');
        }
    }

    // 初始化 WebSocket 连接
    initWebSocket() {
        if (!this.useWebSocket) {
            this.log('WebSocket 功能已禁用，使用 HTTP API');
            return;
        }

        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/ws`;
        
        this.log(`🔌 正在连接 WebSocket: ${wsUrl}`);
        this.updateConnectionStatus('connecting', '正在连接...', 'WebSocket 模式');
        
        try {
            this.ws = new WebSocket(wsUrl);
            this.setupWebSocketEventHandlers();
        } catch (error) {
            this.log(`❌ WebSocket 连接失败: ${error.message}`, 'error');
            this.fallbackToHTTP();
        }
    }

    // 设置 WebSocket 事件处理器
    setupWebSocketEventHandlers() {
        this.ws.onopen = () => {
            this.log('✅ WebSocket 连接成功', 'success');
            this.wsReconnectAttempts = 0;
            this.updateConnectionStatus('connected', 'WebSocket 已连接', 'WebSocket 模式');
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleWebSocketMessage(message);
            } catch (error) {
                this.log(`❌ WebSocket 消息解析失败: ${error.message}`, 'error');
            }
        };

        this.ws.onclose = (event) => {
            this.log(`🔌 WebSocket 连接关闭: ${event.code} ${event.reason}`);
            this.updateConnectionStatus('disconnected', 'WebSocket 已断开', 'WebSocket 模式');
            this.handleWebSocketClose(event);
        };

        this.ws.onerror = (error) => {
            this.log(`❌ WebSocket 错误: ${error}`, 'error');
        };
    }

    // 处理 WebSocket 消息
    handleWebSocketMessage(message) {
        const { type, data, request_id } = message;
        
        // 处理请求响应
        if (request_id && this.wsRequestCallbacks.has(request_id)) {
            const callback = this.wsRequestCallbacks.get(request_id);
            this.wsRequestCallbacks.delete(request_id);
            
            if (type === 'error') {
                callback.reject(new Error(data.message || 'Unknown error'));
            } else {
                callback.resolve(data);
            }
            return;
        }

        // 处理其他消息类型
        switch (type) {
            case 'pong':
                // 心跳响应
                break;
            case 'broadcast':
                this.log(`📢 收到广播: ${JSON.stringify(data)}`);
                break;
            default:
                this.log(`📥 收到未知消息类型: ${type}`);
        }
    }

    // 处理 WebSocket 连接关闭
    handleWebSocketClose(event) {
        if (event.code === 1000) {
            // 正常关闭
            this.updateConnectionStatus('disconnected', 'WebSocket 连接已关闭', 'WebSocket 模式');
            return;
        }

        // 异常关闭，尝试重连
        if (this.wsReconnectAttempts < this.wsMaxReconnectAttempts) {
            this.wsReconnectAttempts++;
            const delay = this.wsReconnectDelay * Math.pow(2, this.wsReconnectAttempts - 1);
            
            this.log(`🔄 WebSocket 重连中... (${this.wsReconnectAttempts}/${this.wsMaxReconnectAttempts})`, 'warning');
            this.updateConnectionStatus('connecting', `重连中... (${this.wsReconnectAttempts}/${this.wsMaxReconnectAttempts})`, 'WebSocket 模式');
            
            setTimeout(() => {
                this.initWebSocket();
            }, delay);
        } else {
            this.log('❌ WebSocket 重连失败，切换到 HTTP API', 'error');
            this.fallbackToHTTP();
        }
    }

    // 降级到 HTTP API
    fallbackToHTTP() {
        this.useWebSocket = false;
        this.ws = null;
        this.updateConnectionStatus('http-mode', '使用 HTTP API', 'HTTP API 模式');
        this.log('🔄 已切换到 HTTP API 模式');
    }

    // 切换连接模式
    toggleConnectionMode() {
        if (this.useWebSocket) {
            // 切换到 HTTP 模式
            this.log('🔄 手动切换到 HTTP API 模式');
            if (this.ws) {
                this.ws.close(1000, 'Manual switch to HTTP');
            }
            this.fallbackToHTTP();
        } else {
            // 切换到 WebSocket 模式
            this.log('🔄 手动切换到 WebSocket 模式');
            this.useWebSocket = true;
            this.wsReconnectAttempts = 0;
            this.initWebSocket();
        }
    }

    // 更新连接状态显示
    updateConnectionStatus(status, message, mode) {
        const connectionDot = document.getElementById('connectionDot');
        const connectionStatus = document.getElementById('connectionStatus');
        const connectionMode = document.getElementById('connectionMode');

        if (connectionDot) {
            connectionDot.className = `status-dot ${status}`;
        }

        if (connectionStatus) {
            connectionStatus.textContent = message;
        }

        if (connectionMode) {
            connectionMode.textContent = mode;
        }

        // 更新切换按钮文本
        const toggleBtn = document.getElementById('toggleConnectionMode');
        if (toggleBtn) {
            if (this.useWebSocket) {
                toggleBtn.textContent = '切换到 HTTP';
            } else {
                toggleBtn.textContent = '切换到 WebSocket';
            }
        }
    }

    // 发送 WebSocket 消息
    sendWebSocketMessage(type, data, expectResponse = false) {
        return new Promise((resolve, reject) => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                reject(new Error('WebSocket 连接未就绪'));
                return;
            }

            let requestId = null;
            if (expectResponse) {
                requestId = `req_${++this.wsRequestId}_${Date.now()}`;
                this.wsRequestCallbacks.set(requestId, { resolve, reject });
                
                // 设置超时
                setTimeout(() => {
                    if (this.wsRequestCallbacks.has(requestId)) {
                        this.wsRequestCallbacks.delete(requestId);
                        reject(new Error('请求超时'));
                    }
                }, 5000);
            }

            const message = {
                type,
                data,
                timestamp: new Date().toISOString(),
                request_id: requestId
            };

            try {
                this.ws.send(JSON.stringify(message));
                if (!expectResponse) {
                    resolve();
                }
            } catch (error) {
                if (requestId) {
                    this.wsRequestCallbacks.delete(requestId);
                }
                reject(error);
            }
        });
    }

    // 页面卸载时清理资源
    cleanup() {
        this.stopStatsAutoRefresh();
        
        // 清理触控板定时器
        if (this.touchpadState && this.touchpadState.longPressTimer) {
            clearTimeout(this.touchpadState.longPressTimer);
        }
        
        // 关闭 WebSocket 连接
        if (this.ws) {
            this.ws.close(1000, 'Page unloading');
        }
        
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