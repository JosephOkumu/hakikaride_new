// Global variables
let allBuses = [];
let allRoutes = [];
let isEditMode = false;

// Initialize when the page loads
document.addEventListener('DOMContentLoaded', function() {
    // Check if user is logged in
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/'; // Redirect to login page
        return;
    }

    initializeEventListeners();
    loadBuses();
    loadRoutes();
});

function initializeEventListeners() {
    // Add event listener for opening the modal
    const addBusBtn = document.getElementById('addBusBtn');
    if (addBusBtn) {
        addBusBtn.addEventListener('click', function() {
            // Reset form and prepare for add mode
            const busForm = document.getElementById('busForm');
            if (busForm) busForm.reset();
            
            document.getElementById('busModal').style.display = 'block';
            document.getElementById('modalTitle').textContent = 'Add New Bus';
            document.getElementById('submitBusBtn').textContent = 'Add Bus';
            isEditMode = false;
        });
    }
    
    // Add event listener for closing the modal
    const closeButtons = document.querySelectorAll('.close');
    closeButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            document.getElementById('busModal').style.display = 'none';
        });
    });
    
    // Form submission
    const busForm = document.getElementById('busForm');
    if (busForm) {
        busForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const token = localStorage.getItem('token');
            if (!token) {
                window.location.href = '/'; // Redirect to login if no token
                return;
            }
            
            const formData = {
                numberPlate: document.getElementById('numberPlate').value,
                routeId: document.getElementById('routeId').value
            };

            try {
                const url = isEditMode ? 
                    `/api/admin/buses/update` : 
                    '/api/admin/buses/add';
                
                const method = isEditMode ? 'PUT' : 'POST';
                
                if (isEditMode) {
                    // For edit mode, include the bus ID in the form data
                    formData.busId = document.getElementById('busId').value;
                }
                
                const response = await fetch(url, {
                    method: method,
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify(formData)
                });

                if (!response.ok) {
                    throw new Error('Network response was not ok: ' + response.statusText);
                }
                
                const data = await response.json();
                if (data.success) {
                    document.getElementById('busModal').style.display = 'none';
                    busForm.reset();
                    loadBuses();
                } else {
                    alert('Error: ' + (data.message || 'Failed to process bus data'));
                    // If unauthorized, redirect to login
                    if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                        localStorage.removeItem('token');
                        window.location.href = '/';
                    }
                }
            } catch (error) {
                console.error('Error:', error);
                alert('Error processing bus data. Please try again.');
            }
        });
    }
    
    // Close modal when clicking outside
    window.addEventListener('click', function(event) {
        const modal = document.getElementById('busModal');
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    });

    // Sign out functionality
    const signOutLinks = document.querySelectorAll('.nav-links li:last-child');
    signOutLinks.forEach(link => {
        link.addEventListener('click', function() {
            localStorage.removeItem('token');
            window.location.href = '/';
        });
    });
}

function loadBuses() {
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/'; // Redirect to login if no token
        return;
    }

    fetch('/api/admin/buses', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok: ' + response.statusText);
        }
        return response.json();
    })
    .then(data => {
        if (data.success) {
            allBuses = data.buses || [];
            renderBuses(allBuses);
        } else {
            console.error('Error loading buses:', data.message);
            // If unauthorized, redirect to login
            if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                localStorage.removeItem('token');
                window.location.href = '/';
            }
        }
    })
    .catch(error => {
        console.error('Error loading buses:', error);
    });
}

function loadRoutes() {
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/';
        return;
    }

    fetch('/api/admin/routes', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok: ' + response.statusText);
        }
        return response.json();
    })
    .then(data => {
        if (data.success) {
            allRoutes = data.routes || [];
            populateRouteDropdown(allRoutes);
        } else {
            console.error('Error loading routes:', data.message);
            if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                localStorage.removeItem('token');
                window.location.href = '/';
            }
        }
    })
    .catch(error => {
        console.error('Error loading routes:', error);
    });
}

// No longer need to populate dropdown as we're using an input field now
function populateRouteDropdown(routes) {
    // This function is kept as a placeholder to avoid breaking existing code references
    // but it no longer performs any dropdown population
}

function renderBuses(buses) {
    const tbody = document.getElementById('busesTableBody');
    if (!tbody) return;
    
    tbody.innerHTML = '';
    buses.forEach(bus => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${bus.plate || ''}</td>
            <td>${bus.route || 'Not Assigned'}</td>
            <td>${bus.status || 'Unknown'}</td>
            <td>
                <button class="btn-edit" data-id="${bus.id}">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn-delete" data-id="${bus.id}">
                    <i class="fas fa-trash"></i> Delete
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });

    // Add event listeners for edit and delete buttons
    document.querySelectorAll('.btn-edit').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const busId = e.currentTarget.dataset.id;
            const bus = allBuses.find(b => b.id.toString() === busId.toString());
            if (bus) editBus(bus);
        });
    });

    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const busId = e.currentTarget.dataset.id;
            deleteBus(busId);
        });
    });
}

function editBus(bus) {
    const modal = document.getElementById('busModal');
    if (!modal) return;
    
    document.getElementById('busId').value = bus.id;
    document.getElementById('numberPlate').value = bus.plate || '';
    
    // Set the route value in the input field
    document.getElementById('routeId').value = bus.route || '';
    
    document.getElementById('modalTitle').textContent = 'Edit Bus';
    document.getElementById('submitBusBtn').textContent = 'Update Bus';
    modal.style.display = 'block';
    isEditMode = true;
}

function deleteBus(busId) {
    if (!confirm('Are you sure you want to delete this bus?')) {
        return;
    }
    
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/';
        return;
    }
    
    fetch(`/api/admin/buses/delete/${busId}`, {
        method: 'DELETE',
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok: ' + response.statusText);
        }
        return response.json();
    })
    .then(data => {
        if (data.success) {
            loadBuses(); // Reload the bus list
        } else {
            alert('Error: ' + (data.message || 'Failed to delete bus'));
            if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                localStorage.removeItem('token');
                window.location.href = '/';
            }
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('Error deleting bus. Please try again.');
    });
}
