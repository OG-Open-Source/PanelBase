/**
 * PanelBase Main JavaScript
 */

document.addEventListener('DOMContentLoaded', function() {
    // Check if user is authenticated
    checkAuthentication();
    
    // Register logout handler
    setupLogoutHandler();
    
    // Load system info
    loadSystemInfo();
});

/**
 * Checks if user is authenticated, redirects to login page if not
 */
function checkAuthentication() {
    // Skip check on login page
    if (window.location.pathname === '/login') {
        return;
    }
    
    const token = localStorage.getItem('token');
    const tokenExpiry = localStorage.getItem('tokenExpiry');
    
    // If no token or token is expired, redirect to login
    if (!token || (tokenExpiry && new Date(tokenExpiry) < new Date())) {
        window.location.href = '/login';
        return;
    }
}

/**
 * Sets up the logout link handler
 */
function setupLogoutHandler() {
    const logoutLink = document.querySelector('a[href="/logout"]');
    if (logoutLink) {
        logoutLink.addEventListener('click', function(e) {
            e.preventDefault();
            
            // Clear authentication data
            localStorage.removeItem('token');
            localStorage.removeItem('tokenExpiry');
            
            // Redirect to login page
            window.location.href = '/login';
        });
    }
}

/**
 * Loads system information via API
 */
function loadSystemInfo() {
    const infoTable = document.querySelector('.info-table');
    if (!infoTable) {
        return;
    }
    
    const token = localStorage.getItem('token');
    if (!token) {
        return;
    }
    
    // Fetch system info from API
    fetch('/api/system/info', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to load system info');
        }
        return response.json();
    })
    .then(data => {
        // Update system info in the UI
        const tableContent = infoTable.querySelector('tbody') || infoTable;
        
        // Replace table content with the fetched data
        let html = '';
        for (const [key, value] of Object.entries(data)) {
            // Convert key from camelCase to Title Case with spaces
            const formattedKey = key
                .replace(/([A-Z])/g, ' $1')
                .replace(/^./, str => str.toUpperCase());
                
            html += `
                <tr>
                    <td>${formattedKey}:</td>
                    <td>${value}</td>
                </tr>
            `;
        }
        
        tableContent.innerHTML = html;
    })
    .catch(error => {
        console.error('Error loading system info:', error);
    });
}

/**
 * Helper function to make authenticated API requests
 * @param {string} url - API endpoint URL
 * @param {Object} options - Fetch options
 * @returns {Promise} - Fetch promise
 */
function apiRequest(url, options = {}) {
    const token = localStorage.getItem('token');
    
    // Set default headers
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
    };
    
    // Merge with provided options
    const requestOptions = {
        ...options,
        headers: {
            ...headers,
            ...(options.headers || {})
        }
    };
    
    return fetch(url, requestOptions)
        .then(response => {
            if (response.status === 401) {
                // Token expired or invalid, redirect to login
                localStorage.removeItem('token');
                localStorage.removeItem('tokenExpiry');
                window.location.href = '/login';
                throw new Error('Unauthorized');
            }
            return response;
        });
} 