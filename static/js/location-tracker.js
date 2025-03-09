class LocationTracker {
    constructor(debug = false) {
        // Initialize variables
        this.watchId = null;
        this.isTracking = false;
        this.websocket = null;
        this.map = null;
        this.marker = null;
        this.tripId = null;
        
        // Use a hardcoded API key instead of environment variable
        // This fixed the Firefox process.env reference error
        this.apiKey = 'zx-6o_i0Sv59n9kKgKKmHGvNpbzERdnZ0ZxkI6KEyug';
        
        this.polyline = null;
        this.positions = [];
        
        // Add debugging function
        this.debug('LocationTracker initialized');
        
        // Create a bus icon for the marker
        this.busIcon = this.createBusIcon();
    }
    
    createBusIcon() {
        // Bus icon SVG definition
        const svgMarkup = `
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="40" height="40">
            <rect x="2" y="6" width="20" height="12" rx="2" fill="#222222"/>
            <rect x="4" y="8" width="16" height="6" fill="#ffffff" opacity="0.3"/>
            <circle cx="7" cy="18" r="2" fill="#333333"/>
            <circle cx="17" cy="18" r="2" fill="#333333"/>
            <rect x="19" y="9" width="2" height="4" fill="#ffffff"/>
            <rect x="3" y="9" width="2" height="4" fill="#ffffff"/>
        </svg>
        `;
        
        // Create the icon
        return new H.map.Icon('data:image/svg+xml;charset=UTF-8,' + encodeURIComponent(svgMarkup), {
            size: { w: 40, h: 40 }
        });
    }
    
    // Debug helper function
    debug(message, isError = false) {
        const debugOutput = document.getElementById('debugOutput');
        if (debugOutput) {
            const newLine = document.createTextNode(message + '\n');
            debugOutput.appendChild(newLine);
            debugOutput.scrollTop = debugOutput.scrollHeight;
        }
        
        if (isError) {
            console.error(message);
        } else {
            console.log(message);
        }
    }

    async fetchApiKey() {
        this.debug('Fetching HERE Maps API key...');
        
        // Show map loading status
        const loadingStatus = document.getElementById('mapLoadingStatus');
        if (loadingStatus) {
            loadingStatus.style.display = 'block';
            loadingStatus.textContent = 'Loading map...';
        }
        
        try {
            // Check if we already have a map instance created by driver-dashboard.js
            if (window.dashboardMap) {
                this.debug('Using existing map instance from dashboard');
                this.map = window.dashboardMap;
                this.marker = window.dashboardMapMarker;
                
                // Add polyline for tracking
                this.initializePolyline();
                
                if (loadingStatus) {
                    loadingStatus.style.display = 'none';
                }
                return;
            }
            
            const response = await fetch('/api/config/here-api-key');
            const data = await response.json();
            if (data.success) {
                this.apiKey = data.apiKey;
                this.debug(`API key successfully fetched: ${this.apiKey.substring(0, 5)}...`);
                this.initializeMap();
            } else {
                const errorMsg = 'Failed to fetch API key: ' + (data.message || 'Unknown error');
                this.debug(errorMsg, true);
                if (loadingStatus) {
                    loadingStatus.textContent = 'Error: Failed to load API key';
                }
                // Try to initialize map with fallback key
                this.initializeMap();
            }
        } catch (error) {
            this.debug('Error fetching API key: ' + error.message, true);
            if (loadingStatus) {
                loadingStatus.textContent = 'Error: ' + error.message;
            }
            // Fallback to initialize map without API key for development purposes
            this.initializeMap();
        }
    }

    initializePolyline() {
        try {
            this.debug('Initializing polyline for tracking path');
            
            if (!this.map) {
                this.debug('Map not initialized yet, cannot create polyline', true);
                return;
            }
            
            // Get a valid starting position (Kampala coordinates as fallback)
            const initialLat = -1.286389;
            const initialLng = 36.817223;
            
            // Create a valid linestring with at least one point
            try {
                // Create a properly formatted LineString
                const lineString = new H.geo.LineString();
                
                // Add the coordinates (must be numeric values)
                lineString.pushPoint({lat: initialLat, lng: initialLng});
                
                this.debug(`Created LineString with point: ${initialLat}, ${initialLng}`);
                
                // Create the polyline
                this.polyline = new H.map.Polyline(
                    lineString, 
                    { 
                        style: {
                            lineWidth: 4,
                            strokeColor: 'rgba(0, 128, 255, 0.7)',
                            lineCap: 'round'
                        }
                    }
                );
                
                // Add the polyline to the map
                this.map.addObject(this.polyline);
                this.debug('Polyline added to map successfully');
            } catch (innerError) {
                this.debug(`LineString error: ${innerError}`, true);
                
                // Try an alternative approach using a simple polyline
                const demoCoords = [
                    {lat: initialLat, lng: initialLng},
                    {lat: initialLat + 0.001, lng: initialLng + 0.001}
                ];
                
                this.debug('Trying alternative polyline approach');
                
                // Create polyline with the array of points
                const strip = new H.geo.LineString();
                strip.pushLatLngAlt(initialLat, initialLng);
                strip.pushLatLngAlt(initialLat + 0.001, initialLng + 0.001);
                
                this.polyline = new H.map.Polyline(strip, {
                    style: { 
                        lineWidth: 4,
                        strokeColor: 'rgba(0, 128, 255, 0.7)' 
                    }
                });
                
                this.map.addObject(this.polyline);
                this.debug('Alternative polyline added successfully');
            }
        } catch (error) {
            this.debug('Error initializing polyline: ' + error, true);
        }
    }

    getCurrentPosition() {
        // Try to get the current position
        if (this.positions.length > 0) {
            return this.positions[this.positions.length - 1];
        }
        
        // If we don't have any positions yet, return a default fallback position
        // This is just for development - in a real app, you'd get the user's actual position
        return { lat: -1.286389, lng: 36.817223 }; // Default location in Nairobi
    }

    initializeMap() {
        this.debug('Initializing HERE map...');
        
        // Get map container
        const mapContainer = document.getElementById('mapContainer');
        const loadingStatus = document.getElementById('mapLoadingStatus');
        
        if (!mapContainer) {
            this.debug('Map container not found!', true);
            if (loadingStatus) {
                loadingStatus.textContent = 'Error: Map container not found';
                loadingStatus.style.display = 'block';
            }
            return;
        }
        
        this.debug(`Map container found, dimensions: ${mapContainer.offsetWidth}x${mapContainer.offsetHeight}`);
        
        try {
            // Check if HERE API is available
            if (typeof H === 'undefined') {
                this.debug('HERE Maps API not loaded!', true);
                if (loadingStatus) {
                    loadingStatus.textContent = 'Error: HERE Maps API not loaded';
                    loadingStatus.style.display = 'block';
                }
                return;
            }
            
            this.debug('HERE Maps API available, creating platform');
            
            // Force the map container to have a specific height for testing
            mapContainer.style.height = '400px';
            
            // Use direct HERE Maps API key instead of process.env
            const apiKey = this.apiKey || 'zx-6o_i0Sv59n9kKgKKmHGvNpbzERdnZ0ZxkI6KEyug';
            
            // Initialize the platform
            this.platform = new H.service.Platform({
                apikey: apiKey
            });
            
            const defaultLayers = this.platform.createDefaultLayers();
            
            this.debug('Creating map instance');
            
            // Create map instance
            this.map = new H.Map(
                mapContainer,
                defaultLayers.vector.normal.map,
                {
                    zoom: 15,
                    pixelRatio: window.devicePixelRatio || 1
                }
            );

            this.debug('Map created, enabling interaction');
            
            // Enable map interaction (pan, zoom)
            const behavior = new H.mapevents.Behavior(new H.mapevents.MapEvents(this.map));
            
            this.debug('Creating UI');
            
            // Create default UI components
            this.ui = H.ui.UI.createDefault(this.map, defaultLayers);
            
            // Make map responsive
            window.addEventListener('resize', () => this.map.getViewPort().resize());
            
            this.debug('Setting initial map center');
            
            // Set initial map center based on device location
            this.setInitialMapCenter();
            
            this.debug('Map initialized successfully!');
            
            // Hide loading status
            if (loadingStatus) {
                loadingStatus.style.display = 'none';
            }
        } catch (error) {
            this.debug('Error initializing map: ' + error.message, true);
            if (loadingStatus) {
                loadingStatus.textContent = 'Error initializing map: ' + error.message;
                loadingStatus.style.display = 'block';
            }
        }
    }

    setInitialMapCenter() {
        if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    const { latitude, longitude } = position.coords;
                    this.map.setCenter({ lat: latitude, lng: longitude });
                    
                    // Add an initial marker for the driver's position
                    this.marker = new H.map.Marker(
                        { lat: latitude, lng: longitude },
                        { icon: this.busIcon }
                    );
                    this.map.addObject(this.marker);
                    
                    console.log('Initial map position set to current location');
                },
                (error) => {
                    console.error('Error getting initial position:', error);
                    // Default to a fallback location (e.g., city center) if geolocation fails
                    this.map.setCenter({ lat: -1.286389, lng: 36.817223 }); // Nairobi coordinates
                },
                { enableHighAccuracy: true, timeout: 5000, maximumAge: 0 }
            );
        } else {
            console.error('Geolocation is not supported by this browser');
            this.map.setCenter({ lat: -1.286389, lng: 36.817223 }); // Nairobi coordinates
        }
    }

    initializeWebSocket() {
        this.debug('Initializing WebSocket connection');
        
        // Basic approach with minimal complexity
        try {
            // Close any existing connection
            if (this.websocket) {
                try {
                    this.websocket.close();
                } catch (e) {
                    // Ignore errors when closing
                }
                this.websocket = null;
            }
            
            // Create WebSocket with simple protocol determination
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            const wsUrl = `${protocol}//${host}/ws`;
            
            this.debug(`Connecting to WebSocket at: ${wsUrl}`);
            
            // Create a simple WebSocket without fancy features
            this.websocket = new WebSocket(wsUrl);
            
            // Set up event handlers
            this.websocket.onopen = () => {
                this.debug('WebSocket connection opened');
                this.updateConnectionStatus(true);
                
                // Send a simple authentication message
                this.websocket.send(JSON.stringify({
                    type: 'authenticate',
                    payload: {
                        userType: 'driver',
                        userId: 1
                    }
                }));
            };
            
            this.websocket.onclose = (event) => {
                this.debug(`WebSocket closed: code=${event.code}`, true);
                this.updateConnectionStatus(false);
            };
            
            this.websocket.onerror = (error) => {
                this.debug('WebSocket error occurred', true);
                this.updateConnectionStatus(false);
            };
            
            this.websocket.onmessage = (event) => {
                this.debug(`Received WebSocket message: ${event.data}`);
            };
            
            return true;
        } catch (error) {
            this.debug(`Failed to initialize WebSocket: ${error.message}`, true);
            this.updateConnectionStatus(false);
            return false;
        }
    }

    updateConnectionStatus(isConnected) {
        const statusIndicator = document.getElementById('connectionStatus');
        if (statusIndicator) {
            statusIndicator.className = 'status-indicator ' + (isConnected ? 'status-active' : 'status-inactive');
        }
    }

    async startTrip() {
        if (!navigator.geolocation) {
            alert('Geolocation is not supported by your browser');
            return;
        }

        try {
            // Start a new trip on the server
            const response = await fetch('/api/trips/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    startTime: new Date().toISOString()
                })
            });

            const data = await response.json();
            if (!data.success) {
                alert(`Failed to start trip: ${data.message || 'Unknown error'}`);
                return;
            }

            // Store the trip ID for location updates
            this.tripId = data.tripId;
            this.isTracking = true;
            this.positions = []; // Reset positions array for the new trip
            
            // Create a polyline to show the travel path
            this.createPolyline();
            
            // Update UI
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
            
            console.log(`Trip started with ID: ${this.tripId}`);
        } catch (error) {
            console.error('Error starting trip:', error);
            alert('Failed to start trip. Please try again.');
        }
    }

    createPolyline() {
        // Create a polyline to show the travel path
        if (this.polyline) {
            this.map.removeObject(this.polyline);
        }
        
        this.polyline = new H.map.Polyline(
            new H.geo.LineString([]),
            {
                style: {
                    lineWidth: 4,
                    strokeColor: 'rgba(0, 128, 255, 0.7)'
                }
            }
        );
        
        this.map.addObject(this.polyline);
    }

    handlePositionUpdate(position) {
        const { latitude, longitude, speed } = position.coords;
        const location = { lat: latitude, lng: longitude };
        
        // Add position to array for the polyline
        this.positions.push(location);
        
        // Update polyline with new position
        this.updatePolyline();

        // Update map marker
        if (!this.marker) {
            this.marker = new H.map.Marker(location, { icon: this.busIcon });
            this.map.addObject(this.marker);
        } else {
            this.marker.setGeometry(location);
        }

        // Center map on current position
        this.map.setCenter(location);

        // Update speed display (convert m/s to km/h)
        const speedKmh = speed ? (speed * 3.6).toFixed(1) : 0;
        document.getElementById('currentSpeed').textContent = `${speedKmh} km/h`;

        // Send location update to server if trip is active
        if (this.tripId && this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({
                type: 'location_update',
                data: {
                    tripId: this.tripId,
                    latitude,
                    longitude,
                    speed: speedKmh,
                    timestamp: new Date().toISOString()
                }
            }));
        } else {
            // If WebSocket isn't ready, try sending via HTTP API
            this.sendLocationUpdateViaApi(latitude, longitude, speedKmh);
        }
    }
    
    updatePolyline() {
        if (this.polyline && this.positions.length > 1) {
            // Create a new LineString with all positions
            const lineString = new H.geo.LineString();
            
            // Add all positions to the LineString
            this.positions.forEach(pos => {
                lineString.pushLatLngAlt(pos.lat, pos.lng, 0);
            });
            
            // Update the polyline geometry
            this.polyline.setGeometry(lineString);
        }
    }

    async sendLocationUpdateViaApi(latitude, longitude, speed) {
        if (!this.tripId) return;
        
        try {
            await fetch('/api/location/update', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    tripId: this.tripId,
                    latitude,
                    longitude,
                    speed
                })
            });
        } catch (error) {
            console.error('Error sending location update via API:', error);
        }
    }

    handlePositionError(error) {
        console.error('Error getting location:', error);
        alert(`Location error: ${error.message}`);
    }

    async endTrip() {
        if (this.watchId) {
            navigator.geolocation.clearWatch(this.watchId);
            this.watchId = null;
        }
        
        if (!this.tripId) {
            alert('No active trip to end');
            return;
        }
        
        try {
            const response = await fetch(`/api/trips/end?tripId=${this.tripId}`, {
                method: 'GET'
            });

            const data = await response.json();
            if (!data.success) {
                alert(`Failed to end trip: ${data.message || 'Unknown error'}`);
                return;
            }
            
            this.isTracking = false;
            this.tripId = null;
            
            // Update UI
            document.getElementById('tripStatus').textContent = 'Completed';
            document.getElementById('startTrip').style.display = 'block';
            document.getElementById('endTrip').style.display = 'none';
            
            // Send trip end notification via WebSocket
            if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
                this.websocket.send(JSON.stringify({
                    type: 'trip_end',
                    data: {
                        timestamp: new Date().toISOString()
                    }
                }));
            }
            
            console.log('Trip ended successfully');
        } catch (error) {
            console.error('Error ending trip:', error);
            alert('Failed to end trip. Please try again.');
        }
    }

    reportEmergency() {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            const location = this.marker ? {
                latitude: this.marker.getGeometry().lat,
                longitude: this.marker.getGeometry().lng
            } : null;
            
            this.websocket.send(JSON.stringify({
                type: 'emergency',
                data: {
                    tripId: this.tripId,
                    timestamp: new Date().toISOString(),
                    location
                }
            }));
            
            alert('Emergency reported! Support team has been notified.');
        } else {
            this.reportEmergencyViaApi();
        }
    }
    
    async reportEmergencyViaApi() {
        try {
            const location = this.marker ? {
                latitude: this.marker.getGeometry().lat,
                longitude: this.marker.getGeometry().lng
            } : null;
            
            const response = await fetch('/api/emergency/report', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    tripId: this.tripId,
                    location,
                    timestamp: new Date().toISOString()
                })
            });
            
            const data = await response.json();
            if (data.success) {
                alert('Emergency reported! Support team has been notified.');
            } else {
                alert('Failed to report emergency. Please try calling emergency services directly.');
            }
        } catch (error) {
            console.error('Error reporting emergency:', error);
            alert('Failed to report emergency. Please try calling emergency services directly.');
        }
    }
}

// Explicitly instantiate the tracker only if we're on a page that needs it
if (document.getElementById('mapContainer') && 
    (document.getElementById('startTrip') || document.getElementById('endTrip'))) {
    console.log('Driver dashboard detected, initializing location tracker');
    document.addEventListener('DOMContentLoaded', () => {
        console.log('DOM loaded, initializing tracker');
        // Add a small delay to ensure the map is initialized in driver-dashboard.js
        setTimeout(() => {
            const tracker = new LocationTracker();
            console.log('Fetching API key and initializing map');
            tracker.fetchApiKey();
            
            // Set up event listeners
            const startTripBtn = document.getElementById('startTrip');
            if (startTripBtn) {
                startTripBtn.addEventListener('click', () => tracker.startTrip());
            }
            
            const endTripBtn = document.getElementById('endTrip');
            if (endTripBtn) {
                endTripBtn.addEventListener('click', () => tracker.endTrip());
            }
            
            const emergencyBtn = document.getElementById('emergencyButton');
            if (emergencyBtn) {
                emergencyBtn.addEventListener('click', () => tracker.reportEmergency());
            }
            
            // Initialize WebSocket connection
            console.log('Initializing WebSocket connection');
            tracker.initializeWebSocket();
        }, 1000); // Wait 1 second for map initialization
    });
}
