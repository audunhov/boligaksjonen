// Initialize the map centered on Oslo
document.addEventListener('DOMContentLoaded', () => {
    // 1. Initialize Leaflet (disable default zoom control)
    const map = L.map('map', {
        zoomControl: false
    }).setView([59.9139, 10.7522], 13);
    
    // Add custom zoom control at bottomleft
    L.control.zoom({
        position: 'bottomleft'
    }).addTo(map);
    
    const topo = L.tileLayer('https://cache.kartverket.no/v1/wmts/1.0.0/topo/default/webmercator/{z}/{y}/{x}.png', {
        maxZoom: 18,
        attribution: '&copy; <a href="http://www.kartverket.no/">Kartverket</a>'
    });

    const detailed = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    });

    topo.addTo(map);
// Add Layer Control
const baseMaps = {
    "Kartverket": topo,
    "OpenStreetMap": detailed
};
L.control.layers(baseMaps, null, { position: 'bottomright' }).addTo(map);


    const markers = L.markerClusterGroup();
    let allHouses = [];

    // Helper for debouncing
    function debounce(func, timeout = 300) {
        let timer;
        return (...args) => {
            clearTimeout(timer);
            timer = setTimeout(() => { func.apply(this, args); }, timeout);
        };
    }

    const debouncedHeaderLookup = debounce((val) => window.lookupCoordinatesHeader(val));
    const debouncedModalLookup = debounce(() => window.lookupCoordinates());
    window.debouncedHeaderLookup = debouncedHeaderLookup;
    window.debouncedModalLookup = debouncedModalLookup;

    function getKartverketUrl(h) {
        if (!h.knr || !h.gnr || !h.bnr) return null;
        return `https://eiendomsregisteret.kartverket.no/eiendom/${h.knr}/${h.gnr}/${h.bnr}/${h.fnr || 0}/${h.snr || 0}`;
    }

    function timeAgo(date) {
        const seconds = Math.floor((new Date() - new Date(date)) / 1000);
        let interval = seconds / 31536000;
        if (interval > 1) return Math.floor(interval) + " år siden";
        interval = seconds / 2592000;
        if (interval > 1) return Math.floor(interval) + " mnd siden";
        interval = seconds / 86400;
        if (interval > 1) return Math.floor(interval) + " dager siden";
        interval = seconds / 3600;
        if (interval > 1) return Math.floor(interval) + " timer siden";
        interval = seconds / 60;
        if (interval > 1) return Math.floor(interval) + " min siden";
        return "akkurat nå";
    }

    // 2. Define global UI helpers
    window.openAddModal = function() {
        document.getElementById('editForm').reset();
        document.getElementById('edit_id').value = "0";
        document.getElementById('dialog_title').innerText = "Rapporter tom bolig";
        document.getElementById('edit_freetext_group').classList.add('hidden');
        document.getElementById('lookup_status').innerText = '';
        document.getElementById('kartverket_link').classList.add('hidden');
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
        
        document.getElementById('edit_knr').value = house.knr || '';
        document.getElementById('edit_gnr').value = house.gnr || 0;
        document.getElementById('edit_bnr').value = house.bnr || 0;
        document.getElementById('edit_fnr').value = house.fnr || 0;
        document.getElementById('edit_snr').value = house.snr || 0;

        const kvLink = document.getElementById('kartverket_link');
        const url = getKartverketUrl(house);
        if (url) {
            kvLink.href = url;
            kvLink.classList.remove('hidden');
        } else {
            kvLink.classList.add('hidden');
        }
        
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

    window.closeAside = function() {
        document.getElementById('detail_aside').classList.remove('is-open');
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
                    const existingHouse = allHouses.find(h => h.address === addr.adressetekst);
                    const item = document.createElement('div');
                    item.className = 'p-4 cursor-pointer border-b border-gray-50 hover:bg-blue-50 transition-colors text-sm last:border-none flex justify-between items-center';
                    let actionText = existingHouse ? 'Vis' : 'Rapporter';
                    let actionColor = existingHouse ? 'text-green-600' : 'text-blue-600';
                    item.innerHTML = `<div><span class="font-bold text-gray-900 text-xs sm:text-sm">${addr.adressetekst}</span><span class="text-gray-400 ml-2 text-[10px] sm:text-xs">${addr.poststed}</span></div><span class="text-[10px] font-black ${actionColor} uppercase tracking-widest">${actionText}</span>`;
                    item.onclick = () => {
                        resultsDiv.classList.add('hidden');
                        if (existingHouse) {
                            map.flyTo([existingHouse.lat, existingHouse.lng], 17);
                            window.showHouseDetails(existingHouse);
                        } else {
                            window.openAddModal();
                            window.selectAddress(addr);
                        }
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

    window.selectAddress = function(addr) {
        document.getElementById('edit_address').value = addr.adressetekst;
        document.getElementById('edit_lat').value = addr.representasjonspunkt.lat;
        document.getElementById('edit_lng').value = addr.representasjonspunkt.lon;
        
        // Priority: Use first matrikkelenhet if available for accurate snr/fnr, 
        // otherwise fallback to top-level fields.
        const m = (addr.matrikkelenheter && addr.matrikkelenheter.length > 0) ? addr.matrikkelenheter[0] : addr;

        document.getElementById('edit_knr').value = m.kommunenummer;
        document.getElementById('edit_gnr').value = m.gardsnummer;
        document.getElementById('edit_bnr').value = m.bruksnummer;
        document.getElementById('edit_fnr').value = m.festenummer || 0;
        document.getElementById('edit_snr').value = m.seksjonsnummer || 0;

        const statusDiv = document.getElementById('lookup_status');
        const resultsDiv = document.getElementById('lookup_results');
        statusDiv.innerText = `Valgt: ${addr.adressetekst}`;
        statusDiv.classList.add('text-green-500');
        resultsDiv.classList.add('hidden');

        // Show direct Kartverket link using the detected matrikkel data
        const kvLink = document.getElementById('kartverket_link');
        const url = `https://eiendomsregisteret.kartverket.no/eiendom/${m.kommunenummer}/${m.gardsnummer}/${m.bruksnummer}/${m.festenummer || 0}/${m.seksjonsnummer || 0}`;
        kvLink.href = url;
        kvLink.classList.remove('hidden');

        map.flyTo([addr.representasjonspunkt.lat, addr.representasjonspunkt.lon], 17);
    }

    window.showHouseDetails = function(house) {
        const aside = document.getElementById('detail_aside');
        const placeholder = document.getElementById('aside_placeholder');
        const content = document.getElementById('aside_content');
        if (placeholder) placeholder.classList.add('hidden');
        if (content) content.classList.remove('hidden');
        
        aside.classList.add('is-open');

        document.getElementById('aside_address').innerText = house.address;
        document.getElementById('aside_description').innerHTML = house.description.replace(/\r\n/g, '<br>').replace(/\n/g, '<br>');
        document.getElementById('aside_author').innerText = house.last_updated_by;
        document.getElementById('aside_date').innerText = new Date(house.updated_at).toLocaleString('no-NO');
        document.getElementById('remove_house_id').value = house.id;
        document.getElementById('comment_house_id').value = house.id;
        document.getElementById('comment_section').classList.remove('hidden');
        
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
        const kvUrl = getKartverketUrl(house);
        if (kvUrl) {
            badge.innerHTML += `<a href="${kvUrl}" target="_blank" class="ml-2 text-[10px] font-bold text-blue-600 hover:underline">Se i matrikkelen →</a>`;
        }

        window.refreshHistory(house.id);
    }

    window.refreshHistory = function(houseID) {
        const historyList = document.getElementById('aside_history_list');
        historyList.innerHTML = '<div class="text-xs text-gray-400">Laster historikk...</div>';
        
        fetch(`/api/houses/history?id=${houseID}`)
            .then(res => res.json())
            .then(logs => {
                if (logs.length === 0) {
                    historyList.innerHTML = '<div class="text-xs text-gray-400 italic">Ingen historikk funnet.</div>';
                    return;
                }
                historyList.innerHTML = '';
                logs.forEach(log => {
                    const li = document.createElement('li');
                    li.className = 'relative flex gap-x-4';
                    const timeStr = timeAgo(log.timestamp);
                    const author = log.username || 'Anonym (' + log.anon_hash + ')';
                    const initial = author.charAt(0).toUpperCase();

                    if (log.action === 'comment') {
                        li.innerHTML = `
                            <div class="absolute -bottom-6 left-0 top-0 flex w-6 justify-center"><div class="w-px bg-gray-200"></div></div>
                            <div class="relative mt-3 size-6 flex-none rounded-full bg-blue-100 text-blue-600 flex items-center justify-center font-bold text-[10px] uppercase outline outline-1 -outline-offset-1 outline-black/5">${initial}</div>
                            <div class="flex-auto rounded-md p-3 ring-1 ring-inset ring-gray-200 bg-white shadow-sm">
                                <div class="flex justify-between gap-x-4">
                                    <div class="py-0.5 text-xs/5 text-gray-500"><span class="font-medium text-gray-900">${author}</span></div>
                                    <time class="flex-none py-0.5 text-[10px] text-gray-400">${timeStr}</time>
                                </div>
                                <p class="text-xs sm:text-sm/6 text-gray-600">${log.new_data}</p>
                            </div>
                        `;
                    } else {
                        let actionMsg = "";
                        switch(log.action) {
                            case 'add': actionMsg = "rapporterte boligen"; break;
                            case 'update': actionMsg = "oppdaterte oppføringen"; break;
                            case 'remove': actionMsg = "fjernet oppføringen"; break;
                            case 'restore': actionMsg = "gjenopprettet oppføringen"; break;
                        }
                        li.innerHTML = `
                            <div class="absolute -bottom-6 left-0 top-0 flex w-6 justify-center"><div class="w-px bg-gray-200"></div></div>
                            <div class="relative flex size-6 flex-none items-center justify-center bg-gray-50">
                                <div class="size-1.5 rounded-full bg-gray-100 ring ring-gray-300"></div>
                            </div>
                            <p class="flex-auto py-0.5 text-xs/5 text-gray-500"><span class="font-medium text-gray-900">${author}</span> ${actionMsg}.</p>
                            <time class="flex-none py-0.5 text-[10px] text-gray-400">${timeStr}</time>
                        `;
                    }
                    historyList.appendChild(li);
                });
            });
    }

    window.submitComment = function(e) {
        e.preventDefault();
        const houseID = document.getElementById('comment_house_id').value;
        const comment = document.getElementById('comment_text').value;
        if (!comment) return;
        const params = new URLSearchParams();
        params.append('house_id', houseID);
        params.append('comment', comment);
        fetch('/api/houses/comment', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: params
        }).then(res => {
            if (res.ok) {
                document.getElementById('comment_text').value = '';
                window.refreshHistory(houseID);
            }
        });
    }

    // 3. Map Interaction Handlers
    map.on('click', () => {
        // Hide sidebar/bottom-sheet on mobile/tablet when map is clicked
        if (window.innerWidth < 1024) {
            window.closeAside();
        }
    });

    map.on('dblclick', async (e) => {
        const { lat, lng } = e.latlng;
        try {
            const response = await fetch(`https://ws.geonorge.no/adresser/v1/punktsok?lon=${lng}&lat=${lat}&radius=10&treffPerSide=1`);
            const data = await response.json();
            if (data.adresser && data.adresser.length > 0) {
                const addr = data.adresser[0];
                window.openAddModal();
                window.selectAddress(addr);
            }
        } catch (error) {
            console.error('Reverse geocoding error:', error);
        }
    });

    // 4. Fetch data
    fetch('/api/houses')
        .then(res => res.json())
        .then(houses => {
            allHouses = houses;
            houses.forEach(h => {
                const m = L.marker([h.lat, h.lng]);
                m.on('click', () => {
                    // Pre-fill sidebar but don't force open immediately on all clicks if popups are used
                    window.showHouseDetails(h);
                });
                
                const popupContent = `
                    <div class="p-4 min-w-[180px] max-w-[240px]">
                        <h4 class="text-sm font-black text-gray-900 leading-tight mb-3">${h.address}</h4>
                        <button class="w-full bg-blue-600 hover:bg-blue-700 text-white text-[10px] font-black uppercase tracking-widest py-2.5 rounded-lg shadow-md shadow-blue-100 transition-all active:scale-95" onclick="window.showHouseDetailsById(${h.id})">
                            Vis detaljer
                        </button>
                    </div>
                `;
                m.bindPopup(popupContent);
                markers.addLayer(m);
            });
            map.addLayer(markers);
        });

    window.showHouseDetailsById = function(id) {
        const h = allHouses.find(house => house.id === id);
        if (h) window.showHouseDetails(h);
    }

    setTimeout(() => map.invalidateSize(), 100);
});
