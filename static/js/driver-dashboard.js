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
            const responseText = await response.text();
            
            try {
                const data = JSON.parse(responseText);
                if (data.success) {
                    document.getElementById('driverName').textContent = 
                        `${data.firstName} ${data.lastName}`;
                } else {
                    // For development, show a placeholder
                    document.getElementById('driverName').textContent = 'John Driver';
                }
            } catch (jsonError) {
                console.error('Error parsing driver info JSON:', jsonError, 'Raw response:', responseText);
                // For development, set a placeholder name
                document.getElementById('driverName').textContent = 'John Driver';
            }
        } catch (error) {
            console.error('Error loading driver info:', error);
            // For development, set a placeholder name
            document.getElementById('driverName').textContent = 'John Driver';
        }
    }

    async loadStudentList() {
        try {
            const response = await fetch('/api/driver/students');
            const responseText = await response.text();
            
            try {
                const data = JSON.parse(responseText);
                if (data.success) {
                    this.students = data.students;
                } else {
                    // For development, use mock data
                    this.students = [
                        { id: 1, name: 'Alice Smith', isPickedUp: false },
                        { id: 2, name: 'Bob Johnson', isPickedUp: false },
                        { id: 3, name: 'Carol Williams', isPickedUp: false },
                        { id: 4, name: 'David Brown', isPickedUp: false }
                    ];
                }
                this.renderStudentChecklist();
            } catch (jsonError) {
                console.error('Error parsing student list JSON:', jsonError, 'Raw response:', responseText);
                // For development, use mock data
                this.students = [
                    { id: 1, name: 'Alice Smith', isPickedUp: false },
                    { id: 2, name: 'Bob Johnson', isPickedUp: false },
                    { id: 3, name: 'Carol Williams', isPickedUp: false },
                    { id: 4, name: 'David Brown', isPickedUp: false }
                ];
                this.renderStudentChecklist();
            }
        } catch (error) {
            console.error('Error loading student list:', error);
            // For development, use mock data
            this.students = [
                { id: 1, name: 'Alice Smith', isPickedUp: false },
                { id: 2, name: 'Bob Johnson', isPickedUp: false },
                { id: 3, name: 'Carol Williams', isPickedUp: false },
                { id: 4, name: 'David Brown', isPickedUp: false }
            ];
            this.renderStudentChecklist();
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
        try {
            const debugOutput = document.getElementById('debugOutput');
            
            const logMessage = (msg, isError = false) => {
                if (debugOutput) {
                    const newLine = document.createTextNode(msg + '\n');
                    debugOutput.appendChild(newLine);
                    debugOutput.scrollTop = debugOutput.scrollHeight;
                    
                    if (isError) {
                        console.error(msg);
                    } else {
                        console.log(msg);
                    }
                }
            };
            
            logMessage('Setting up WebSocket connection...');
            
            // Close any existing connection
            if (this.websocket) {
                try {
                    this.websocket.close();
                } catch (e) {
                    // Ignore errors when closing
                }
                this.websocket = null;
            }
            
            // Simple WebSocket creation
            const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsHost = window.location.host;
            const wsUrl = `${wsProtocol}//${wsHost}/ws`;
            
            logMessage(`WebSocket URL: ${wsUrl}`);
            
            // Create a new WebSocket
            const websocket = new WebSocket(wsUrl);
            
            websocket.onopen = () => {
                logMessage('WebSocket connection established');
                document.getElementById('connectionStatus').classList.add('connected');
                
                // Send authentication message
                const authMessage = JSON.stringify({
                    type: 'authenticate',
                    payload: {
                        userType: 'driver',
                        userId: 1
                    }
                });
                
                websocket.send(authMessage);
                logMessage('Sent authentication message');
            };
            
            websocket.onclose = (event) => {
                logMessage(`WebSocket connection closed: code=${event.code}`, true);
                document.getElementById('connectionStatus').classList.remove('connected');
            };
            
            websocket.onerror = () => {
                logMessage('WebSocket error occurred', true);
            };
            
            websocket.onmessage = (event) => {
                logMessage(`WebSocket message received: ${event.data}`);
                try {
                    const data = JSON.parse(event.data);
                    this.handleWebSocketMessage(data);
                } catch (error) {
                    logMessage(`Error parsing WebSocket message: ${error.message}`, true);
                }
            };
            
            this.websocket = websocket;
        } catch (error) {
            const debugOutput = document.getElementById('debugOutput');
            if (debugOutput) {
                debugOutput.innerHTML += `Error setting up WebSocket: ${error.message}<br>`;
                debugOutput.style.color = 'red';
            }
            console.error('Error setting up WebSocket:', error);
        }
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

    handleWebSocketMessage(data) {
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
    }
}

// Initialize the dashboard when the page loads
document.addEventListener('DOMContentLoaded', function() {
    // Update connection status
    document.getElementById('connectionStatus').classList.add('connected');
    
    // Set driver name
    document.getElementById('driverName').textContent = 'Driver';
    
    // Initialize map directly to ensure it displays
    initializeMap();
    
    // Load other dashboard functionality
    loadDashboard();
});

function initializeMap() {
    try {
        console.log('Initializing HERE map directly in driver-dashboard.js');
        
        // Initialize the Platform
        const platform = new H.service.Platform({
            apikey: 'zx-6o_i0Sv59n9kKgKKmHGvNpbzERdnZ0ZxkI6KEyug'
        });
        
        const defaultLayers = platform.createDefaultLayers();
        
        // Initialize map with specified dimensions
        const mapElement = document.getElementById('mapContainer');
        
        if (!mapElement) {
            console.error('Map container element not found');
            return;
        }
        
        console.log('Map container dimensions:', mapElement.offsetWidth, mapElement.offsetHeight);
        
        const map = new H.Map(
            mapElement,
            defaultLayers.vector.normal.map,
            {
                zoom: 13,
                center: { lat: -1.2921, lng: 36.8219 }, // Default to Nairobi
                pixelRatio: window.devicePixelRatio || 1
            }
        );
        
        // Make map responsive
        window.addEventListener('resize', () => map.getViewPort().resize());
        
        // Add map interaction and controls
        const behavior = new H.mapevents.Behavior(new H.mapevents.MapEvents(map));
        const ui = H.ui.UI.createDefault(map, defaultLayers);
        
        // Create a marker icon
        const svgMarkup = `
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="40" height="40">
            <rect x="2" y="6" width="20" height="12" rx="2" fill="#222222"/>
            <rect x="4" y="8" width="16" height="6" fill="#ffffff" opacity="0.3"/>
            <circle cx="7" cy="18" r="2" fill="#333333"/>
            <circle cx="17" cy="18" r="2" fill="#333333"/>
            <rect x="19" y="9" width="2" height="4" fill="#ffffff"/>
            <rect x="3" y="9" width="2" height="4" fill="#ffffff"/>
        </svg>`;
        
        const icon = new H.map.Icon('data:image/svg+xml;charset=UTF-8,' + encodeURIComponent(svgMarkup), {
            size: { w: 40, h: 40 }
        });
        
        // Add a marker for the driver's position
        const marker = new H.map.Marker(
            { lat: -1.2921, lng: 36.8219 },
            { icon: icon }
        );
        
        map.addObject(marker);
        
        console.log('Map initialized successfully');
        
        // Expose map to window for location-tracker.js to use
        window.dashboardMap = map;
        window.dashboardMapMarker = marker;
        
        // Try to center map on current location
        if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    const { latitude, longitude } = position.coords;
                    map.setCenter({ lat: latitude, lng: longitude });
                    marker.setGeometry({ lat: latitude, lng: longitude });
                    console.log('Map centered on current location');
                },
                (error) => {
                    console.warn('Geolocation error:', error);
                }
            );
        }
        
        // Enable autotracking
        const locationTracker = new LocationTracker();
        locationTracker.fetchApiKey(); // This will use our exposed map
        
        // Update the connection status display
        document.getElementById('connectionStatus').classList.add('map-ready');
        
    } catch (error) {
        console.error('Error initializing map:', error);
        document.getElementById('mapError').textContent = 'Error initializing map: ' + error.message;
        document.getElementById('mapError').style.display = 'block';
    }
}

function loadDashboard() {
    // Additional dashboard functionality will be loaded here
    console.log("Loading dashboard data...");
    
    // Example: Mock data for demonstration
    setTimeout(() => {
        document.getElementById('studentCount').textContent = '5';
    }, 1000);
    
    const driverDashboard = new DriverDashboard();
}
