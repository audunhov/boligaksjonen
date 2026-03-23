// Initialize the map centered on Oslo
const map = L.map('map').setView([59.9139, 10.7522], 13);

// Add OpenStreetMap tile layer
L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
}).addTo(map);

// Create a marker cluster group
const markers = L.markerClusterGroup();

// Fetch house data from the API
fetch('/api/houses')
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(houses => {
        // Iterate over the data and add markers to the cluster group
        houses.forEach(house => {
            const popupContent = `
                <b>${house.address}</b><br>
                <span class="badge badge-${house.ownership_type}">${house.ownership_type}</span><br>
                ${house.description}<br>
                <div style="margin-top: 10px; font-size: 0.9em; color: #666;">
                    Last updated by: ${house.last_updated_by}<br>
                    Date: ${new Date(house.updated_at).toLocaleString()}
                </div>
                <hr>
                <a href="/houses/edit?id=${house.id}">Edit this entry</a>
            `;
            
            const marker = L.marker([house.lat, house.lng])
                .bindPopup(popupContent);
            
            markers.addLayer(marker);
        });

        // Add the cluster group to the map
        map.addLayer(markers);
    })
    .catch(error => {
        console.error('There was a problem fetching the house data:', error);
    });
