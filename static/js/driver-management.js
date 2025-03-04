// Global variables
let allDrivers = [];
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
    loadDrivers();
});

function initializeEventListeners() {
    // Add event listener for opening the modal
    const addDriverBtn = document.getElementById('addDriverBtn');
    if (addDriverBtn) {
        addDriverBtn.addEventListener('click', function() {
            // Reset form and prepare for add mode
            const driverForm = document.getElementById('driverForm');
            if (driverForm) driverForm.reset();
            
            document.getElementById('driverModal').style.display = 'block';
            document.getElementById('modalTitle').textContent = 'Add New Driver';
            document.getElementById('submitDriverBtn').textContent = 'Add Driver';
            isEditMode = false;
        });
    }
    
    // Add event listener for closing the modal
    const closeButtons = document.querySelectorAll('.close');
    closeButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            document.getElementById('driverModal').style.display = 'none';
        });
    });
    
    // Form submission
    const driverForm = document.getElementById('driverForm');
    if (driverForm) {
        driverForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const token = localStorage.getItem('token');
            if (!token) {
                window.location.href = '/'; // Redirect to login if no token
                return;
            }
            
            const formData = {
                firstName: document.getElementById('firstName').value,
                lastName: document.getElementById('lastName').value,
                phoneNumber: document.getElementById('phoneNumber').value,
                busNumberPlate: document.getElementById('busId').value
            };

            try {
                const url = isEditMode ? 
                    `/api/admin/drivers/update` : 
                    '/api/admin/drivers/add';
                
                const method = isEditMode ? 'PUT' : 'POST';
                
                if (isEditMode) {
                    // For edit mode, include the driver ID in the form data
                    formData.driverId = document.getElementById('driverId').value;
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
                    document.getElementById('driverModal').style.display = 'none';
                    driverForm.reset();
                    loadDrivers();
                    // If it's a new driver, show the password to the admin
                    if (!isEditMode && data.driverDetails && data.driverDetails.initialPassword) {
                        alert(`Driver added successfully!\n\nDriver Phone: ${data.driverDetails.phoneNumber}\nInitial Password: ${data.driverDetails.initialPassword}\n\nPlease share these credentials with the driver.`);
                    }
                } else {
                    alert('Error: ' + (data.message || 'Failed to process driver data'));
                    // If unauthorized, redirect to login
                    if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                        localStorage.removeItem('token');
                        window.location.href = '/';
                    }
                }
            } catch (error) {
                console.error('Error:', error);
                alert('Error processing driver data. Please try again.');
            }
        });
    }
    
    // Close modal when clicking outside
    window.addEventListener('click', function(event) {
        const modal = document.getElementById('driverModal');
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    });
}

function loadDrivers() {
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/'; // Redirect to login if no token
        return;
    }

    fetch('/api/admin/drivers/list', {
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
            allDrivers = data.drivers || [];
            renderDrivers(allDrivers);
        } else {
            console.error('Error loading drivers:', data.message);
            // If unauthorized, redirect to login
            if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                localStorage.removeItem('token');
                window.location.href = '/';
            }
        }
    })
    .catch(error => {
        console.error('Error loading drivers:', error);
    });
}

function renderDrivers(drivers) {
    const tbody = document.getElementById('driversTableBody');
    if (!tbody) return;
    
    tbody.innerHTML = '';
    drivers.forEach(driver => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${driver.firstName || ''}</td>
            <td>${driver.lastName || ''}</td>
            <td>${driver.phoneNumber || ''}</td>
            <td>${driver.busNumberPlate || 'Not Assigned'}</td>
            <td>
                <button class="btn-edit" data-id="${driver.driverId || driver.id}">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn-delete" data-id="${driver.driverId || driver.id}">
                    <i class="fas fa-trash"></i> Delete
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });

    // Add event listeners for edit and delete buttons
    document.querySelectorAll('.btn-edit').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const driverId = e.currentTarget.dataset.id;
            const driver = allDrivers.find(d => d.driverId === driverId || d.id === driverId);
            if (driver) editDriver(driver);
        });
    });

    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const driverId = e.currentTarget.dataset.id;
            deleteDriver(driverId);
        });
    });
}

function editDriver(driver) {
    const modal = document.getElementById('driverModal');
    if (!modal) return;
    
    document.getElementById('driverId').value = driver.driverId || driver.id;
    document.getElementById('firstName').value = driver.firstName || '';
    document.getElementById('lastName').value = driver.lastName || '';
    document.getElementById('phoneNumber').value = driver.phoneNumber || '';
    document.getElementById('busId').value = driver.busNumberPlate || '';
    
    document.getElementById('modalTitle').textContent = 'Edit Driver';
    document.getElementById('submitDriverBtn').textContent = 'Update Driver';
    modal.style.display = 'block';
    isEditMode = true;
}

function deleteDriver(driverId) {
    if (confirm('Are you sure you want to delete this driver?')) {
        const token = localStorage.getItem('token');
        if (!token) {
            window.location.href = '/'; // Redirect to login if no token
            return;
        }
        
        fetch(`/api/admin/drivers/delete/${driverId}`, {
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
                loadDrivers();
            } else {
                alert('Error: ' + (data.message || 'Failed to delete driver'));
                // If unauthorized, redirect to login
                if (data.message && (data.message.includes('token') || data.message.includes('authorization'))) {
                    localStorage.removeItem('token');
                    window.location.href = '/';
                }
            }
        })
        .catch(error => {
            console.error('Error:', error);
            alert('Error deleting driver. Please try again.');
        });
    }
}

// Handle sign out
function handleSignOut() {
    // Implement sign out logic here
    window.location.href = '/';
}
