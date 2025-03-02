document.addEventListener('DOMContentLoaded', function() {
    // Check if user is authenticated
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = '/';
        return;
    }

    // DOM Elements
    const studentModal = document.getElementById('studentModal');
    const bulkUploadModal = document.getElementById('bulkUploadModal');
    const studentForm = document.getElementById('studentForm');
    const bulkUploadForm = document.getElementById('bulkUploadForm');
    const addStudentBtn = document.getElementById('addStudentBtn');
    const bulkUploadBtn = document.getElementById('bulkUploadBtn');
    const closeButtons = document.querySelectorAll('.close');
    
    // Load students on page load
    loadStudents();

    // Event Listeners
    addStudentBtn.addEventListener('click', () => {
        document.getElementById('modalTitle').textContent = 'Add New Student';
        document.getElementById('studentId').value = '';
        studentForm.reset();
        studentModal.style.display = 'block';
    });

    bulkUploadBtn.addEventListener('click', () => {
        bulkUploadModal.style.display = 'block';
    });

    closeButtons.forEach(button => {
        button.addEventListener('click', () => {
            studentModal.style.display = 'none';
            bulkUploadModal.style.display = 'none';
        });
    });

    // Handle student form submission
    studentForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const studentId = document.getElementById('studentId').value;
        const studentData = {
            studentId: studentId ? parseInt(studentId) : null,
            parentId: parseInt(document.getElementById('parentId').value),
            firstName: document.getElementById('firstName').value,
            lastName: document.getElementById('lastName').value,
            grade: document.getElementById('grade').value,
            admNumber: document.getElementById('admNumber').value,
            address: document.getElementById('address').value,
            emergencyContact: document.getElementById('emergencyContact').value
        };

        try {
            const response = await fetch(`/api/admin/students/${studentId ? 'update' : 'add'}`, {
                method: studentId ? 'PUT' : 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify(studentData)
            });

            const result = await response.json();
            if (result.success) {
                alert(studentId ? 'Student updated successfully' : 'Student added successfully');
                studentModal.style.display = 'none';
                loadStudents();
            } else {
                alert('Error: ' + result.message);
            }
        } catch (error) {
            alert('Error processing request');
            console.error('Error:', error);
        }
    });

    // Handle bulk upload form submission
    bulkUploadForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData();
        formData.append('file', document.getElementById('csvFile').files[0]);

        try {
            const response = await fetch('/api/admin/students/bulk-upload', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                },
                body: formData
            });

            const result = await response.json();
            if (result.success) {
                alert(result.message);
                bulkUploadModal.style.display = 'none';
                loadStudents();
            } else {
                alert('Error: ' + result.message);
            }
        } catch (error) {
            alert('Error uploading file');
            console.error('Error:', error);
        }
    });
});

// Load students from the server
async function loadStudents() {
    try {
        const response = await fetch('/api/admin/students/list', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            }
        });
        const data = await response.json();
        
        if (data.success) {
            const tableBody = document.getElementById('studentsTableBody');
            tableBody.innerHTML = '';
            
            data.students.forEach(student => {
                const row = document.createElement('tr');
                const cellStyle = 'padding: 1rem; border-bottom: 1px solid var(--border-color);';
                row.innerHTML = `
                    <td style="${cellStyle}">${student.admNumber}</td>
                    <td style="${cellStyle}; min-width: 200px;">${student.firstName} ${student.lastName}</td>
                    <td style="${cellStyle}">${student.grade}</td>
                    <td style="${cellStyle}">${student.address}</td>
                    <td style="${cellStyle}">${student.emergencyContact}</td>
                `;
                tableBody.appendChild(row);
            });
        }
    } catch (error) {
        console.error('Error loading students:', error);
    }
}

// Edit student
function editStudent(student) {
    document.getElementById('modalTitle').textContent = 'Edit Student';
    document.getElementById('studentId').value = student.studentId;
    document.getElementById('firstName').value = student.firstName;
    document.getElementById('lastName').value = student.lastName;
    document.getElementById('grade').value = student.grade;
    document.getElementById('admNumber').value = student.admNumber;
    document.getElementById('address').value = student.address;
    document.getElementById('emergencyContact').value = student.emergencyContact;
    document.getElementById('parentId').value = student.parentId;
    
    document.getElementById('studentModal').style.display = 'block';
}

// Delete student
async function deleteStudent(studentId) {
    if (confirm('Are you sure you want to delete this student?')) {
        try {
            const response = await fetch(`/api/admin/students/delete?id=${studentId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('token')}`
                }
            });
            
            const result = await response.json();
            if (result.success) {
                alert('Student deleted successfully');
                loadStudents();
            } else {
                alert('Error: ' + result.message);
            }
        } catch (error) {
            alert('Error deleting student');
            console.error('Error:', error);
        }
    }
}
