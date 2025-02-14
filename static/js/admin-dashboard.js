class AdminDashboard {
    constructor() {
        this.fleetMap = null;
        this.busMarkers = new Map();
        this.selectedBus = null;
        this.platform = null;
        this.websocket = null;
        this.activePanel = 'dashboard';
        
        this.initializeHereMaps();
        this.initializeWebSocket();
        this.setupEventListeners();
        this.loadDashboardData();
    }

    initializeHereMaps() {
        this.platform = new H.service.Platform({
            'apikey': process.env.HERE_API_KEY
        });

        const defaultLayers = this.platform.createDefaultLayers();
        
        // Initialize fleet overview map
        this.fleetMap = new H.Map(
            document.getElementById('fleetMap'),
            defaultLayers.vector.normal.map,
            {
                zoom: 12,
                center: { lat: 0, lng: 0 }
            }
        );

        // Enable map interaction
        const behavior = new H.mapevents.Behavior(new H.mapevents.MapEvents(this.fleetMap));
        const ui = H.ui.UI.createDefault(this.fleetMap, defaultLayers);

        // Make map responsive
        window.addEventListener('resize', () => this.fleetMap.getViewPort().resize());
    }

    initializeWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.websocket = new WebSocket(`${protocol}//${window.location.host}/ws/admin`);

        this.websocket.onopen = () => {
            console.log('Admin WebSocket connection established');
            this.loadActiveFleet();
        };

        this.websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };

        this.websocket.onclose = () => {
            console.log('WebSocket connection closed');
            setTimeout(() => this.initializeWebSocket(), 5000);
        };
    }

    async loadDashboardData() {
        try {
            const response = await fetch('/api/admin/dashboard-stats');
            const data = await response.json();
            
            if (data.success) {
                document.getElementById('activeBusCount').textContent = data.activeBuses;
                document.getElementById('totalStudents').textContent = data.totalStudents;
                document.getElementById('activeRoutes').textContent = data.activeRoutes;
                document.getElementById('issuesCount').textContent = data.issuesCount;
            }
        } catch (error) {
            console.error('Error loading dashboard data:', error);
        }
    }

    async loadActiveFleet() {
        try {
            const response = await fetch('/api/admin/active-fleet');
            const data = await response.json();
            
            if (data.success) {
                this.updateFleetMarkers(data.fleet);
                this.renderActiveTrips(data.fleet);
            }
        } catch (error) {
            console.error('Error loading active fleet:', error);
        }
    }

    updateFleetMarkers(fleet) {
        // Clear existing markers
        this.busMarkers.forEach(marker => this.fleetMap.removeObject(marker));
        this.busMarkers.clear();

        fleet.forEach(bus => {
            const coords = { lat: bus.latitude, lng: bus.longitude };
            const icon = new H.map.Icon('/static/images/bus-icon.png', { size: { w: 32, h: 32 } });
            const marker = new H.map.Marker(coords, { icon: icon });
            
            // Add click event to marker
            marker.addEventListener('tap', () => this.showBusDetails(bus));
            
            this.fleetMap.addObject(marker);
            this.busMarkers.set(bus.busId, marker);
        });

        // Adjust map view to show all buses
        if (fleet.length > 0) {
            const group = new H.map.Group(Array.from(this.busMarkers.values()));
            this.fleetMap.getViewModel().setLookAtData({
                bounds: group.getBoundingBox()
            });
        }
    }

    renderActiveTrips(fleet) {
        const container = document.getElementById('activeTripsGrid');
        container.innerHTML = '';

        fleet.forEach(bus => {
            const tripCard = document.createElement('div');
            tripCard.className = 'trip-card';
            tripCard.innerHTML = `
                <div class="trip-info">
                    <h4>Bus ${bus.numberPlate}</h4>
                    <p>Route: ${bus.routeName}</p>
                    <p>Driver: ${bus.driverName}</p>
                    <p>Students: ${bus.studentsOnBoard}/${bus.totalStudents}</p>
                </div>
                <div class="trip-actions">
                    <button class="btn btn-small" onclick="adminDashboard.showBusDetails(${bus.busId})">
                        View Details
                    </button>
                </div>
            `;
            container.appendChild(tripCard);
        });
    }

    async showBusDetails(busId) {
        try {
            const response = await fetch(`/api/admin/bus/${busId}`);
            const data = await response.json();
            
            if (data.success) {
                document.getElementById('busNumberPlate').textContent = data.numberPlate;
                document.getElementById('busDriver').textContent = data.driverName;
                document.getElementById('busRoute').textContent = data.routeName;
                document.getElementById('busStatus').textContent = data.status;

                this.renderBusStudentsList(data.students);
                this.showModal('busDetailsModal');
            }
        } catch (error) {
            console.error('Error loading bus details:', error);
        }
    }

    renderBusStudentsList(students) {
        const container = document.getElementById('busStudentsList');
        container.innerHTML = '';

        students.forEach(student => {
            const studentItem = document.createElement('div');
            studentItem.className = 'student-item';
            studentItem.innerHTML = `
                <span>${student.firstName} ${student.lastName}</span>
                <span class="status-${student.status.toLowerCase()}">${student.status}</span>
            `;
            container.appendChild(studentItem);
        });
    }

    async addNewStudent(formData) {
        try {
            const response = await fetch('/api/admin/students', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(Object.fromEntries(formData))
            });

            const data = await response.json();
            if (data.success) {
                this.closeModal('addStudentModal');
                this.loadStudents();
                alert('Student added successfully!');
            }
        } catch (error) {
            console.error('Error adding student:', error);
            alert('Failed to add student. Please try again.');
        }
    }

    async loadDrivers() {
        try {
            const response = await fetch('/api/admin/drivers');
            const data = await response.json();
            
            if (data.success) {
                this.renderDriversTable(data.drivers);
            }
        } catch (error) {
            console.error('Error loading drivers:', error);
        }
    }

    async loadRoutes() {
        try {
            const response = await fetch('/api/admin/routes');
            const data = await response.json();
            
            if (data.success) {
                this.renderRoutesTable(data.routes);
            }
        } catch (error) {
            console.error('Error loading routes:', error);
        }
    }

    async loadBuses() {
        try {
            const response = await fetch('/api/admin/buses');
            const data = await response.json();
            
            if (data.success) {
                this.renderBusesTable(data.buses);
            }
        } catch (error) {
            console.error('Error loading buses:', error);
        }
    }

    renderDriversTable(drivers) {
        const container = document.getElementById('driversTable');
        if (!container) return;

        container.innerHTML = `
            <table class="data-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Phone</th>
                        <th>Active Trips</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${drivers.map(driver => `
                        <tr>
                            <td>${driver.name}</td>
                            <td>${driver.phone}</td>
                            <td>${driver.activeTrips}</td>
                            <td>
                                <button class="btn btn-small" onclick="adminDashboard.viewDriverDetails(${driver.id})">
                                    View Details
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    renderRoutesTable(routes) {
        const container = document.getElementById('routesTable');
        if (!container) return;

        container.innerHTML = `
            <table class="data-table">
                <thead>
                    <tr>
                        <th>Route Name</th>
                        <th>Description</th>
                        <th>Assigned Buses</th>
                        <th>Active Trips</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${routes.map(route => `
                        <tr>
                            <td>${route.name}</td>
                            <td>${route.description}</td>
                            <td>${route.buses}</td>
                            <td>${route.activeTrips}</td>
                            <td>
                                <button class="btn btn-small" onclick="adminDashboard.viewRouteDetails(${route.id})">
                                    View Details
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    renderBusesTable(buses) {
        const container = document.getElementById('busesTable');
        if (!container) return;

        container.innerHTML = `
            <table class="data-table">
                <thead>
                    <tr>
                        <th>Number Plate</th>
                        <th>Route</th>
                        <th>Status</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${buses.map(bus => `
                        <tr>
                            <td>${bus.plate}</td>
                            <td>${bus.route || 'Not Assigned'}</td>
                            <td>
                                <span class="status-${bus.status.toLowerCase().replace(' ', '-')}">
                                    ${bus.status}
                                </span>
                            </td>
                            <td>
                                <button class="btn btn-small" onclick="adminDashboard.viewBusDetails(${bus.id})">
                                    View Details
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    async loadStudents() {
        try {
            const response = await fetch('/api/admin/students');
            const data = await response.json();
            
            if (data.success) {
                this.renderStudentsTable(data.students);
            }
        } catch (error) {
            console.error('Error loading students:', error);
        }
    }

    renderStudentsTable(students) {
        const tbody = document.getElementById('studentsTableBody');
        tbody.innerHTML = '';

        students.forEach(student => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${student.admNumber}</td>
                <td>${student.firstName} ${student.lastName}</td>
                <td>${student.grade}</td>
                <td>${student.routeName}</td>
                <td>${student.parentPhone}</td>
                <td class="status-${student.status.toLowerCase()}">${student.status}</td>
                <td>
                    <button class="btn btn-small" onclick="adminDashboard.editStudent(${student.studentId})">
                        Edit
                    </button>
                    <button class="btn btn-small btn-danger" onclick="adminDashboard.removeStudent(${student.studentId})">
                        Remove
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'location_update':
                this.updateBusLocation(data.data);
                break;
            case 'student_status':
                this.updateStudentStatus(data.data);
                break;
            case 'emergency':
                this.handleEmergency(data.data);
                break;
        }
    }

    updateBusLocation(data) {
        const marker = this.busMarkers.get(data.busId);
        if (marker) {
            marker.setGeometry({ lat: data.latitude, lng: data.longitude });
        }
    }

    handleEmergency(data) {
        const alertCount = parseInt(document.querySelector('.alert-badge').textContent);
        document.querySelector('.alert-badge').textContent = alertCount + 1;
        
        // Show emergency notification
        this.showEmergencyAlert(data);
    }

    showEmergencyAlert(data) {
        // Implementation for emergency alerts
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-links li').forEach(link => {
            link.addEventListener('click', () => this.switchPanel(link.dataset.panel));
        });

        // Add Student Form
        document.getElementById('addStudentForm').addEventListener('submit', (e) => {
            e.preventDefault();
            this.addNewStudent(new FormData(e.target));
        });

        // Refresh tracking
        document.getElementById('refreshTracking').addEventListener('click', () => {
            this.loadActiveFleet();
        });

        // Add new entities buttons
        document.getElementById('addStudent').addEventListener('click', () => {
            this.showModal('addStudentModal');
        });
    }

    switchPanel(panelId) {
        document.querySelector(`.nav-links li.active`).classList.remove('active');
        document.querySelector(`.nav-links li[data-panel="${panelId}"]`).classList.add('active');

        document.querySelector('.panel.active').classList.remove('active');
        document.getElementById(`${panelId}Panel`).classList.add('active');

        this.activePanel = panelId;
    }

    showModal(modalId) {
        document.getElementById(modalId).style.display = 'block';
    }

    closeModal(modalId) {
        document.getElementById(modalId).style.display = 'none';
    }

    updateDashboardStats(data) {
        document.getElementById('activeBusCount').textContent = data.activeBuses;
        document.getElementById('totalStudents').textContent = data.totalStudents;
        document.getElementById('activeRoutes').textContent = data.activeRoutes;
        document.getElementById('issuesCount').textContent = data.issuesCount;
    }
}

// Initialize the dashboard when the page loads
const adminDashboard = new AdminDashboard();
