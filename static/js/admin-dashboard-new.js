// Initialize map
let map;
let busMarkers = {};

function initMap() {
    map = L.map('map').setView([0, 0], 13);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors'
    }).addTo(map);
}

// Populate bus fleet table
function populateBusFleet(buses) {
    const tbody = document.querySelector('.fleet-table tbody');
    tbody.innerHTML = '';

    buses.forEach(bus => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${bus.id}</td>
            <td>${bus.numberPlate}</td>
            <td>${bus.route}</td>
            <td>${bus.driver}</td>
            <td><span class="status ${bus.status.toLowerCase()}">${bus.status}</span></td>
        `;
        tr.addEventListener('click', () => showBusLocation(bus));
        tbody.appendChild(tr);
    });
}

// Show bus location on map
function showBusLocation(bus) {
    const mapSection = document.getElementById('mapSection');
    mapSection.style.display = 'block';

    if (!map) {
        initMap();
    }

    // Update map view and marker
    map.setView([bus.location.lat, bus.location.lng], 15);
    
    if (busMarkers[bus.id]) {
        busMarkers[bus.id].setLatLng([bus.location.lat, bus.location.lng]);
    } else {
        const busIcon = L.divIcon({
            html: '<i class="fas fa-bus"></i>',
            className: 'bus-marker',
            iconSize: [30, 30]
        });

        busMarkers[bus.id] = L.marker([bus.location.lat, bus.location.lng], {
            icon: busIcon
        }).addTo(map);
    }
}

// Handle navigation
document.querySelectorAll('.nav-links li').forEach(item => {
    item.addEventListener('click', () => {
        const panel = item.getAttribute('data-panel');
        document.querySelectorAll('.nav-links li').forEach(i => i.classList.remove('active'));
        item.classList.add('active');

        // Handle panel switching
        switch(panel) {
            case 'fleet':
                // Show fleet management view
                break;
            case 'students':
                // Show students view
                break;
            case 'drivers':
                // Show drivers view
                break;
            case 'sign-out':
                // Handle sign out
                window.location.href = '/logout';
                break;
        }
    });
});

// Search functionality
const searchInput = document.querySelector('.search-input');
searchInput.addEventListener('input', (e) => {
    const searchTerm = e.target.value.toLowerCase();
    // Implement search logic here
});

// Example data for testing
const sampleBuses = [
    {
        id: 'BUS001',
        numberPlate: 'XYZ-1234',
        route: 'Route A - North Side',
        driver: 'John Smith',
        status: 'Active',
        location: { lat: -1.2921, lng: 36.8219 } // Example coordinates for Nairobi
    },
    {
        id: 'BUS002',
        numberPlate: 'ABC-5678',
        route: 'Route B - South Side',
        driver: 'Mary Johnson',
        status: 'Active',
        location: { lat: -1.2975, lng: 36.8123 }
    },
    {
        id: 'BUS003',
        numberPlate: 'DEF-9012',
        route: 'Route C - East Side',
        driver: 'Robert Wilson',
        status: 'Maintenance',
        location: { lat: -1.2876, lng: 36.8314 }
    }
];

// Initialize the dashboard
document.addEventListener('DOMContentLoaded', () => {
    populateBusFleet(sampleBuses);
});
