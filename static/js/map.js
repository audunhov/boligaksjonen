// Initialize the map centered on Oslo
document.addEventListener('DOMContentLoaded', () => {
    // 1. Initialize Leaflet
    const map = L.map('map').setView([59.9139, 10.7522], 13);
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }).addTo(map);

    const markers = L.markerClusterGroup();
    let allHouses = [];

    // 2. Define global UI helpers
    window.openAddModal = function() {
        document.getElementById('editForm').reset();
        document.getElementById('edit_id').value = "0";
        document.getElementById('dialog_title').innerText = "Rapporter tom bolig";
        document.getElementById('edit_freetext_group').classList.add('hidden');
        document.getElementById('lookup_status').innerText = '';
        document.getElementById('editDialog').showModal();
    }

    window.openEditModal = function(id) {
        const house = allHouses.find(h => h.id === id);
        if (!house) return;

        document.getElementById('edit_id').value = house.id;
        document.getElementById('edit_address').value = house.address;
        document.getElementById('edit_lat').value = house.lat;
        document.getElementById('edit_lng').value = house.lng;
        document.getElementById('edit_description').value = house.description;
        
        const standardTypes = ['privat-bolig', 'privat-bygård', 'offentlig', 'næring', 'industri', 'gård', 'annet'];
        if (standardTypes.includes(house.ownership_type)) {
            document.getElementById('edit_ownership_type').value = house.ownership_type;
            document.getElementById('edit_freetext_group').classList.add('hidden');
        } else {
            document.getElementById('edit_ownership_type').value = 'freetext';
            document.getElementById('edit_freetext_group').classList.remove('hidden');
            document.getElementById('edit_ownership_freetext').value = house.ownership_type;
        }
        document.getElementById('editDialog').showModal();
    };

    window.toggleFreetext = function(value) {
        const freetextGroup = document.getElementById('edit_freetext_group');
        if (value === 'freetext') {
            freetextGroup.classList.remove('hidden');
        } else {
            freetextGroup.classList.add('hidden');
        }
    }

    window.lookupCoordinatesHeader = async function(address) {
        const resultsDiv = document.getElementById('header_lookup_results');
        if (!address) {
            resultsDiv.classList.add('hidden');
            return;
        }

        try {
            const response = await fetch(`https://ws.geonorge.no/adresser/v1/sok?sok=${encodeURIComponent(address)}&treffPerSide=10`);
            const data = await response.json();

            if (data.adresser && data.adresser.length > 0) {
                resultsDiv.classList.remove('hidden');
                resultsDiv.innerHTML = '';
                data.adresser.forEach(addr => {
                    const item = document.createElement('div');
                    item.className = 'p-4 cursor-pointer border-b border-gray-50 hover:bg-blue-50 transition-colors text-sm last:border-none flex justify-between items-center';
                    item.innerHTML = `<div><span class="font-bold text-gray-900">${addr.adressetekst}</span><span class="text-gray-400 ml-2">${addr.poststed}</span></div><span class="text-[10px] font-black text-blue-600 uppercase tracking-widest">Velg</span>`;
                    item.onclick = () => {
                        resultsDiv.classList.add('hidden');
                        window.openAddModal();
                        window.selectAddress(addr);
                    };
                    resultsDiv.appendChild(item);
                });
            }
        } catch (error) {
            console.error('Header lookup error:', error);
        }
    }

    window.lookupCoordinates = async function() {
        const address = document.getElementById('edit_address').value;
        const statusDiv = document.getElementById('lookup_status');
        const resultsDiv = document.getElementById('lookup_results');
        
        if (!address) {
            statusDiv.innerText = 'Vennligst skriv inn en adresse.';
            statusDiv.classList.add('text-red-500');
            return;
        }

        statusDiv.innerText = 'Slår opp...';
        statusDiv.classList.remove('text-red-500', 'text-green-500');
        resultsDiv.classList.add('hidden');
        resultsDiv.innerHTML = '';

        try {
            const response = await fetch(`https://ws.geonorge.no/adresser/v1/sok?sok=${encodeURIComponent(address)}&treffPerSide=10`);
            const data = await response.json();

            if (data.adresser && data.adresser.length > 0) {
                statusDiv.innerText = `Fant ${data.adresser.length} resultater:`;
                resultsDiv.classList.remove('hidden');
                data.adresser.forEach(addr => {
                    const item = document.createElement('div');
                    item.className = 'p-3 cursor-pointer border-b border-gray-50 hover:bg-blue-50 transition-colors text-xs last:border-none';
                    item.innerHTML = `<span class="font-bold text-gray-800">${addr.adressetekst}</span> <span class="text-gray-500 ml-1">${addr.poststed}</span>`;
                    item.onclick = () => window.selectAddress(addr);
                    resultsDiv.appendChild(item);
                });
            } else {
                statusDiv.innerText = 'Ingen adresser funnet.';
                statusDiv.classList.add('text-red-500');
            }
        } catch (error) {
            statusDiv.innerText = 'Tilkoblingsfeil.';
            statusDiv.classList.add('text-red-500');
        }
    }

    let searchMarker = null;

    window.selectAddress = function(addr) {
        document.getElementById('edit_address').value = addr.adressetekst;
        document.getElementById('edit_lat').value = addr.representasjonspunkt.lat;
        document.getElementById('edit_lng').value = addr.representasjonspunkt.lon;
        const statusDiv = document.getElementById('lookup_status');
        const resultsDiv = document.getElementById('lookup_results');
        statusDiv.innerText = `Valgt: ${addr.adressetekst}`;
        statusDiv.classList.add('text-green-500');
        resultsDiv.classList.add('hidden');

        // Move map and add temporary search marker
        const latlng = [addr.representasjonspunkt.lat, addr.representasjonspunkt.lon];
        map.flyTo(latlng, 17);
        
        if (searchMarker) map.removeLayer(searchMarker);
        searchMarker = L.marker(latlng, {
            icon: L.divIcon({
                className: 'custom-div-icon',
                html: "<div class='size-4 bg-blue-600 border-2 border-white rounded-full shadow-lg animate-bounce'></div>",
                iconSize: [16, 16],
                iconAnchor: [8, 8]
            })
        }).addTo(map);
    }

    window.showHouseDetails = function(house) {
        const placeholder = document.getElementById('aside_placeholder');
        const content = document.getElementById('aside_content');
        
        if (placeholder) placeholder.classList.add('hidden');
        if (content) content.classList.remove('hidden');
        
        document.getElementById('aside_address').innerText = house.address;
        document.getElementById('aside_description').innerHTML = house.description.replace(/\r\n/g, '<br>').replace(/\n/g, '<br>');
        document.getElementById('aside_author').innerText = house.last_updated_by;
        document.getElementById('aside_date').innerText = new Date(house.updated_at).toLocaleString('no-NO');
        
        const badge = document.getElementById('aside_badge');
        const type = (house.ownership_type || 'annet').toLowerCase().trim();
        const colors = {
            'privat-bolig': 'bg-green-100 text-green-700',
            'privat-bygård': 'bg-green-100 text-green-700',
            'offentlig': 'bg-blue-100 text-blue-700',
            'næring': 'bg-yellow-100 text-yellow-800',
            'industri': 'bg-gray-100 text-gray-700',
            'gård': 'bg-orange-100 text-orange-800',
            'annet': 'bg-gray-100 text-gray-500'
        };
        badge.innerHTML = `<span class="px-2.5 py-0.5 rounded-full text-[10px] font-black uppercase tracking-widest ${colors[type] || 'bg-gray-100 text-gray-500'}">${type.replace('-', ' ')}</span>`;
        document.getElementById('aside_edit_btn').onclick = () => window.openEditModal(house.id);

        // Fetch and show history
        const historyList = document.getElementById('aside_history_list');
        historyList.innerHTML = '<div class="text-xs text-gray-400">Laster historikk...</div>';
        
        fetch(`/api/houses/history?id=${house.id}`)
            .then(res => res.json())
            .then(logs => {
                if (logs.length === 0) {
                    historyList.innerHTML = '<div class="text-xs text-gray-400 italic">Ingen historikk funnet.</div>';
                    return;
                }
                historyList.innerHTML = '';
                logs.forEach(log => {
                    const item = document.createElement('div');
                    item.className = 'relative pl-6 pb-6 border-l border-gray-100 last:border-0';
                    item.innerHTML = `
                        <div class="absolute -left-[5px] top-1.5 size-2.5 rounded-full border-2 border-white ${log.action === 'add' ? 'bg-green-500' : 'bg-blue-500'}"></div>
                        <div class="flex justify-between items-start mb-1">
                            <span class="text-[10px] font-black uppercase tracking-widest ${log.action === 'add' ? 'text-green-600' : 'text-blue-600'}">${log.action === 'add' ? 'Opprettet' : 'Endret'}</span>
                            <span class="text-[10px] font-bold text-gray-400">${new Date(log.timestamp).toLocaleDateString('no-NO')}</span>
                        </div>
                        <p class="text-xs font-bold text-gray-900 mb-1">${log.username || 'Anonym (' + log.anon_hash + ')'}</p>
                    `;
                    historyList.appendChild(item);
                });
            })
            .catch(err => {
                console.error('Failed to fetch history:', err);
                historyList.innerHTML = '<div class="text-xs text-red-500">Klarte ikke å hente historikk.</div>';
            });
    }

    // 3. Fetch data
    fetch('/api/houses')
        .then(res => res.json())
        .then(houses => {
            allHouses = houses;
            houses.forEach(h => {
                // IMPORTANT: Use JSON tag names 'lat' and 'lng'
                const m = L.marker([h.lat, h.lng]);
                m.on('click', () => window.showHouseDetails(h));
                m.bindPopup(`<b class="text-gray-900">${h.address}</b><br><span class="text-[10px] font-black uppercase text-blue-600">${h.ownership_type.replace('-', ' ')}</span>`);
                markers.addLayer(m);
            });
            map.addLayer(markers);
        });

    setTimeout(() => map.invalidateSize(), 100);
});
