// Global variables
let allDrivers = [];
let isEditMode = false;

// Initialize when the page loads
document.addEventListener('DOMContentLoaded', function() {
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
            
            const formData = {
                firstName: document.getElementById('firstName').value,
                lastName: document.getElementById('lastName').value,
                phoneNumber: document.getElementById('phoneNumber').value,
                busNumberPlate: document.getElementById('busId').value
            };

            try {
                const token = localStorage.getItem('token');
                const url = isEditMode ? 
                    `/api/admin/drivers/update/${document.getElementById('driverId').value}` : 
                    '/api/admin/drivers/add';
                
                const method = isEditMode ? 'PUT' : 'POST';
                
                const response = await fetch(url, {
                    method: method,
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify(formData)
                });

                const data = await response.json();
                if (data.success) {
                    document.getElementById('driverModal').style.display = 'none';
                    driverForm.reset();
                    loadDrivers();
                } else {
                    alert('Error: ' + (data.message || 'Failed to process driver data'));
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
    fetch('/api/admin/drivers/list', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            allDrivers = data.drivers;
            renderDrivers(allDrivers);
        }
    })
    .catch(error => console.error('Error loading drivers:', error));
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
        fetch(`/api/admin/drivers/delete/${driverId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                loadDrivers();
            } else {
                alert('Error: ' + data.message);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            alert('Error deleting driver');
        });
    }
}

// Handle sign out
function handleSignOut() {
    // Implement sign out logic here
    window.location.href = '/';
}
