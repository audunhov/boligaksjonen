// Map Logic for Boligaksjonen
document.addEventListener('DOMContentLoaded', () => {
    // Load saved position or default to Oslo
    const savedPos = JSON.parse(localStorage.getItem('map_position') || '{"lat": 59.9139, "lng": 10.7522, "zoom": 13}');

    // 1. Initialize Leaflet (disable default zoom control)
    const map = L.map('map', {
        zoomControl: false
    }).setView([savedPos.lat, savedPos.lng], savedPos.zoom);
    window.map = map; // Export to window for access from Templ scripts

    // Save position on move or zoom
    map.on('moveend', () => {
        const center = map.getCenter();
        const state = {
            lat: center.lat,
            lng: center.lng,
            zoom: map.getZoom()
        };
        localStorage.setItem('map_position', JSON.stringify(state));
    });
    
    // Add custom zoom control at bottomleft
    L.control.zoom({
        position: 'bottomleft'
    }).addTo(map);
    
    // Define Base Layers
    const topo = L.tileLayer('https://cache.kartverket.no/v1/wmts/1.0.0/topo/default/webmercator/{z}/{y}/{x}.png', {
        maxZoom: 18,
        attribution: '&copy; <a href="http://www.kartverket.no/">Kartverket</a>'
    });

    const detailed = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    });

    topo.addTo(map);

    const baseMaps = {
        "Kartverket": topo,
        "OpenStreetMap": detailed
    };
    L.control.layers(baseMaps, null, { position: 'bottomright' }).addTo(map);

    const markers = L.markerClusterGroup();

    // 2. Define global UI helpers
    window.openAddModal = function() {
        // Use HTMX to load a fresh empty form
        htmx.ajax('GET', '/houses/edit?id=0', {target: '#edit_dialog_container', swap: 'innerHTML'});
        
        // Listen for the specific request to finish
        const onFinish = (e) => {
            if (e.detail.pathInfo.requestPath === '/houses/edit?id=0') {
                document.getElementById('editDialog').showModal();
                document.body.removeEventListener('htmx:afterOnLoad', onFinish);
            }
        };
        document.body.addEventListener('htmx:afterOnLoad', onFinish);
    }

    window.toggleFreetext = function(value) {
        const freetextGroup = document.getElementById('edit_freetext_group');
        if (value === 'freetext') {
            freetextGroup.classList.remove('hidden');
        } else {
            freetextGroup.classList.add('hidden');
        }
    }

    window.closeAside = function() {
        const aside = document.getElementById('detail_aside');
        if (aside) {
            aside.classList.remove('is-open');
            // Give the transition time to finish before invalidating map size
            setTimeout(() => {
                if (window.map) window.map.invalidateSize();
            }, 350);
        }
        
        const url = new URL(window.location);
        if (url.searchParams.has('id')) {
            url.searchParams.delete('id');
            window.history.pushState({}, '', url);
        }
    }

    window.selectAddress = function(addr) {
        const addrField = document.getElementById('edit_address');
        const latField = document.getElementById('edit_lat');
        const lngField = document.getElementById('edit_lng');
        const knrField = document.getElementById('edit_knr');
        const gnrField = document.getElementById('edit_gnr');
        const bnrField = document.getElementById('edit_bnr');
        const fnrField = document.getElementById('edit_fnr');
        const snrField = document.getElementById('edit_snr');

        if(addrField) addrField.value = addr.adressetekst;
        if(latField) latField.value = addr.representasjonspunkt.lat;
        if(lngField) lngField.value = addr.representasjonspunkt.lon;
        if(knrField) knrField.value = addr.kommunenummer;
        if(gnrField) gnrField.value = addr.gardsnummer;
        if(bnrField) bnrField.value = addr.bruksnummer;
        if(fnrField) fnrField.value = addr.festenummer;
        if(snrField) snrField.value = addr.seksjonsnummer;
        
        const statusDiv = document.getElementById('lookup_status');
        const resultsDiv = document.getElementById('lookup_results');
        if(statusDiv) {
            statusDiv.innerText = `Valgt: ${addr.adressetekst}`;
            statusDiv.classList.add('text-green-500');
        }
        if(resultsDiv) resultsDiv.classList.add('hidden');
        
        map.flyTo([addr.representasjonspunkt.lat, addr.representasjonspunkt.lon], 17);
    }

    window.showHouseDetails = function(house) {
        const aside = document.getElementById('detail_aside');
        if (aside) {
            aside.classList.add('is-open');
            setTimeout(() => {
                if (window.map) window.map.invalidateSize();
            }, 350);
        }

        const wrapper = document.getElementById('aside_content_wrapper');
        if (wrapper) {
            wrapper.innerHTML = '<div class="text-center py-20 text-gray-400 text-sm font-medium">Laster detaljer...</div>';
            htmx.ajax('GET', `/houses/details?id=${house.id}`, {target: '#aside_content_wrapper', swap: 'innerHTML'});
        }

        const url = new URL(window.location);
        if (url.searchParams.get('id') !== String(house.id)) {
            url.searchParams.set('id', house.id);
            window.history.pushState({houseId: house.id}, '', url);
        }
    }


    // 3. Map Interaction Handlers
    map.on('click', () => {
        window.closeAside();
    });

    map.on('dblclick', async (e) => {
        const { lat, lng } = e.latlng;
        try {
            const response = await fetch(`https://ws.geonorge.no/adresser/v1/punktsok?lon=${lng}&lat=${lat}&radius=10&treffPerSide=1`);
            const data = await response.json();
            if (data.adresser && data.adresser.length > 0) {
                const addr = data.adresser[0];
                window.openAddModal();
                // Wait for HTMX to potentially finish loading the modal before selecting address
                setTimeout(() => window.selectAddress(addr), 100);
            }
        } catch (error) {
            console.error('Reverse geocoding error:', error);
        }
    });

    // 4. Fetch data
    fetch('/api/houses')
        .then(res => {
            if (!res.ok) throw new Error('Failed to fetch houses');
            const contentType = res.headers.get('content-type');
            if (!contentType || !contentType.includes('application/json')) {
                throw new TypeError("Oops, we haven't got JSON!");
            }
            return res.json();
        })
        .then(houses => {
            const customIcon = L.divIcon({
                className: 'custom-div-icon',
                html: `
                    <svg class="size-10 drop-shadow-xl" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <path d="M12 2C8.13 2 5 5.13 5 9C5 14.25 12 22 12 22C12 22 19 14.25 19 9C19 5.13 15.87 2 12 2Z" fill="#2563EB" stroke="#1E40AF" stroke-width="0.5"/>
                        <circle cx="12" cy="9" r="2.5" fill="#1E40AF"/>
                    </svg>
                `,
                iconSize: [40, 40],
                iconAnchor: [20, 40]
            });

            const urlParams = new URLSearchParams(window.location.search);
            const initialId = urlParams.get('id');
            let targetHouse = null;

            houses.forEach(h => {
                // Only show active houses on the map
                if (h.is_deleted) return;

                if (initialId && h.id === parseInt(initialId)) {
                    targetHouse = h;
                }

                const m = L.marker([h.lat, h.lng], { icon: customIcon });
                m.on('click', () => {
                    map.flyTo([h.lat, h.lng], 17);
                    window.showHouseDetails(h);
                });
                markers.addLayer(m);
            });
            map.addLayer(markers);

            if (targetHouse) {
                map.flyTo([targetHouse.lat, targetHouse.lng], 17);
                window.showHouseDetails(targetHouse);
            }
        });

    window.showHouseDetailsById = function(id) {
        window.showHouseDetails({id: id});
    }

    setTimeout(() => map.invalidateSize(), 100);
});
