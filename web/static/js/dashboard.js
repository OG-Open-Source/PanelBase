document.addEventListener('DOMContentLoaded', () => {
    // Check if token exists, if not redirect to login
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/nucustomentry';
        return;
    }

    // Logout functionality
    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', (e) => {
            e.preventDefault();
            localStorage.removeItem('token');
            window.location.href = '/nucustomentry';
        });
    }

    // Simulate loading system stats
    setTimeout(() => {
        // Update uptime
        const uptimeElem = document.getElementById('uptimeValue');
        if (uptimeElem) {
            uptimeElem.textContent = '3 days, 7 hours, 25 minutes';
        }
        
        // Update memory usage
        const memoryElem = document.getElementById('memoryValue');
        if (memoryElem) {
            memoryElem.textContent = '1.2 GB / 8 GB (15%)';
        }
        
        // Update CPU usage
        const cpuElem = document.getElementById('cpuValue');
        if (cpuElem) {
            cpuElem.textContent = '23%';
        }
    }, 1000);

    // Simulate loading logs
    setTimeout(() => {
        const logsContainer = document.getElementById('recentLogs');
        if (logsContainer) {
            // Sample log entries
            const logs = [
                { time: '2025-03-20 21:58:32', message: 'User admin logged in', level: 'info' },
                { time: '2025-03-20 21:45:17', message: 'System backup completed', level: 'info' },
                { time: '2025-03-20 21:30:05', message: 'Failed login attempt: user123', level: 'warning' },
                { time: '2025-03-20 21:15:43', message: 'System update available', level: 'info' },
                { time: '2025-03-20 21:00:22', message: 'CPU usage spike detected (87%)', level: 'warning' },
                { time: '2025-03-20 20:45:11', message: 'Disk space warning: 85% used', level: 'warning' },
                { time: '2025-03-20 20:30:05', message: 'System restarted', level: 'info' }
            ];
            
            // Clear loading message
            logsContainer.innerHTML = '';
            
            // Append logs
            logs.forEach(log => {
                const logEntry = document.createElement('div');
                logEntry.className = `log-entry ${log.level}`;
                logEntry.innerHTML = `<span class="log-time">[${log.time}]</span> <span class="log-message">${log.message}</span>`;
                logsContainer.appendChild(logEntry);
            });
        }
    }, 1500);

    // In a real application, you would fetch these values from an API endpoint
    // Something like:
    /*
    async function fetchSystemStats() {
        try {
            const response = await fetch('/nucustomentry/api/stats', {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                // Update UI with data
            } else {
                console.error('Failed to fetch stats');
            }
        } catch (error) {
            console.error('Error:', error);
        }
    }
    
    fetchSystemStats();
    */
}); 