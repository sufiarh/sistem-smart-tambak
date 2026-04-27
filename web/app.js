// ============================================================
// SMART TAMBAK — Frontend Logic
// ============================================================

// Interval polling data (ms)
const POLL_INTERVAL = 5000

// Simpan data historis untuk chart
let historyData = []
let chartInstance = null

// ============================================================
// TAB NAVIGATION
// ============================================================

function showTab(name) {
    // Sembunyikan semua tab
    document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'))
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'))

    // Tampilkan tab yang dipilih
    document.getElementById('tab-' + name).classList.add('active')
    event.target.classList.add('active')

    // Load data historis saat tab history dibuka
    if (name === 'history') {
        fetchHistory()
    }
}

// ============================================================
// POLLING — Ambil data sensor & relay setiap 5 detik
// ============================================================

async function pollData() {
    try {
        await Promise.all([
            fetchSensor(),
                          fetchRelay(),
                          fetchStatus(),
                          fetchMode(),
        ])
    } catch (err) {
        console.error('Polling error:', err)
    }
}

// ============================================================
// FETCH SENSOR — Update kartu sensor
// ============================================================

async function fetchSensor() {
    const res = await fetch('/api/sensor')
    if (!res.ok) return
        const d = await res.json()

        // Update nilai
        setValue('val-ph', d.ph.toFixed(2))
        setValue('val-do', d.do.toFixed(2))
        setValue('val-temp', d.temperature.toFixed(1))
        setValue('val-sal', d.salinity.toFixed(1))
        setValue('val-turb', d.turbidity.toFixed(1))
        setValue('val-level', d.water_level.toFixed(2))
        setValue('val-light', d.light.toFixed(0))

        // Update badge & warna kartu
        updateSensorCard('card-ph', 'badge-ph', d.ph,
                         v => v >= 7.5 && v <= 8.5 ? 'ok' : v >= 7.0 && v <= 9.0 ? 'warn' : 'crit')

        updateSensorCard('card-do', 'badge-do', d.do,
                         v => v >= 4.0 ? 'ok' : v >= 2.0 ? 'warn' : 'crit')

        updateSensorCard('card-temp', 'badge-temp', d.temperature,
                         v => v >= 26 && v <= 30 ? 'ok' : v >= 24 && v <= 32 ? 'warn' : 'crit')

        updateSensorCard('card-sal', 'badge-sal', d.salinity,
                         v => v >= 15 && v <= 25 ? 'ok' : v >= 10 && v <= 28 ? 'warn' : 'crit')

        updateSensorCard('card-turb', 'badge-turb', d.turbidity,
                         v => v < 30 ? 'ok' : v < 50 ? 'warn' : 'crit')

        updateSensorCard('card-level', 'badge-level', d.water_level,
                         v => v >= 1.0 ? 'ok' : v >= 0.8 ? 'warn' : 'crit')

        updateSensorCard('card-light', 'badge-light', d.light,
                         v => 'ok') // Cahaya tidak ada threshold kritis

        // Update timestamp
        document.getElementById('last-update').textContent = 'Update: ' + d.timestamp
}

// ============================================================
// FETCH RELAY — Update status aktuator
// ============================================================

async function fetchRelay() {
    const res = await fetch('/api/relay')
    if (!res.ok) return
        const d = await res.json()

        updateActuator('act-aerator1', d.aerator1)
        updateActuator('act-aerator2', d.aerator2)
        updateActuator('act-pump_in', d.pump_in)
        updateActuator('act-pump_out', d.pump_out)
        updateActuator('act-feeder', d.feeder)
        updateActuator('act-lamp', d.lamp)
}

// ============================================================
// FETCH STATUS — Update indikator status besar
// ============================================================

async function fetchStatus() {
    const res = await fetch('/api/status')
    if (!res.ok) return
        const d = await res.json()

        const bar = document.getElementById('status-bar')
        const text = document.getElementById('status-text')
        const badge = document.getElementById('status-badge')

        // Reset class
        bar.className = 'status-bar'
        badge.className = 'badge'

        switch (d.status) {
            case 'normal':
                bar.classList.add('status-normal')
                badge.classList.add('badge-normal')
                badge.textContent = '● Normal'
                break
            case 'warning':
                bar.classList.add('status-warning')
                badge.classList.add('badge-warning')
                badge.textContent = '⚠ Warning'
                break
            case 'danger':
                bar.classList.add('status-danger')
                badge.classList.add('badge-danger')
                badge.textContent = '🚨 Kritis'
                break
            default:
                badge.classList.add('badge-loading')
                badge.textContent = 'Memuat...'
        }

        text.textContent = d.message
}

// ============================================================
// FETCH MODE — Update tampilan mode AUTO/MANUAL
// ============================================================

async function fetchMode() {
    const res = await fetch('/api/mode')
    if (!res.ok) return
        const modes = await res.json()

        for (const [actuator, mode] of Object.entries(modes)) {
            // Update label mode di dashboard
            const el = document.getElementById('mode-' + actuator)
            if (el) el.textContent = mode

                // Update tombol AUTO/MANUAL di tab kontrol
                const btnAuto   = document.getElementById('btn-auto-' + actuator)
                const btnManual = document.getElementById('btn-manual-' + actuator)
                if (btnAuto && btnManual) {
                    btnAuto.classList.toggle('active', mode === 'AUTO')
                    btnManual.classList.toggle('active', mode === 'MANUAL')
                }
        }
}

// ============================================================
// FETCH HISTORY — Ambil data historis untuk chart & tabel
// ============================================================

async function fetchHistory() {
    const res = await fetch('/api/history?hours=24')
    if (!res.ok) return
        historyData = await res.json()

        renderChart()
        renderTable()
}

// ============================================================
// RENDER CHART — Chart.js
// ============================================================

function renderChart() {
    if (!historyData || historyData.length === 0) return

        const param = document.getElementById('chart-param').value
        const labels = historyData.map(d => d.timestamp)
        const values = historyData.map(d => d[param])

        // Warna garis per parameter
        const colors = {
            ph: '#3498db',
            do: '#2ecc71',
                temperature: '#e74c3c',
                salinity: '#9b59b6',
                turbidity: '#e67e22',
                water_level: '#1abc9c',
        }

        const color = colors[param] || '#2ecc71'

        // Destroy chart lama jika ada
        if (chartInstance) {
            chartInstance.destroy()
        }

        const ctx = document.getElementById('sensorChart').getContext('2d')
        chartInstance = new Chart(ctx, {
            type: 'line',
            data: {
                labels,
                datasets: [{
                    label: paramLabel(param),
                                  data: values,
                                  borderColor: color,
                                  backgroundColor: color + '22',
                                  borderWidth: 2,
                                  pointRadius: 2,
                                  fill: true,
                                  tension: 0.3,
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: true },
                    tooltip: { mode: 'index', intersect: false }
                },
                scales: {
                    x: {
                        ticks: {
                            maxTicksLimit: 8,
                            maxRotation: 0,
                        }
                    },
                    y: { beginAtZero: false }
                }
            }
        })
}

// ============================================================
// RENDER TABLE — Tabel data historis
// ============================================================

function renderTable() {
    const tbody = document.getElementById('history-tbody')
    if (!tbody || !historyData) return

        // Tampilkan 50 data terakhir
        const recent = historyData.slice(-50).reverse()

        tbody.innerHTML = recent.map(d => `
        <tr>
        <td>${d.timestamp}</td>
        <td>${d.ph.toFixed(2)}</td>
        <td>${d.do.toFixed(2)}</td>
        <td>${d.temperature.toFixed(1)}</td>
        <td>${d.salinity.toFixed(1)}</td>
        <td>${d.turbidity.toFixed(1)}</td>
        <td>${d.water_level.toFixed(2)}</td>
        </tr>
        `).join('')
}

// ============================================================
// KONTROL MANUAL — Kirim perintah relay
// ============================================================

async function controlRelay(actuator, state) {
    try {
        const res = await fetch('/api/control', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ actuator, state, mode: 'MANUAL' })
        })
        if (!res.ok) throw new Error('Gagal kontrol relay')
            await fetchRelay()
            await fetchMode()
    } catch (err) {
        alert('Error: ' + err.message)
    }
}

async function setMode(actuator, mode) {
    try {
        const res = await fetch('/api/control', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ actuator, mode, state: false })
        })
        if (!res.ok) throw new Error('Gagal set mode')
            await fetchMode()
    } catch (err) {
        alert('Error: ' + err.message)
    }
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

function setValue(id, val) {
    const el = document.getElementById(id)
    if (el) el.textContent = val
}

function updateSensorCard(cardId, badgeId, value, statusFn) {
    const card  = document.getElementById(cardId)
    const badge = document.getElementById(badgeId)
    if (!card || !badge) return

        const status = statusFn(value)

        card.className = 'sensor-card status-' + (status === 'ok' ? 'ok' : status === 'warn' ? 'warn' : 'crit')

        badge.className = 'sensor-badge badge-' + status
        badge.textContent = status === 'ok' ? '✓ Normal' : status === 'warn' ? '⚠ Perhatian' : '✗ Kritis'
}

function updateActuator(id, isOn) {
    const el = document.getElementById(id)
    if (!el) return
        el.className = 'actuator-status ' + (isOn ? 'status-on' : 'status-off')
        el.textContent = isOn ? 'ON' : 'OFF'
}

function paramLabel(param) {
    const labels = {
        ph: 'pH Air',
        do: 'Oksigen Terlarut (mg/L)',
            temperature: 'Suhu (°C)',
            salinity: 'Salinitas (ppt)',
            turbidity: 'Turbiditas (NTU)',
            water_level: 'Ketinggian Air (m)',
    }
    return labels[param] || param
}

// ============================================================
// INISIALISASI
// ============================================================

document.addEventListener('DOMContentLoaded', () => {
    // Poll pertama langsung
    pollData()

    // Poll setiap 5 detik
    setInterval(pollData, POLL_INTERVAL)
})
