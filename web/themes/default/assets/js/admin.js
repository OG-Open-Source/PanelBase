/**
 * PanelBase Admin JavaScript
 */

document.addEventListener('DOMContentLoaded', function() {
    // Check if user is authenticated and has admin rights
    checkAdminAccess();
    
    // Setup danger buttons with confirmation
    setupDangerButtons();
});

/**
 * Checks if user is authenticated and has admin rights
 */
function checkAdminAccess() {
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/login';
        return;
    }
    
    // Decode JWT token to check user role
    // (This is just for UI purposes, actual access control happens on the server)
    try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        if (payload.role !== 'admin') {
            // Not an admin, redirect to dashboard
            window.location.href = '/';
        }
    } catch (e) {
        console.error('Error checking admin access:', e);
        window.location.href = '/login';
    }
}

/**
 * Setup confirmation for dangerous operations
 */
function setupDangerButtons() {
    const dangerButtons = document.querySelectorAll('.danger-button');
    
    dangerButtons.forEach(button => {
        button.addEventListener('click', function(e) {
            const action = button.textContent.trim();
            const confirmed = confirm(`Are you sure you want to ${action}? This action cannot be undone.`);
            
            if (!confirmed) {
                e.preventDefault();
                return;
            }
            
            // Simulate API call (in a real app, this would call the actual API)
            console.log(`Executing action: ${action}`);
            alert(`${action} command sent to server. This may take a moment to complete.`);
        });
    });
}

/**
 * Load user management interface
 */
function loadUserManagement() {
    const token = localStorage.getItem('token');
    
    fetch('/api/users', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to load users');
        }
        return response.json();
    })
    .then(users => {
        // Render user list
        const userList = document.getElementById('user-list');
        if (userList) {
            let html = '';
            users.forEach(user => {
                html += `
                    <tr>
                        <td>${user.id}</td>
                        <td>${user.username}</td>
                        <td>${user.role}</td>
                        <td>${user.email || '-'}</td>
                        <td>
                            <button class="edit-user" data-id="${user.id}">Edit</button>
                            <button class="delete-user danger-button" data-id="${user.id}">Delete</button>
                        </td>
                    </tr>
                `;
            });
            userList.innerHTML = html;
            
            // Attach event handlers to new buttons
            setupUserActionHandlers();
        }
    })
    .catch(error => {
        console.error('Error loading users:', error);
    });
}

/**
 * Setup event handlers for user management actions
 */
function setupUserActionHandlers() {
    // Edit user buttons
    document.querySelectorAll('.edit-user').forEach(button => {
        button.addEventListener('click', function() {
            const userId = this.getAttribute('data-id');
            window.location.href = `/users/edit/${userId}`;
        });
    });
    
    // Delete user buttons
    document.querySelectorAll('.delete-user').forEach(button => {
        button.addEventListener('click', function() {
            const userId = this.getAttribute('data-id');
            const confirmed = confirm(`Are you sure you want to delete this user? This action cannot be undone.`);
            
            if (!confirmed) {
                return;
            }
            
            const token = localStorage.getItem('token');
            
            fetch(`/api/users/${userId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to delete user');
                }
                return response.json();
            })
            .then(() => {
                // Reload user list
                loadUserManagement();
            })
            .catch(error => {
                console.error('Error deleting user:', error);
                alert('Failed to delete user: ' + error.message);
            });
        });
    });
} 