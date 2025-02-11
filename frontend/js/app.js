const { createApp, ref, computed, onMounted } = Vue;

const app = createApp({
	setup() {
		const isAuthenticated = ref(false);
		const isDark = ref(false);
		const token = ref('');
		const connection = ref({
			host: '',
			port: 80,
			securityCode: ''
		});
		const keys = ref({
			key1: '',
			key2: '',
			key3: ''
		});
		const command = ref('');
		const commandOutput = ref('');
		const shareConfig = ref({
			maxUsers: 1,
			expiryHours: 24,
			permissions: []
		});
		const shareToken = ref('');

		// Compute API base URL from connection settings
		const apiBaseUrl = computed(() => {
			const protocol = connection.value.port === 443 ? 'https' : 'http';
			return `${protocol}://${connection.value.host}:${connection.value.port}/cgi-bin`;
		});

		// Theme management
		const toggleTheme = () => {
			isDark.value = !isDark.value;
			document.documentElement.classList.toggle('dark', isDark.value);
			localStorage.setItem('theme', isDark.value ? 'dark' : 'light');
		};

		// Initialize theme and connection from localStorage
		onMounted(() => {
			const savedTheme = localStorage.getItem('theme');
			if (savedTheme) {
				isDark.value = savedTheme === 'dark';
				document.documentElement.classList.toggle('dark', isDark.value);
			}

			// Load saved connection settings
			const savedConnection = localStorage.getItem('connection');
			if (savedConnection) {
				connection.value = JSON.parse(savedConnection);
			}

			// Check for existing token
			const savedToken = localStorage.getItem('token');
			if (savedToken) {
				token.value = savedToken;
				isAuthenticated.value = true;
			}
		});

		// Authentication
		const login = async () => {
			// Validate connection settings
			if (!connection.value.host || !connection.value.port || !connection.value.securityCode) {
				alert('Please fill in all connection settings');
				return;
			}

			// Validate security code format
			if (!/^\d{8}$/.test(connection.value.securityCode)) {
				alert('Security code must be 8 digits');
				return;
			}

			try {
				const response = await axios.post(`${apiBaseUrl.value}/token.cgi`, {
					keys: keys.value,
					security_code: connection.value.securityCode
				});

				if (response.data.status === 200) {
					token.value = response.data.token.token;
					localStorage.setItem('token', token.value);
					localStorage.setItem('connection', JSON.stringify(connection.value));
					isAuthenticated.value = true;
					keys.value = { key1: '', key2: '', key3: '' };
					connection.value.securityCode = ''; // Clear security code
				}
			} catch (error) {
				alert('Login failed: ' + (error.response?.data?.error || error.message));
			}
		};

		const logout = () => {
			token.value = '';
			localStorage.removeItem('token');
			isAuthenticated.value = false;
		};

		// Command execution
		const executeCommand = async () => {
			if (!command.value) return;

			try {
				const response = await axios.post(`${apiBaseUrl.value}/panel.cgi`, {
					token: token.value,
					command: command.value
				});

				if (response.data.status === 200) {
					commandOutput.value = response.data.output;
					command.value = '';
				}
			} catch (error) {
				if (error.response?.status === 401) {
					logout();
				}
				alert('Command execution failed: ' + (error.response?.data?.error || error.message));
			}
		};

		// Share token management
		const createShareToken = async () => {
			try {
				const response = await axios.post(`${apiBaseUrl.value}/share.cgi`, {
					token: token.value,
					action: 'create',
					max_users: shareConfig.value.maxUsers,
					expiry: Math.floor(Date.now() / 1000) + (shareConfig.value.expiryHours * 3600),
					permissions: shareConfig.value.permissions
				});

				if (response.data.status === 200) {
					shareToken.value = response.data.share_token;
				}
			} catch (error) {
				if (error.response?.status === 401) {
					logout();
				}
				alert('Failed to create share token: ' + (error.response?.data?.error || error.message));
			}
		};

		const copyShareToken = () => {
			navigator.clipboard.writeText(shareToken.value)
				.then(() => alert('Token copied to clipboard!'))
				.catch(err => alert('Failed to copy token: ' + err.message));
		};

		// Token refresh mechanism
		const refreshToken = async () => {
			try {
				const response = await axios.post(`${apiBaseUrl.value}/token.cgi`, {
					token: token.value
				});

				if (response.data.status === 200) {
					token.value = response.data.token.token;
					localStorage.setItem('token', token.value);
				}
			} catch (error) {
				if (error.response?.status === 401) {
					logout();
				}
			}
		};

		// Set up token refresh interval
		onMounted(() => {
			setInterval(refreshToken, 1000 * 60 * 60); // Refresh token every hour
		});

		return {
			isAuthenticated,
			isDark,
			connection,
			keys,
			command,
			commandOutput,
			shareConfig,
			shareToken,
			login,
			logout,
			toggleTheme,
			executeCommand,
			createShareToken,
			copyShareToken
		};
	}
});

app.mount('#app');