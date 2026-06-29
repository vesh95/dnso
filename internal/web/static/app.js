// DNSO Web Interface — управление зонами и DNS-записями

let currentZoneName = null;
let confirmCallback = null;

// --- Zones ---

async function loadZones() {
    const tbody = document.getElementById('zones-tbody');
    tbody.innerHTML = '<tr><td colspan="4" class="loading">Загрузка...</td></tr>';

    try {
        const res = await fetch('/api/zones');
        const zones = await res.json();
        renderZones(zones);
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="4" class="empty">Ошибка загрузки: ${err.message}</td></tr>`;
    }
}

function renderZones(zones) {
    const tbody = document.getElementById('zones-tbody');

    if (!zones || zones.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="empty">Нет зон. Добавьте первую зону.</td></tr>';
        return;
    }

    tbody.innerHTML = zones.map(z => `
        <tr>
            <td>${z.Id}</td>
            <td><a class="zone-link" onclick="showRecords('${escapeHtml(z.Name)}')">${escapeHtml(z.Name)}</a></td>
            <td>${z.TTL}</td>
            <td class="actions">
                <button class="btn btn-sm" onclick="showEditZoneModal('${escapeHtml(z.Name)}', ${z.TTL})">✏️</button>
                <button class="btn btn-sm btn-danger" onclick="confirmDeleteZone('${escapeHtml(z.Name)}')">🗑️</button>
            </td>
        </tr>
    `).join('');
}

// --- Records ---

async function showRecords(zoneName) {
    currentZoneName = zoneName;
    document.getElementById('zones-section').classList.add('hidden');
    document.getElementById('records-section').classList.remove('hidden');
    document.getElementById('records-zone-name').textContent = zoneName;

    const tbody = document.getElementById('records-tbody');
    tbody.innerHTML = '<tr><td colspan="6" class="loading">Загрузка...</td></tr>';

    try {
        const res = await fetch(`/api/zones/${encodeURIComponent(zoneName)}/records`);
        const records = await res.json();
        renderRecords(records);
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="6" class="empty">Ошибка загрузки: ${err.message}</td></tr>`;
    }
}

function renderRecords(records) {
    const tbody = document.getElementById('records-tbody');

    if (!records || records.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty">Нет записей в этой зоне.</td></tr>';
        return;
    }

    tbody.innerHTML = records.map(r => `
        <tr>
            <td>${r.Id}</td>
            <td>${escapeHtml(r.Domain)}</td>
            <td><span class="type-badge">${escapeHtml(r.Type)}</span></td>
            <td>${escapeHtml(r.Rdata)}</td>
            <td>${r.TTL}</td>
            <td class="actions">
                <button class="btn btn-sm" onclick="showEditRecordModal(${r.Id}, '${escapeHtml(r.Domain)}', '${escapeHtml(r.Type)}', '${escapeHtml(r.Rdata)}', ${r.TTL})">✏️</button>
                <button class="btn btn-sm btn-danger" onclick="confirmDeleteRecord(${r.Id})">🗑️</button>
            </td>
        </tr>
    `).join('');
}

function showZones() {
    document.getElementById('records-section').classList.add('hidden');
    document.getElementById('zones-section').classList.remove('hidden');
    currentZoneName = null;
}

// --- Zone CRUD ---

function showAddZoneModal() {
    document.getElementById('zone-modal-title').textContent = 'Добавить зону';
    document.getElementById('zone-edit-name').value = '';
    document.getElementById('zone-name').value = '';
    document.getElementById('zone-name').disabled = false;
    document.getElementById('zone-ttl').value = '300';
    document.getElementById('zone-modal').classList.remove('hidden');
}

function showEditZoneModal(name, ttl) {
    document.getElementById('zone-modal-title').textContent = 'Редактировать зону';
    document.getElementById('zone-edit-name').value = name;
    document.getElementById('zone-name').value = name;
    document.getElementById('zone-name').disabled = true;
    document.getElementById('zone-ttl').value = ttl;
    document.getElementById('zone-modal').classList.remove('hidden');
}

async function saveZone(event) {
    event.preventDefault();

    const editName = document.getElementById('zone-edit-name').value;
    const name = document.getElementById('zone-name').value.trim();
    const ttl = parseInt(document.getElementById('zone-ttl').value) || 300;

    const isEdit = !!editName;

    try {
        let res;
        if (isEdit) {
            res = await fetch(`/api/zones/${encodeURIComponent(editName)}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ttl }),
            });
        } else {
            res = await fetch('/api/zones', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, ttl }),
            });
        }

        if (!res.ok) {
            const text = await res.text();
            alert(`Ошибка: ${text}`);
            return;
        }

        closeModal('zone-modal');
        loadZones();
    } catch (err) {
        alert(`Ошибка: ${err.message}`);
    }
}

function confirmDeleteZone(name) {
    confirmCallback = async () => {
        try {
            const res = await fetch(`/api/zones/${encodeURIComponent(name)}`, { method: 'DELETE' });
            if (!res.ok) {
                const text = await res.text();
                alert(`Ошибка: ${text}`);
                return;
            }
            closeModal('confirm-modal');
            loadZones();
        } catch (err) {
            alert(`Ошибка: ${err.message}`);
        }
    };
    document.getElementById('confirm-text').textContent = `Удалить зону "${name}" и все её записи?`;
    document.getElementById('confirm-btn').textContent = 'Удалить';
    document.getElementById('confirm-modal').classList.remove('hidden');
}

// --- Record CRUD ---

function showAddRecordModal() {
    document.getElementById('record-modal-title').textContent = 'Добавить запись';
    document.getElementById('record-edit-id').value = '';
    document.getElementById('record-domain').value = '';
    document.getElementById('record-type').value = 'A';
    document.getElementById('record-rdata').value = '';
    document.getElementById('record-ttl').value = '300';
    document.getElementById('record-modal').classList.remove('hidden');
}

function showEditRecordModal(id, domain, type, rdata, ttl) {
    document.getElementById('record-modal-title').textContent = 'Редактировать запись';
    document.getElementById('record-edit-id').value = id;
    document.getElementById('record-domain').value = domain;
    document.getElementById('record-type').value = type;
    document.getElementById('record-rdata').value = rdata;
    document.getElementById('record-ttl').value = ttl;
    document.getElementById('record-modal').classList.remove('hidden');
}

async function saveRecord(event) {
    event.preventDefault();

    const editId = document.getElementById('record-edit-id').value;
    const domain = document.getElementById('record-domain').value.trim();
    const type = document.getElementById('record-type').value;
    const rdata = document.getElementById('record-rdata').value.trim();
    const ttl = parseInt(document.getElementById('record-ttl').value) || 300;

    try {
        let res;

        if (editId) {
            // Для редактирования нужен zone_id — получаем зону
            const zoneRes = await fetch(`/api/zones/${encodeURIComponent(currentZoneName)}`);
            const zone = await zoneRes.json();

            res = await fetch(`/api/records/${editId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ zone_id: zone.Id, domain, type, rdata, ttl }),
            });
        } else {
            res = await fetch(`/api/zones/${encodeURIComponent(currentZoneName)}/records`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ domain, type, rdata, ttl }),
            });
        }

        if (!res.ok) {
            const text = await res.text();
            alert(`Ошибка: ${text}`);
            return;
        }

        closeModal('record-modal');
        showRecords(currentZoneName);
    } catch (err) {
        alert(`Ошибка: ${err.message}`);
    }
}

function confirmDeleteRecord(id) {
    confirmCallback = async () => {
        try {
            const res = await fetch(`/api/records/${id}`, { method: 'DELETE' });
            if (!res.ok) {
                const text = await res.text();
                alert(`Ошибка: ${text}`);
                return;
            }
            closeModal('confirm-modal');
            showRecords(currentZoneName);
        } catch (err) {
            alert(`Ошибка: ${err.message}`);
        }
    };
    document.getElementById('confirm-text').textContent = 'Удалить эту запись?';
    document.getElementById('confirm-btn').textContent = 'Удалить';
    document.getElementById('confirm-modal').classList.remove('hidden');
}

// --- Modals ---

function closeModal(id) {
    document.getElementById(id).classList.add('hidden');
}

function confirmAction() {
    if (confirmCallback) {
        confirmCallback();
        confirmCallback = null;
    }
}

// Close modal on outside click
document.addEventListener('click', (event) => {
    document.querySelectorAll('.modal:not(.hidden)').forEach(modal => {
        if (event.target === modal) {
            modal.classList.add('hidden');
        }
    });
});

// Close modal on Escape
document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
        document.querySelectorAll('.modal:not(.hidden)').forEach(modal => {
            modal.classList.add('hidden');
        });
    }
});

// --- Helpers ---

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// --- Init ---

document.addEventListener('DOMContentLoaded', loadZones);