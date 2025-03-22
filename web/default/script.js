document.addEventListener('DOMContentLoaded', () => {
	// Fetch theme information from the API
	fetchThemeInfo();
});

// Function to fetch theme information
async function fetchThemeInfo() {
	const themeInfoElement = document.getElementById('theme-info');

	try {
		// Get current path to build the correct API endpoint
		const path = window.location.pathname;
		const entryPath = path.split('/')[1]; // Get the entry path from URL

		// Fetch theme info from API
		const response = await fetch("/" + entryPath + "/theme/info");

		if (!response.ok) {
			throw new Error("HTTP error! Status: " + response.status);
		}

		const data = await response.json();

		// 创建包含所有主题信息的 HTML
		let themeInfoHTML = '<div class="theme-info-content">';

		// 基本信息
		themeInfoHTML +=
			'<h3>基本信息</h3>' +
			'<p><strong>名称:</strong> ' + data.name + '</p>' +
			'<p><strong>版本:</strong> ' + data.version + '</p>' +
			'<p><strong>作者:</strong> ' + data.authors + '</p>' +
			'<p><strong>描述:</strong> ' + data.description + '</p>' +
			'<p><strong>源代码:</strong> <a href="' + data.source_link + '" target="_blank">GitHub 仓库</a></p>';

		// 目录信息
		themeInfoHTML +=
			'<h3>主题目录</h3>' +
			'<p><strong>目录:</strong> ' + data.directory + '</p>';

		// 结构信息
		themeInfoHTML += '<h3>主题结构</h3><ul>';

		for (const [key, value] of Object.entries(data.structure)) {
			themeInfoHTML += '<li><strong>' + key + ':</strong> ' + value + '</li>';
		}

		themeInfoHTML += '</ul></div>';

		// 更新主题信息元素
		themeInfoElement.innerHTML = themeInfoHTML;

	} catch (error) {
		console.error('Error fetching theme information:', error);
		themeInfoElement.innerHTML = '<p>Error loading theme information: ' + error.message + '</p>';
	}
}