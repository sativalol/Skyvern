async function fetchStats() {
    try {
        const res = await fetch('/api/stats');
        const data = await res.json();
        document.getElementById('stat-active').innerText = data.active_bots + ' / ' + data.total_bots;
        document.getElementById('stat-cmds').innerText = data.total_commands;
        
        const sec = data.uptime_seconds;
        const h = Math.floor(sec / 3600);
        const m = Math.floor((sec % 3600) / 60);
        const s = sec % 60;
        document.getElementById('stat-uptime').innerText = 
            (h > 0 ? h + 'h ' : '') + (m > 0 ? m + 'm ' : '') + s + 's';
    } catch (err) {
        console.error("stats fetch failed", err);
    }
}

async function fetchBots() {
    try {
        const res = await fetch('/api/bots');
        const bots = await res.json();
        document.getElementById('bot-count').innerText = bots.length + ' loaded';
        
        const container = document.getElementById('bot-list');
        container.innerHTML = '';
        
        bots.forEach(bot => {
            const row = document.createElement('div');
            row.className = 'bot-row';
            
            let statusClass = 'stopped';
            if (bot.running) {
                statusClass = 'running';
            } else if (bot.last_err) {
                statusClass = 'error';
            }
            
            const name = bot.custom_name ? bot.custom_name : 'Unnamed Bot';
            const btnClass = bot.running ? 'stop' : 'start';
            const btnText = bot.running ? 'Stop Bot' : 'Start Bot';
            
            const errHtml = bot.last_err ? '<span class="bot-error">Error: ' + bot.last_err + '</span>' : '';
            
            row.innerHTML = 
                '<div class="bot-info">' +
                    '<span class="status-dot ' + statusClass + '"></span>' +
                    '<div class="bot-details">' +
                        '<span class="bot-name">' + name + '</span>' +
                        '<span class="bot-id">Client ID: ' + bot.client_id + '</span>' +
                        errHtml +
                    '</div>' +
                '</div>' +
                '<div>' +
                    '<button class="btn ' + btnClass + '" onclick="toggleBot(\'' + bot.client_id + '\', ' + bot.running + ')">' +
                        btnText +
                    '</button>' +
                '</div>';
            container.appendChild(row);
        });
    } catch (err) {
        console.error("bots fetch failed", err);
    }
}

async function toggleBot(id, running) {
    const endpoint = running ? '/api/bot/stop' : '/api/bot/start';
    try {
        const res = await fetch(endpoint + '?id=' + id);
        if (res.ok) {
            fetchBots();
            fetchStats();
        } else {
            alert("Action failed!");
        }
    } catch (err) {
        alert("Request error: " + err);
    }
}

setInterval(() => {
    fetchStats();
    fetchBots();
}, 3000);

fetchStats();
fetchBots();
