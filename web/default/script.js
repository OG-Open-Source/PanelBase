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

		// Create HTML with theme information
		const themeInfoHTML =
			'<div class="theme-info-content">' +
				'<p><strong>Name:</strong> ' + data.name + '</p>' +
				'<p><strong>Version:</strong> ' + data.version + '</p>' +
				'<p><strong>Authors:</strong> ' + data.authors + '</p>' +
				'<p><strong>Description:</strong> ' + data.description + '</p>' +
				'<p><strong>Source:</strong> <a href="' + data.source_link + '" target="_blank">GitHub Repository</a></p>' +
			'</div>';

		// Update the theme info element
		themeInfoElement.innerHTML = themeInfoHTML;

	} catch (error) {
		console.error('Error fetching theme information:', error);
		themeInfoElement.innerHTML = '<p>Error loading theme information: ' + error.message + '</p>';
	}
}