document.addEventListener('DOMContentLoaded', () => {
    // Simulate loading system stats
    setTimeout(() => {
        // Update system metrics
        document.getElementById('loadValue').textContent = '0.15, 0.10, 0.05';
        document.getElementById('memoryValue').textContent = '1.2 GB / 8 GB (15%)';
        document.getElementById('cpuValue').textContent = '5%';
        document.getElementById('diskValue').textContent = '25 GB / 100 GB (25%)';
        document.getElementById('uptimeValue').textContent = '7 天, 3 小時, 45 分鐘';
        
        // Update system info
        document.getElementById('hostname').textContent = 'panel-server';
        document.getElementById('os').textContent = 'Linux Ubuntu 22.04 LTS';
        document.getElementById('kernel').textContent = '5.15.0-25-generic';
        document.getElementById('ip').textContent = '10.0.0.5';
        document.getElementById('cpu').textContent = 'Intel Xeon E5-2680 @ 2.80GHz';
        document.getElementById('memory').textContent = '8 GB DDR4';
    }, 1000);
    
    // In a real application, you would fetch these values from an API
    // Something like:
    /*
    async function fetchSystemStats() {
        try {
            const response = await fetch('/nucurtomentry/api/stats', {
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