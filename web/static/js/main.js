// PanelBase Main JavaScript

// Get the entry point from the URL path
const getEntryPoint = () => {
	const path = window.location.pathname;
	const parts = path.split('/').filter(p => p.length > 0);
	return parts.length > 0 ? parts[0] : 'panel'; // Default to 'panel' if not found
};

// API base URL
const API_BASE = `/${getEntryPoint()}/api`;

// Create Axios instance with common settings
const api = axios.create({
	baseURL: API_BASE,
	headers: {
		'Content-Type': 'application/json'
	}
});

// Add auth token to requests if available
api.interceptors.request.use(config => {
	const token = localStorage.getItem('token');
	if (token) {
		config.headers.Authorization = `Bearer ${token}`;
	}
	return config;
});

// Main Vue application
new Vue({
	el: '#app',
	data: {
		// Authentication
		isAuthenticated: false,
		loggingIn: false,
		loginError: null,
		credentials: {
			username: '',
			password: ''
		},
		user: null,

		// UI state
		activeTab: 'dashboard',

		// Tasks
		tasks: [],
		recentTasks: [],
		selectedTask: null,
		showCreateTaskModal: false,
		newTask: {
			name: '',
			work_dir: '',
			commands: [
				{ command: '', args: '' }
			]
		},

		// Users
		users: [],
		showUserModal: false,
		userForm: {
			id: null,
			username: '',
			password: '',
			role: 'USER',
			active: true
		},
		apiKeyModal: {
			show: false,
			username: '',
			key: ''
		},
		deleteUserModal: {
			show: false,
			id: null,
			username: ''
		},

		// Stats
		stats: {
			totalTasks: 0,
			runningTasks: 0,
			completedTasks: 0,
			failedTasks: 0
		},

		// Status polling
		pollingInterval: null,
		pollingDelay: 1000 // 1秒更新一次，获得更流畅的体验
	},

	created() {
		// Check if user is logged in
		const token = localStorage.getItem('token');
		const userData = localStorage.getItem('user');
		if (token && userData) {
			try {
				this.user = JSON.parse(userData);
				this.isAuthenticated = true;
				this.loadInitialData();
				this.startPolling(); // 开始轮询任务状态
			} catch (error) {
				this.logout();
			}
		}
	},

	beforeDestroy() {
		// 清理轮询定时器
		this.stopPolling();
	},

	methods: {
		// 开始轮询任务状态
		startPolling() {
			if (this.pollingInterval) {
				clearInterval(this.pollingInterval);
			}

			if (this.isAuthenticated) {
				this.pollingInterval = setInterval(() => {
					this.pollTaskStatus();
				}, this.pollingDelay);
			}
		},

		// 停止轮询
		stopPolling() {
			if (this.pollingInterval) {
				clearInterval(this.pollingInterval);
				this.pollingInterval = null;
			}
		},

		// 轮询任务状态
		async pollTaskStatus() {
			try {
				const response = await api.get('/tasks');
				const updatedTasks = response.data;

				// 始终更新任务列表和状态数据
				this.tasks = updatedTasks;

				// 更新最近任务列表
				this.recentTasks = [...updatedTasks]
					.sort((a, b) => b.start_time - a.start_time)
					.slice(0, 5);

				// 更新统计数据
				this.stats.totalTasks = updatedTasks.length;
				this.stats.runningTasks = updatedTasks.filter(t => t.status === 'RUNNING').length;
				this.stats.completedTasks = updatedTasks.filter(t => t.status === 'COMPLETED').length;
				this.stats.failedTasks = updatedTasks.filter(t => t.status === 'FAILED').length;

				// 如果当前有选中的任务，更新它的详情
				if (this.selectedTask) {
					const updatedSelectedTask = updatedTasks.find(t => t.id === this.selectedTask.id);
					if (updatedSelectedTask) {
						this.selectedTask = updatedSelectedTask;
					}
				}
			} catch (error) {
				console.error('轮询任务状态失败:', error);
				// 如果发生错误，暂停一段时间后再尝试
				this.stopPolling();
				setTimeout(() => this.startPolling(), 5000); // 5秒后重试
			}
		},

		// Authentication
		async login() {
			this.loggingIn = true;
			this.loginError = null;

			try {
				const response = await api.post('/auth/login', this.credentials);
				this.user = response.data.user;
				localStorage.setItem('token', response.data.token);
				localStorage.setItem('user', JSON.stringify(response.data.user));
				this.isAuthenticated = true;
				this.loadInitialData();
				this.startPolling(); // 登录成功后开始轮询
			} catch (error) {
				console.error('Login error:', error);
				this.loginError = error.response?.data || 'Login failed. Please check your credentials.';
			} finally {
				this.loggingIn = false;
				this.credentials.password = '';
			}
		},

		logout() {
			this.stopPolling(); // 登出时停止轮询
			localStorage.removeItem('token');
			localStorage.removeItem('user');
			this.isAuthenticated = false;
			this.user = null;
			this.activeTab = 'dashboard';
		},

		hasPermission(permission) {
			if (!this.user) return false;

			const permissionMap = {
				'ROOT': [
					'VIEW_TASK', 'CREATE_TASK', 'RUN_TASK', 'STOP_TASK', 'DELETE_TASK',
					'VIEW_USER', 'CREATE_USER', 'UPDATE_USER', 'DELETE_USER',
					'SYSTEM_CONFIG'
				],
				'ADMIN': [
					'VIEW_TASK', 'CREATE_TASK', 'RUN_TASK', 'STOP_TASK', 'DELETE_TASK',
					'VIEW_USER'
				],
				'USER': [
					'VIEW_TASK', 'CREATE_TASK', 'RUN_TASK', 'STOP_TASK'
				],
				'GUEST': [
					'VIEW_TASK'
				]
			};

			const userPerms = permissionMap[this.user.role] || [];
			return userPerms.includes(permission);
		},

		// Data Loading
		async loadInitialData() {
			this.loadTasks();
			if (this.hasPermission('VIEW_USER')) {
				this.loadUsers();
			}
		},

		// Tasks
		async loadTasks() {
			try {
				const response = await api.get('/tasks');
				this.tasks = response.data;
				this.recentTasks = [...this.tasks]
					.sort((a, b) => b.start_time - a.start_time)
					.slice(0, 5);

				// Update stats
				this.stats.totalTasks = this.tasks.length;
				this.stats.runningTasks = this.tasks.filter(t => t.status === 'RUNNING').length;
				this.stats.completedTasks = this.tasks.filter(t => t.status === 'COMPLETED').length;
				this.stats.failedTasks = this.tasks.filter(t => t.status === 'FAILED').length;
			} catch (error) {
				console.error('Failed to load tasks:', error);
			}
		},

		viewTask(task) {
			this.selectedTask = task;
		},

		async startTask(task) {
			try {
				await api.post(`/tasks/${task.id}/start`);
				await this.loadTasks();
				// Refresh the selected task if necessary
				if (this.selectedTask && this.selectedTask.id === task.id) {
					const response = await api.get(`/tasks/${task.id}`);
					this.selectedTask = response.data;
				}
			} catch (error) {
				console.error('Failed to start task:', error);
				alert('Failed to start task: ' + (error.response?.data || error.message));
			}
		},

		async stopTask(task) {
			try {
				await api.post(`/tasks/${task.id}/stop`);
				await this.loadTasks();
				// Refresh the selected task if necessary
				if (this.selectedTask && this.selectedTask.id === task.id) {
					const response = await api.get(`/tasks/${task.id}`);
					this.selectedTask = response.data;
				}
			} catch (error) {
				console.error('Failed to stop task:', error);
				alert('Failed to stop task: ' + (error.response?.data || error.message));
			}
		},

		addCommand() {
			this.newTask.commands.push({ command: '', args: '' });
		},

		removeCommand(index) {
			if (this.newTask.commands.length > 1) {
				this.newTask.commands.splice(index, 1);
			}
		},

		submitCreateTask() {
			// Process the form data
			const task = {
				name: this.newTask.name,
				work_dir: this.newTask.work_dir,
				commands: this.newTask.commands.map(cmd => {
					return {
						command: cmd.command,
						args: cmd.args.split(' ').filter(arg => arg.trim() !== '')
					};
				})
			};

			// Create the task
			api.post('/tasks', task)
				.then(response => {
					this.showCreateTaskModal = false;
					this.loadTasks();
					// Reset form
					this.newTask = {
						name: '',
						work_dir: '',
						commands: [{ command: '', args: '' }]
					};
				})
				.catch(error => {
					console.error('Failed to create task:', error);
					alert('Failed to create task: ' + (error.response?.data || error.message));
				});
		},

		// Users
		async loadUsers() {
			try {
				const response = await api.get('/users');
				this.users = response.data;
			} catch (error) {
				console.error('Failed to load users:', error);
			}
		},

		showCreateUserModal() {
			console.log('showCreateUserModal called');
			this.userForm = {
				id: null,
				username: '',
				password: '',
				role: 'USER',
				active: true
			};
			this.showUserModal = true;
		},

		editUser(user) {
			this.userForm = {
				id: user.id,
				username: user.username,
				password: '',
				role: user.role,
				active: user.active
			};
			this.showUserModal = true;
		},

		submitUserForm() {
			console.log('submitUserForm called with form data:', this.userForm);

			if (this.userForm.id) {
				// Update existing user
				const updates = {
					username: this.userForm.username,
					role: this.userForm.role,
					active: this.userForm.active
				};

				if (this.userForm.password) {
					updates.password = this.userForm.password;
				}

				console.log('Updating user with data:', updates);

				api.put(`/users/${this.userForm.id}`, updates)
					.then(response => {
						console.log('User updated successfully:', response.data);
						this.showUserModal = false;
						this.loadUsers();
					})
					.catch(error => {
						console.error('Failed to update user:', error);
						alert('Failed to update user: ' + (error.response?.data || error.message));
					});
			} else {
				// Create new user
				const newUser = {
					username: this.userForm.username,
					password: this.userForm.password,
					role: String(this.userForm.role) // 确保role是字符串类型
				};

				console.log('Creating new user with data:', JSON.stringify(newUser));

				api.post('/users', newUser)
					.then(response => {
						console.log('User created successfully:', response.data);
						this.showUserModal = false;
						this.loadUsers();
					})
					.catch(error => {
						console.error('Failed to create user:', error);
						if (error.response) {
							console.error('Error response:', error.response.data);
							console.error('Status:', error.response.status);
						}
						alert('Failed to create user: ' + (error.response?.data || error.message));
					});
			}
		},

		generateApiKey(user) {
			api.post(`/users/${user.id}/api-key`)
				.then(response => {
					this.apiKeyModal = {
						show: true,
						username: user.username,
						key: response.data.api_key
					};
					this.loadUsers();
				})
				.catch(error => {
					console.error('Failed to generate API key:', error);
					alert('Failed to generate API key: ' + (error.response?.data || error.message));
				});
		},

		copyApiKey() {
			const textArea = document.createElement('textarea');
			textArea.value = this.apiKeyModal.key;
			document.body.appendChild(textArea);
			textArea.select();
			document.execCommand('copy');
			document.body.removeChild(textArea);
			alert('API key copied to clipboard!');
		},

		confirmDeleteUser(user) {
			this.deleteUserModal = {
				show: true,
				id: user.id,
				username: user.username
			};
		},

		deleteUser() {
			api.delete(`/users/${this.deleteUserModal.id}`)
				.then(response => {
					this.deleteUserModal.show = false;
					this.loadUsers();
				})
				.catch(error => {
					console.error('Failed to delete user:', error);
					alert('Failed to delete user: ' + (error.response?.data || error.message));
				});
		},

		// Utility functions
		formatTime(timestamp) {
			if (!timestamp) return 'N/A';
			return new Date(timestamp * 1000).toLocaleString();
		}
	}
});