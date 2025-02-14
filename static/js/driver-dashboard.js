class DriverDashboard {
    constructor() {
        this.students = [];
        this.tripActive = false;
        this.initializeEventListeners();
    }

    async initializeEventListeners() {
        // Load driver information
        await this.loadDriverInfo();
        
        // Initialize student checklist
        await this.loadStudentList();
        
        // Set up WebSocket event handlers for real-time updates
        this.setupWebSocketHandlers();
    }

    async loadDriverInfo() {
        try {
            const response = await fetch('/api/driver/info');
            const data = await response.json();
            
            if (data.success) {
                document.getElementById('driverName').textContent = 
                    `${data.firstName} ${data.lastName}`;
            }
        } catch (error) {
            console.error('Error loading driver info:', error);
        }
    }

    async loadStudentList() {
        try {
            const response = await fetch('/api/driver/students');
            const data = await response.json();
            
            if (data.success) {
                this.students = data.students;
                this.renderStudentChecklist();
            }
        } catch (error) {
            console.error('Error loading student list:', error);
        }
    }

    renderStudentChecklist() {
        const container = document.getElementById('studentChecklistContainer');
        container.innerHTML = '';

        this.students.forEach(student => {
            const studentCard = document.createElement('div');
            studentCard.className = 'student-card';
            studentCard.innerHTML = `
                <div class="student-info">
                    <h4>${student.firstName} ${student.lastName}</h4>
                    <p>Grade: ${student.grade}</p>
                    <p>Admission: ${student.admNumber}</p>
                </div>
                <div class="pickup-status">
                    <label class="checkbox-container">
                        <input type="checkbox" 
                               data-student-id="${student.studentId}"
                               ${student.isPickedUp ? 'checked' : ''}
                               onchange="driverDashboard.updateStudentStatus(${student.studentId}, this.checked)">
                        <span class="checkmark"></span>
                        ${student.isPickedUp ? 'Picked Up' : 'Not Picked Up'}
                    </label>
                </div>
            `;
            container.appendChild(studentCard);
        });

        this.updateStudentCount();
    }

    async updateStudentStatus(studentId, isPickedUp) {
        try {
            const response = await fetch('/api/driver/student-status', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    studentId,
                    isPickedUp,
                    timestamp: new Date().toISOString()
                })
            });

            const data = await response.json();
            if (data.success) {
                // Update local student data
                const student = this.students.find(s => s.studentId === studentId);
                if (student) {
                    student.isPickedUp = isPickedUp;
                    this.updateStudentCount();
                }
            }
        } catch (error) {
            console.error('Error updating student status:', error);
            alert('Failed to update student status. Please try again.');
        }
    }

    updateStudentCount() {
        const pickedUpCount = this.students.filter(s => s.isPickedUp).length;
        document.getElementById('studentCount').textContent = pickedUpCount;
    }

    setupWebSocketHandlers() {
        const websocket = new WebSocket(`${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/driver`);
        
        websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            
            switch (data.type) {
                case 'student_update':
                    this.handleStudentUpdate(data.data);
                    break;
                case 'emergency_response':
                    this.handleEmergencyResponse(data.data);
                    break;
                case 'trip_status':
                    this.handleTripStatusUpdate(data.data);
                    break;
            }
        };
    }

    handleStudentUpdate(data) {
        const student = this.students.find(s => s.studentId === data.studentId);
        if (student) {
            student.isPickedUp = data.isPickedUp;
            this.renderStudentChecklist();
        }
    }

    handleEmergencyResponse(data) {
        alert(`Emergency Response: ${data.message}`);
    }

    handleTripStatusUpdate(data) {
        this.tripActive = data.isActive;
        document.getElementById('tripStatus').textContent = data.status;
    }
}

// Initialize the dashboard when the page loads
const driverDashboard = new DriverDashboard();
