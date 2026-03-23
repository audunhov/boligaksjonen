// Initialize the map centered on Oslo
const map = L.map('map').setView([59.9139, 10.7522], 13);

// Add OpenStreetMap tile layer
L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
}).addTo(map);

// Create a marker cluster group
const markers = L.markerClusterGroup();
let allHouses = []; // Store houses globally for modal access

// Function to open the edit modal
window.openEditModal = function(id) {
    const house = allHouses.find(h => h.id === id);
    if (!house) return;

    document.getElementById('edit_id').value = house.id;
    document.getElementById('edit_address').value = house.address;
    document.getElementById('edit_lat').value = house.lat;
    document.getElementById('edit_lng').value = house.lng;
    document.getElementById('edit_description').value = house.description;
    
    const standardTypes = ['kommune', 'fylke', 'stat', 'privatperson', 'selskap'];
    if (standardTypes.includes(house.ownership_type)) {
        document.getElementById('edit_ownership_type').value = house.ownership_type;
        document.getElementById('edit_freetext_group').style.display = 'none';
        document.getElementById('edit_ownership_freetext').value = '';
    } else {
        document.getElementById('edit_ownership_type').value = 'freetext';
        document.getElementById('edit_freetext_group').style.display = 'block';
        document.getElementById('edit_ownership_freetext').value = house.ownership_type;
    }

    document.getElementById('editDialog').showModal();
};

// Fetch house data from the API
fetch('/api/houses')
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(houses => {
        allHouses = houses;
        // Iterate over the data and add markers to the cluster group
        houses.forEach(house => {
            const marker = L.marker([house.latitude, house.longitude]);

            // Handle marker click to show details in sidebar
            marker.on('click', () => {
                showHouseDetails(house);
            });

            // Keep popup as a fallback/quick-view if desired, or remove it
            const formattedDescription = house.description.replace(/\r\n/g, '<br>').replace(/\n/g, '<br>');
            const popupContent = `
                <div class="p-1">
                    <b class="text-gray-900">${house.address}</b><br>
                    <span class="text-[10px] font-bold uppercase text-blue-600">${house.ownership_type.replace('-', ' ')}</span>
                </div>
            `;
            marker.bindPopup(popupContent);

            markers.addLayer(marker);
        });

            
            markers.addLayer(marker);
        });

        // Add the cluster group to the map
        map.addLayer(markers);
    })
    .catch(error => {
        console.error('There was a problem fetching the house data:', error);
    });
