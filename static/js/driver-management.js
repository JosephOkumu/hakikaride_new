// Global variables for driver management
let allDrivers = [];
let isEditMode = false;

// Load drivers when the page loads
document.addEventListener('DOMContentLoaded', () => {
    loadDrivers();
    initializeEventListeners();
});

function loadDrivers() {
    fetch('/api/admin/drivers/list')
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
            <td>${driver.firstName}</td>
            <td>${driver.lastName}</td>
            <td>${driver.phoneNumber}</td>
            <td>${driver.busNumberPlate || 'Not Assigned'}</td>
            <td>
                <button class="btn-edit" data-driver='${JSON.stringify(driver)}'>
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn-delete" data-id="${driver.driverId}">
                    <i class="fas fa-trash"></i> Delete
                </button>
            </td>
        `;
        tbody.appendChild(row);
    });

    // Add event listeners to the new buttons
    document.querySelectorAll('.btn-edit').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const driver = JSON.parse(e.currentTarget.dataset.driver);
            editDriver(driver);
        });
    });

    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const driverId = e.currentTarget.dataset.id;
            deleteDriver(driverId);
        });
    });
}

function initializeEventListeners() {
    // Modal elements
    const driverModal = document.getElementById('driverModal');
    const closeBtn = document.querySelector('.close');
    const driverForm = document.getElementById('driverForm');
    const addDriverBtn = document.getElementById('addDriverBtn');
    const searchInput = document.getElementById('driverSearchInput');

    // Add Driver button
    if (addDriverBtn) {
        addDriverBtn.addEventListener('click', () => {
            isEditMode = false;
            document.getElementById('modalTitle').textContent = 'Add New Driver';
            document.getElementById('submitDriverBtn').textContent = 'Add Driver';
            driverForm.reset();
            document.getElementById('driverId').value = '';
            document.getElementById('userId').value = '';
            driverModal.style.display = 'block';
        });
    }

    // Close modal button
    if (closeBtn) {
        closeBtn.addEventListener('click', () => {
            driverModal.style.display = 'none';
        });
    }

    // Click outside modal to close
    window.addEventListener('click', (e) => {
        if (e.target === driverModal) {
            driverModal.style.display = 'none';
        }
    });

    // Form submission
    if (driverForm) {
        driverForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = {
                driverId: document.getElementById('driverId').value,
                userId: document.getElementById('userId').value,
                firstName: document.getElementById('firstName').value,
                lastName: document.getElementById('lastName').value,
                phoneNumber: document.getElementById('phoneNumber').value
            };

            const url = isEditMode ? '/api/admin/drivers/update' : '/api/admin/drivers/add';
            const method = isEditMode ? 'PUT' : 'POST';

            try {
                const response = await fetch(url, {
                    method: method,
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(formData)
                });

                const data = await response.json();
                if (data.success) {
                    driverModal.style.display = 'none';
                    driverForm.reset();
                    loadDrivers();
                } else {
                    alert('Error: ' + data.message);
                }
            } catch (error) {
                console.error('Error:', error);
                alert('Error processing request');
            }
        });
    }


}

function editDriver(driver) {
    isEditMode = true;
    document.getElementById('modalTitle').textContent = 'Edit Driver';
    document.getElementById('submitDriverBtn').textContent = 'Update Driver';
    
    // Fill form fields
    document.getElementById('driverId').value = driver.driverId;
    document.getElementById('userId').value = driver.userId;
    document.getElementById('firstName').value = driver.firstName;
    document.getElementById('lastName').value = driver.lastName;
    document.getElementById('phoneNumber').value = driver.phoneNumber;
    
    document.getElementById('driverModal').style.display = 'block';
}

function deleteDriver(driverId) {
    if (confirm('Are you sure you want to delete this driver?')) {
        fetch(`/api/admin/drivers/delete?id=${driverId}`, {
            method: 'DELETE'
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
