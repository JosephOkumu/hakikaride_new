class ParentDashboard {
    constructor() {
        this.map = null;
        this.busMarker = null;
        this.platform = null;
        this.websocket = null;
        this.children = [];
        this.notifications = [];
        this.lastKnownBusLocation = null;
        
        this.initializeHereMaps();
        this.initializeWebSocket();
        this.loadParentInfo();
        this.loadChildren();
        this.setupEventListeners();
    }

    initializeHereMaps() {
        // Initialize HERE Maps platform
        this.platform = new H.service.Platform({
            'apikey': process.env.HERE_API_KEY
        });

        const defaultLayers = this.platform.createDefaultLayers();
        
        // Initialize map
        this.map = new H.Map(
            document.getElementById('mapContainer'),
            defaultLayers.vector.normal.map,
            {
                zoom: 13,
                center: { lat: 0, lng: 0 }
            }
        );

        // Enable map interaction and UI
        const behavior = new H.mapevents.Behavior(new H.mapevents.MapEvents(this.map));
        const ui = H.ui.UI.createDefault(this.map, defaultLayers);

        // Make map responsive
        window.addEventListener('resize', () => this.map.getViewPort().resize());
    }

    initializeWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.websocket = new WebSocket(`${protocol}//${window.location.host}/ws/parent`);

        this.websocket.onopen = () => {
            console.log('WebSocket connection established');
            this.updateConnectionStatus(true);
        };

        this.websocket.onclose = () => {
            console.log('WebSocket connection closed');
            this.updateConnectionStatus(false);
            // Attempt to reconnect after 5 seconds
            setTimeout(() => this.initializeWebSocket(), 5000);
        };

        this.websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };
    }

    async loadParentInfo() {
        try {
            const response = await fetch('/api/parent/info');
            const data = await response.json();
            
            if (data.success) {
                document.getElementById('parentName').textContent = 
                    `${data.firstName} ${data.lastName}`;
            }
        } catch (error) {
            console.error('Error loading parent info:', error);
        }
    }

    async loadChildren() {
        try {
            const response = await fetch('/api/parent/children');
            const data = await response.json();
            
            if (data.success) {
                this.children = data.children;
                this.renderChildrenList();
            }
        } catch (error) {
            console.error('Error loading children:', error);
        }
    }

    renderChildrenList() {
        const container = document.getElementById('childrenList');
        container.innerHTML = '';

        this.children.forEach(child => {
            const childCard = document.createElement('div');
            childCard.className = 'child-card';
            childCard.innerHTML = `
                <div class="child-info">
                    <h4>${child.firstName} ${child.lastName}</h4>
                    <p>Grade: ${child.grade}</p>
                    <p>Status: <span class="status-${child.status.toLowerCase()}">${child.status}</span></p>
                </div>
                <div class="child-actions">
                    <button class="btn btn-small" onclick="parentDashboard.showChildDetails(${child.studentId})">
                        Details
                    </button>
                </div>
            `;
            container.appendChild(childCard);
        });
    }

    updateBusLocation(location) {
        this.lastKnownBusLocation = location;
        
        // Update bus marker on map
        const coords = { lat: location.latitude, lng: location.longitude };
        
        if (!this.busMarker) {
            const icon = new H.map.Icon('/static/images/bus-icon.png', { size: { w: 32, h: 32 } });
            this.busMarker = new H.map.Marker(coords, { icon: icon });
            this.map.addObject(this.busMarker);
        } else {
            this.busMarker.setGeometry(coords);
        }

        // Center map on bus location
        this.map.setCenter(coords);

        // Update stats
        document.getElementById('busLocation').textContent = 
            `${location.latitude.toFixed(6)}, ${location.longitude.toFixed(6)}`;
        document.getElementById('currentSpeed').textContent = 
            `${location.speed.toFixed(1)} km/h`;

        // Calculate and update ETA
        this.updateEstimatedArrival(coords);
    }

    async updateEstimatedArrival(busLocation) {
        // Get the first child's drop-off point (assuming all children have same drop-off)
        if (this.children.length === 0) return;

        const child = this.children[0];
        const dropoffPoint = child.dropoffPoint;

        try {
            // Calculate route using HERE Maps Routing API
            const router = this.platform.getRoutingService();
            
            const routingParameters = {
                'mode': 'fastest;car',
                'waypoint0': `${busLocation.lat},${busLocation.lng}`,
                'waypoint1': dropoffPoint,
                'representation': 'display'
            };

            router.calculateRoute(routingParameters, (result) => {
                const route = result.response.route[0];
                const eta = new Date();
                eta.setSeconds(eta.getSeconds() + route.summary.trafficTime);

                document.getElementById('estimatedArrival').textContent = 
                    eta.toLocaleTimeString();
            }, (error) => {
                console.error('Error calculating ETA:', error);
            });
        } catch (error) {
            console.error('Error updating ETA:', error);
        }
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'location_update':
                this.updateBusLocation(data.data);
                break;
            case 'student_status':
                this.updateChildStatus(data.data);
                break;
            case 'notification':
                this.addNotification(data.data);
                break;
        }
    }

    updateChildStatus(statusData) {
        const child = this.children.find(c => c.studentId === statusData.studentId);
        if (child) {
            child.status = statusData.status;
            this.renderChildrenList();
            this.addNotification({
                type: 'status',
                message: `${child.firstName}'s status updated to ${statusData.status}`,
                timestamp: new Date().toISOString()
            });
        }
    }

    addNotification(notification) {
        this.notifications.unshift(notification);
        this.updateNotificationBadge();
        this.renderNotifications();
    }

    updateNotificationBadge() {
        const unreadCount = this.notifications.filter(n => !n.read).length;
        const badge = document.getElementById('notificationCount');
        badge.textContent = unreadCount;
        badge.style.display = unreadCount > 0 ? 'block' : 'none';
    }

    renderNotifications() {
        const container = document.getElementById('notificationsList');
        container.innerHTML = '';

        this.notifications.forEach(notification => {
            const notificationEl = document.createElement('div');
            notificationEl.className = `notification ${notification.read ? 'read' : 'unread'}`;
            notificationEl.innerHTML = `
                <div class="notification-content">
                    <p>${notification.message}</p>
                    <small>${new Date(notification.timestamp).toLocaleTimeString()}</small>
                </div>
            `;
            container.appendChild(notificationEl);
        });
    }

    showChildDetails(studentId) {
        const child = this.children.find(c => c.studentId === studentId);
        if (child) {
            // TODO: Implement child details modal
            console.log('Show child details:', child);
        }
    }

    setupEventListeners() {
        // Notification panel toggle
        document.getElementById('notificationToggle').addEventListener('click', () => {
            const panel = document.getElementById('notificationsPanel');
            panel.classList.toggle('show');
            
            // Mark notifications as read
            if (panel.classList.contains('show')) {
                this.notifications.forEach(n => n.read = true);
                this.updateNotificationBadge();
                this.renderNotifications();
            }
        });

        // Contact driver button
        document.querySelectorAll('.contact-driver').forEach(button => {
            button.addEventListener('click', () => {
                document.getElementById('contactModal').style.display = 'block';
            });
        });
    }

    updateConnectionStatus(isConnected) {
        const status = document.getElementById('busStatus');
        status.textContent = isConnected ? 'Connected - Receiving Updates' : 'Disconnected - Trying to reconnect...';
        status.className = isConnected ? 'status-active' : 'status-inactive';
    }
}

// Initialize the dashboard when the page loads
const parentDashboard = new ParentDashboard();
