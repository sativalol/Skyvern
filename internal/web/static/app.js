// XSS Prevention / HTML Sanitizer
function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Logo typing animation placeholders if needed
const logoText = document.getElementById('logoText');

// SPA Routing
function showPage(name) {
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    const pg = document.getElementById('page-' + name);
    if (pg) pg.classList.add('active');
    
    // Smooth scroll to top
    window.scrollTo({ top: 0, behavior: 'smooth' });
    
    // Close mobile drawers
    const mobileMenu = document.getElementById('mobileMenu');
    if (mobileMenu) mobileMenu.classList.remove('active');
    document.body.style.overflow = 'auto';

    // Start/Stop dashboard sync depending on page focus
    if (name === 'skyvern') {
        startDashboardSync();
    } else {
        stopDashboardSync();
    }
}

function navigate(name) { 
    showPage(name); 
}

// URL Routing Mapping
const hashMap = { 
    auth: 'auth', 
    esoterica: 'bot', 
    script: 'script', 
    bot: 'bot',
    skyvern: 'skyvern'
};

const path = window.location.pathname.replace('/', '');
if (hashMap[path]) {
    showPage(hashMap[path]);
}

// Mobile Menu Event Listeners
const mobileMenu = document.getElementById('mobileMenu');
const burgerBtn = document.getElementById('burgerBtn');
const closeMenuBtn = document.getElementById('closeMenuBtn');
const mobileBackdrop = document.getElementById('mobileBackdrop');

if (burgerBtn && mobileMenu) {
    burgerBtn.onclick = () => { 
        mobileMenu.classList.add('active'); 
        document.body.style.overflow = 'hidden'; 
    };
}
if (closeMenuBtn && mobileMenu) {
    closeMenuBtn.onclick = () => { 
        mobileMenu.classList.remove('active'); 
        document.body.style.overflow = 'auto'; 
    };
}
if (mobileBackdrop && mobileMenu) {
    mobileBackdrop.onclick = () => { 
        mobileMenu.classList.remove('active'); 
        document.body.style.overflow = 'auto'; 
    };
}

// Interactive Spotlight Gradients
document.addEventListener('mousemove', e => {
    document.querySelectorAll('.home-card, .feature-card, .score-card, .game-card, .pricing-card-alt').forEach(card => {
        const rect = card.getBoundingClientRect();
        card.style.setProperty('--mouse-x', (e.clientX - rect.left) + 'px');
        card.style.setProperty('--mouse-y', (e.clientY - rect.top) + 'px');
    });
});

// Scroll Reveal
const obs = new IntersectionObserver(entries => {
    entries.forEach(e => { 
        if (e.isIntersecting) { 
            e.target.style.opacity = '1'; 
            e.target.style.transform = 'translateY(0)'; 
        } 
    });
}, { threshold: 0.1, rootMargin: '0px 0px -60px 0px' });

document.querySelectorAll('.section-title, .section-desc, .feature-card, .score-card, .game-card, .pricing-card-main, .pricing-card-alt, .ecosystem-card').forEach(el => {
    el.style.opacity = '0'; 
    el.style.transform = 'translateY(25px)'; 
    el.style.transition = 'opacity 0.7s ease, transform 0.7s ease';
    obs.observe(el);
});

// Navbar Scroll Styling Tint
window.addEventListener('scroll', () => {
    const navbar = document.getElementById('navbar');
    if (navbar) {
        navbar.style.background = window.pageYOffset > 50 ? 'rgba(10,10,10,0.95)' : 'rgba(10,10,10,0.6)';
    }
});


/* SKYVERN DASHBOARD MANAGER MODULE */
let allBots = [];
let dashboardInterval = null;

async function fetchStats() {
    try {
        const res = await fetch('/api/stats');
        if (!res.ok) return;
        const data = await res.json();
        
        const activeSpan = document.getElementById('stat-active');
        const cmdsSpan = document.getElementById('stat-cmds');
        const uptimeSpan = document.getElementById('stat-uptime');
        const progressBar = document.getElementById('active-progress');

        if (activeSpan) activeSpan.innerText = `${data.active_bots} / ${data.total_bots}`;
        if (cmdsSpan) cmdsSpan.innerText = Number(data.total_commands).toLocaleString();
        
        let pct = 0;
        if (data.total_bots > 0) {
            pct = (data.active_bots / data.total_bots) * 100;
        }
        if (progressBar) progressBar.style.width = `${pct}%`;
        
        const sec = data.uptime_seconds;
        const h = Math.floor(sec / 3600);
        const m = Math.floor((sec % 3600) / 60);
        const s = sec % 60;
        if (uptimeSpan) {
            uptimeSpan.innerText = (h > 0 ? `${h}h ` : '') + (m > 0 ? `${m}m ` : '') + `${s}s`;
        }
    } catch (err) {
        console.error("Dashboard stats sync error:", err);
    }
}

async function fetchBots() {
    try {
        const res = await fetch('/api/bots');
        if (!res.ok) return;
        allBots = await res.json();
        renderBots();
    } catch (err) {
        console.error("Dashboard list fetch error:", err);
    }
}

function renderBots() {
    const searchInput = document.getElementById('bot-search');
    const query = searchInput ? searchInput.value.toLowerCase() : '';
    
    const filtered = allBots.filter(bot => {
        const name = (bot.custom_name || 'Unnamed Bot').toLowerCase();
        const id = bot.client_id.toLowerCase();
        return name.includes(query) || id.includes(query);
    });

    const botCountSpan = document.getElementById('bot-count');
    if (botCountSpan) {
        botCountSpan.innerText = `${filtered.length} bots`;
    }

    const container = document.getElementById('bot-grid');
    if (!container) return;

    container.innerHTML = '';

    if (filtered.length === 0) {
        container.innerHTML = `
            <div class="grid-loading">
                <span>No bots found.</span>
            </div>
        `;
        return;
    }

    filtered.forEach(bot => {
        const card = document.createElement('div');
        card.className = 'bot-panel';

        let statusClass = 'stopped';
        let statusLabel = 'Stopped';
        if (bot.running) {
            statusClass = 'running';
            statusLabel = 'Active';
        } else if (bot.last_err) {
            statusClass = 'error';
            statusLabel = 'Failed';
        }

        const rawName = bot.custom_name ? bot.custom_name : 'Unnamed Bot';
        const name = escapeHtml(rawName);
        const clientId = escapeHtml(bot.client_id);
        const lastErr = escapeHtml(bot.last_err);

        const btnClass = bot.running ? 'stop' : 'start';
        const btnText = bot.running ? 'Stop' : 'Start';

        card.innerHTML = `
            <div class="panel-top">
                <div class="panel-meta">
                    <span class="panel-dot ${statusClass}"></span>
                    <div class="panel-title-block">
                        <span class="panel-name">${name}</span>
                        <span class="panel-id">ID: ${clientId}</span>
                    </div>
                </div>
                <span class="panel-pill ${statusClass}">${statusLabel}</span>
            </div>
            ${lastErr ? `<div class="panel-error">${lastErr}</div>` : ''}
            <div class="panel-bottom">
                <button class="btn-ctrl ${btnClass}" id="btn-${clientId}" onclick="toggleBot('${clientId}', ${bot.running})">
                    ${btnText}
                </button>
            </div>
        `;
        container.appendChild(card);
    });
}

async function toggleBot(id, running) {
    const btn = document.getElementById(`btn-${id}`);
    if (btn) {
        btn.disabled = true;
        btn.innerText = running ? 'Stopping...' : 'Starting...';
    }

    const endpoint = running ? '/api/bot/stop' : '/api/bot/start';
    try {
        const res = await fetch(`${endpoint}?id=${encodeURIComponent(id)}`);
        if (res.ok) {
            await fetchBots();
            await fetchStats();
        } else {
            alert("Action failed to execute on server.");
            if (btn) btn.disabled = false;
        }
    } catch (err) {
        alert("Network request failed: " + err);
        if (btn) btn.disabled = false;
    }
}

// Filter listener on search box
const searchBox = document.getElementById('bot-search');
if (searchBox) {
    searchBox.addEventListener('input', renderBots);
}

// Background sync control
function startDashboardSync() {
    if (dashboardInterval) return;
    fetchStats();
    fetchBots();
    dashboardInterval = setInterval(() => {
        fetchStats();
        fetchBots();
    }, 3000);
}

function stopDashboardSync() {
    if (dashboardInterval) {
        clearInterval(dashboardInterval);
        dashboardInterval = null;
    }
}
