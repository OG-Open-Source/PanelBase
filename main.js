// PanelBase API 管理類
class PanelBase {
	constructor() {
		this.baseUrl = 'http://IP:PORT/ENTRY';
		this.activeRequests = new Map();
		this.ws = null;
		this.activeModal = null;
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
			const modalId = element.dataset.modal;
			const modalTitle = element.dataset.title;

			if (!name) {
				console.error('Command name is required');
				return;
			}

			// 如果有指定模態框，則顯示它
			if (modalId) {
				const modal = document.getElementById(modalId);
				const titleElement = modal.querySelector('#modalTitle');
				const logElement = modal.querySelector('#logContainer');
				const closeBtn = modal.querySelector('#modalCloseBtn');

				if (titleElement && modalTitle) {
					titleElement.textContent = modalTitle;
				}
				if (logElement) {
					logElement.textContent = '';
				}
				if (closeBtn) {
					closeBtn.style.display = 'none';
				}

				modal.style.display = 'block';

				// 保存模態框元素以供 WebSocket 消息處理使用
				this.activeModal = {
					id: modalId,
					command: args[0],
					elements: {
						log: logElement,
						close: closeBtn
					}
				};
			}

			try {
				await this.executeCommand(name, args);
			} catch (error) {
				console.error('Command execution failed:', error);
				if (this.activeModal?.elements.close) {
					this.activeModal.elements.close.style.display = 'block';
				}
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
		console.log('WebSocket message:', response);

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

		// 處理模態框顯示
		if (this.activeModal && response.command === this.activeModal.command) {
			const { log, close } = this.activeModal.elements;

			if (log) {
				// 追加新的輸出到日誌容器
				const newOutput = response.data || response.message || '';
				if (newOutput) {
					// 如果日誌容器為空，直接設置內容
					if (!log.textContent) {
						log.textContent = newOutput;
					} else {
						// 否則追加新行
						log.textContent += '\n' + newOutput;
					}
					// 自動滾動到底部
					log.scrollTop = log.scrollHeight;
				}
			}

			// 命令完成時
			if (response.status === 'success' || response.status === 'error') {
				if (close) {
					close.style.display = 'block';
				}
				this.activeModal = null;
			}
		}
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

	// 添加模態框關閉按鈕事件處理
	const closeButtons = document.querySelectorAll('.modal .modal-btn');
	closeButtons.forEach(button => {
		button.addEventListener('click', () => {
			const modal = button.closest('.modal');
			if (modal) {
				modal.style.display = 'none';
			}
		});
	});
}); 