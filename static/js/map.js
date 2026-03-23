// Initialize the map centered on Oslo
const map = L.map('map').setView([59.9139, 10.7522], 13);

// Add OpenStreetMap tile layer
L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
}).addTo(map);

// Fetch house data from the API
fetch('/api/houses')
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(houses => {
        // Iterate over the data and add markers
        houses.forEach(house => {
            const popupContent = `
                <b>${house.address}</b><br>
                ${house.description}
            `;
            
            L.marker([house.lat, house.lng])
                .addTo(map)
                .bindPopup(popupContent);
        });
    })
    .catch(error => {
        console.error('There was a problem fetching the house data:', error);
    });
