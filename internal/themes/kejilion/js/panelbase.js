// PanelBase API 管理類
class PanelBase {
	constructor() {
		this.baseUrl = 'http://192.168.1.170:8080/api';
		this.activeRequests = new Map();
		this.ws = null;
		this.init();
	}

	init() {
		if (!this.baseUrl) {
			console.error('API url is not set');
			return;
		}
		this.initializeRequests();
		this.initWebSocket();
	}

	// 初始化所有請求
	initializeRequests() {
		document.querySelectorAll('[data-command]').forEach(element => {
			const interval = parseInt(element.dataset.interval);
			const maxCalls = parseInt(element.dataset.maxCalls);
			
			if (interval || (maxCalls === 1)) {
				this.setupActiveRequest(element, interval, maxCalls);
			} else {
				this.setupPassiveRequest(element);
			}
		});
	}

	// 初始化 WebSocket
	initWebSocket() {
		this.ws = this.connectWebSocket();
	}

	// 設置被動請求
	setupPassiveRequest(element) {
		element.addEventListener('click', async () => {
			const confirmMsg = element.dataset.confirm;
			if (confirmMsg && !confirm(confirmMsg)) {
				return;
			}

			const name = element.dataset.command;
			const args = JSON.parse(element.dataset.args || '[]');

			if (!name) {
				console.error('Command name is required');
				return;
			}

			try {
				await this.executeCommand(name, args);
			} catch (error) {
				console.error('Command execution failed:', error);
			}
		});
	}

	// 設置主動請求
	setupActiveRequest(element, interval, maxCalls) {
		const name = element.dataset.command;
		const args = JSON.parse(element.dataset.args || '[]');

		if (!name) {
			console.error('Command name is required');
			return;
		}

		const execute = async () => {
			await this.executeCommand(name, args);
		};

		execute();

		if (maxCalls === 1) return;

		if (interval) {
			const intervalId = setInterval(execute, interval);
			this.activeRequests.set(element, { intervalId });
		}
	}

	// 執行命令
	async executeCommand(name, args = []) {
		if (!this.baseUrl) {
			throw new Error('API url is not set');
		}

		const response = await fetch(`${this.baseUrl}/execute`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				commands: [{
					name: name,
					args: args
				}]
			})
		});

		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}

		return await response.json();
	}

	// WebSocket 連接
	connectWebSocket() {
		const ws = new WebSocket(`ws://${this.baseUrl.replace('http://', '')}/ws-execute`);
		
		ws.onopen = () => {
			console.log('WebSocket connected');
		};
		
		ws.onmessage = (event) => {
			try {
				const response = JSON.parse(event.data);
				console.log('WebSocket message:', response);
				
				// 處理 WebSocket 消息
				this.handleWebSocketMessage(response);
				
			} catch (error) {
				console.error('Failed to parse WebSocket message:', error);
			}
		};
		
		ws.onerror = (error) => {
			console.error('WebSocket error:', error);
		};
		
		ws.onclose = () => {
			console.log('WebSocket closed');
			setTimeout(() => this.initWebSocket(), 5000);
		};

		return ws;
	}

	// 處理 WebSocket 消息
	handleWebSocketMessage(response) {
		// 查找所有帶有 data-command 的元素
		document.querySelectorAll('[data-command]').forEach(element => {
			const command = element.dataset.command;
			const args = JSON.parse(element.dataset.args || '[]');
			const displayType = element.dataset.display;

			// 檢查命令和參數是否匹配
			if (command === 'get_info' && args[0] === response.command) {
				// 根據 data-display 屬性顯示不同類型的信息
				switch (displayType) {
					case 'status':
						element.textContent = response.status;
						element.className = `status-${response.status}`;
						break;
					case 'message':
						element.textContent = response.message || '';
						break;
					case 'data':
						element.textContent = response.data || '';
						break;
					case 'command':
						element.textContent = response.command || '';
						break;
					default:
						// 如果沒有指定 display 類型，顯示 data
						element.textContent = response.data || '';
				}
			}
		});
	}

	// 清理資源
	destroy() {
		this.activeRequests.forEach(request => {
			clearInterval(request.intervalId);
		});
		this.activeRequests.clear();
		if (this.ws) {
			this.ws.close();
		}
	}
}

// 當 DOM 加載完成後初始化
document.addEventListener('DOMContentLoaded', () => {
	window.panelbase = new PanelBase();
}); 