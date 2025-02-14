class LocationTracker {
    constructor() {
        this.watchId = null;
        this.isTracking = false;
        this.websocket = null;
        this.map = null;
        this.marker = null;
        this.platform = new H.service.Platform({
            'apikey': process.env.HERE_API_KEY
        });
    }

    initializeMap() {
        // Initialize HERE Map
        const defaultLayers = this.platform.createDefaultLayers();
        this.map = new H.Map(
            document.getElementById('mapContainer'),
            defaultLayers.vector.normal.map,
            {
                zoom: 15,
                center: { lat: 0, lng: 0 }
            }
        );

        // Enable map interaction
        const behavior = new H.mapevents.Behavior(new H.mapevents.MapEvents(this.map));
        const ui = H.ui.UI.createDefault(this.map, defaultLayers);

        // Make map responsive
        window.addEventListener('resize', () => this.map.getViewPort().resize());
    }

    initializeWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        this.websocket = new WebSocket(`${protocol}//${window.location.host}/ws/location`);
        
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

        this.websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus(false);
        };
    }

    updateConnectionStatus(isConnected) {
        const statusIndicator = document.getElementById('connectionStatus');
        statusIndicator.className = 'status-indicator ' + (isConnected ? 'status-active' : 'status-inactive');
    }

    startTracking() {
        if (!navigator.geolocation) {
            alert('Geolocation is not supported by your browser');
            return;
        }

        this.isTracking = true;
        document.getElementById('tripStatus').textContent = 'In Progress';
        document.getElementById('startTrip').style.display = 'none';
        document.getElementById('endTrip').style.display = 'block';

        // Start watching position
        this.watchId = navigator.geolocation.watchPosition(
            (position) => this.handlePositionUpdate(position),
            (error) => this.handlePositionError(error),
            {
                enableHighAccuracy: true,
                maximumAge: 0,
                timeout: 5000
            }
        );
    }

    handlePositionUpdate(position) {
        const { latitude, longitude, speed } = position.coords;
        const location = { lat: latitude, lng: longitude };

        // Update map marker
        if (!this.marker) {
            this.marker = new H.map.Marker(location);
            this.map.addObject(this.marker);
        } else {
            this.marker.setGeometry(location);
        }

        // Center map on current position
        this.map.setCenter(location);

        // Update speed display
        const speedKmh = speed ? (speed * 3.6).toFixed(1) : 0;
        document.getElementById('currentSpeed').textContent = `${speedKmh} km/h`;

        // Send location update to server
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({
                type: 'location_update',
                data: {
                    latitude,
                    longitude,
                    speed: speedKmh,
                    timestamp: new Date().toISOString()
                }
            }));
        }
    }

    handlePositionError(error) {
        console.error('Error getting location:', error);
        alert(`Location error: ${error.message}`);
    }

    stopTracking() {
        if (this.watchId) {
            navigator.geolocation.clearWatch(this.watchId);
            this.watchId = null;
        }
        this.isTracking = false;
        document.getElementById('tripStatus').textContent = 'Completed';
        document.getElementById('startTrip').style.display = 'block';
        document.getElementById('endTrip').style.display = 'none';

        // Send trip end notification
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({
                type: 'trip_end',
                data: {
                    timestamp: new Date().toISOString()
                }
            }));
        }
    }

    reportEmergency() {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({
                type: 'emergency',
                data: {
                    timestamp: new Date().toISOString(),
                    location: this.marker ? {
                        latitude: this.marker.getGeometry().lat,
                        longitude: this.marker.getGeometry().lng
                    } : null
                }
            }));
        }
        alert('Emergency reported! Support team has been notified.');
    }
}

// Initialize location tracker when the page loads
document.addEventListener('DOMContentLoaded', () => {
    const tracker = new LocationTracker();
    tracker.initializeMap();
    tracker.initializeWebSocket();

    // Set up event listeners
    document.getElementById('startTrip').addEventListener('click', () => tracker.startTracking());
    document.getElementById('endTrip').addEventListener('click', () => tracker.stopTracking());
    document.getElementById('emergencyButton').addEventListener('click', () => tracker.reportEmergency());
});
