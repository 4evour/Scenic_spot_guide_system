const API_BASE = '/api/v1';

function escapeHtml(value) {
    return String(value ?? '').replace(/[&<>"']/g, char => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[char]));
}

function authHeaders() {
    const token = localStorage.getItem('authToken');
    return token ? { Authorization: `Bearer ${token}` } : {};
}

const mockDashboard = {
    overview: {
        total_visitors: 12356,
        total_chats: 8923,
        satisfaction_rate: '96.8%',
        avg_response_time: '2.1s',
        online_users: 2345
    },
    hourly: [420, 360, 310, 280, 390, 560, 920, 1360, 1880, 2140, 2360, 2510, 2420, 2300, 2260, 2180, 1980, 1760, 1430, 1180, 920, 760, 620, 510],
    questions: [
        ['灵山大佛有多高？', 1234],
        ['推荐一条历史文化路线', 1135],
        ['九龙灌浴几点开始？', 1047],
        ['梵宫适合游览多久？', 968],
        ['停车场怎么走？', 889]
    ],
    categories: [
        ['景点特色', 42, '#52f0ee'],
        ['历史文化', 24, '#f4c765'],
        ['路线推荐', 18, '#7ef2a0'],
        ['服务咨询', 16, '#8aa4ff']
    ],
    response: [
        ['<1s', 42],
        ['1-3s', 38],
        ['3-5s', 15],
        ['>5s', 5]
    ],
    conversations: [
        ['游客', '灵山大佛为什么建在这里？', '刚刚'],
        ['小灵', '这里依山面湖，兼具礼佛朝圣与江南山水意境，适合作为文化地标展开讲解。', '2.1s'],
        ['游客', '带老人游览有什么路线？', '2分钟前'],
        ['小灵', '建议选择游客中心、九龙灌浴、梵宫、商业街的轻松路线，减少长距离爬坡。', '1.9s']
    ]
};

function updateCurrentTime() {
    const now = new Date();
    const time = now.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
    document.getElementById('currentTime').textContent = time;
    document.getElementById('updateTime').textContent = time;
}

async function loadDashboardData() {
    let overview = mockDashboard.overview;
    try {
        const resp = await fetch(`${API_BASE}/admin/dashboard/overview`, {
            headers: authHeaders()
        });
        const data = await resp.json();
        if (data && data.code === 0 && data.data) overview = { ...overview, ...data.data };
    } catch (error) {}
    updateOverview(overview);
    renderTrend(mockDashboard.hourly);
    renderTopQuestions(mockDashboard.questions);
    renderCategoryDonut(mockDashboard.categories);
    renderResponseChart(mockDashboard.response);
    renderConversations(mockDashboard.conversations);
}

function updateOverview(data) {
    document.getElementById('totalVisitors').textContent = Number(data.total_visitors || 0).toLocaleString('zh-CN');
    document.getElementById('totalChats').textContent = Number(data.total_chats || 0).toLocaleString('zh-CN');
    document.getElementById('satisfactionRate').textContent = data.satisfaction_rate || '96.8%';
    document.getElementById('avgResponseTime').textContent = data.avg_response_time || '2.1s';
    document.getElementById('onlineUsers').textContent = Number(data.online_users || 2345).toLocaleString('zh-CN');
}

function renderTrend(values) {
    const svg = document.getElementById('trendChart');
    const max = Math.max(...values);
    const width = 900;
    const height = 320;
    const pad = 26;
    const points = values.map((value, index) => {
        const x = (index / (values.length - 1)) * width;
        const y = height - pad - (value / max) * (height - pad * 2);
        return [x, y];
    });
    const line = points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point[0].toFixed(1)} ${point[1].toFixed(1)}`).join(' ');
    const area = `${line} L ${width} ${height - pad} L 0 ${height - pad} Z`;
    svg.innerHTML = `
        <defs>
            <linearGradient id="trendFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="#52f0ee" stop-opacity="0.35"/>
                <stop offset="100%" stop-color="#52f0ee" stop-opacity="0"/>
            </linearGradient>
        </defs>
        <path d="${area}" fill="url(#trendFill)"></path>
        <path d="${line}" fill="none" stroke="#52f0ee" stroke-width="4"></path>
        ${points.filter((_, i) => i % 4 === 0).map(p => `<circle cx="${p[0]}" cy="${p[1]}" r="5" fill="#f4c765"></circle>`).join('')}
    `;
}

function renderTopQuestions(items) {
    const max = Math.max(...items.map(item => item[1]));
    document.getElementById('topQuestions').innerHTML = items.map(([label, value]) => `
        <div class="h-bar-item">
            <span class="h-bar-label">${escapeHtml(label)}</span>
            <div class="h-bar-wrapper"><div class="h-bar" style="width:${Math.round(value / max * 100)}%"></div></div>
            <strong>${value.toLocaleString('zh-CN')}</strong>
        </div>
    `).join('');
}

function renderCategoryDonut(items) {
    const svg = document.getElementById('categoryDonut');
    const radius = 82;
    const circumference = Math.PI * 2 * radius;
    let offset = 0;
    svg.innerHTML = items.map(([label, percent, color]) => {
        const dash = circumference * percent / 100;
        const circle = `<circle cx="110" cy="110" r="${radius}" fill="none" stroke="${color}" stroke-width="24" stroke-dasharray="${dash} ${circumference - dash}" stroke-dashoffset="${-offset}" transform="rotate(-90 110 110)"></circle>`;
        offset += dash;
        return circle;
    }).join('') + '<circle cx="110" cy="110" r="54" fill="rgba(5,17,20,.96)"></circle>';
    document.getElementById('categoryLegend').innerHTML = items.map(([label, percent, color]) => `
        <div class="legend-item"><span class="legend-color" style="background:${escapeHtml(color)}"></span><span>${escapeHtml(label)}</span><strong>${escapeHtml(percent)}%</strong></div>
    `).join('');
}

function renderResponseChart(items) {
    document.getElementById('responseChart').innerHTML = items.map(([label, value]) => `
        <div class="response-bar" style="height:${Math.max(value, 8)}%">
            <span>${escapeHtml(value)}%</span>
            <small>${escapeHtml(label)}</small>
        </div>
    `).join('');
}

function renderConversations(items) {
    document.getElementById('recentConversations').innerHTML = items.map(([name, message, time]) => `
        <div class="chat-item ${name === '小灵' ? 'ai' : ''}">
            <div class="chat-avatar">${name === '小灵' ? 'AI' : '客'}</div>
            <div class="chat-content">
                <span class="chat-name">${escapeHtml(name)}</span>
                <span class="chat-text">${escapeHtml(message)}</span>
                <span class="chat-time">${escapeHtml(time)}</span>
            </div>
        </div>
    `).join('');
}

function tickMockNumbers() {
    mockDashboard.overview.total_visitors += Math.floor(Math.random() * 9) + 1;
    mockDashboard.overview.total_chats += Math.floor(Math.random() * 5) + 1;
    mockDashboard.overview.online_users += Math.floor(Math.random() * 13) - 5;
    updateOverview(mockDashboard.overview);
}

document.addEventListener('DOMContentLoaded', () => {
    updateCurrentTime();
    setInterval(updateCurrentTime, 1000);
    loadDashboardData();
    setInterval(tickMockNumbers, 2500);
    document.querySelectorAll('.date-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.date-btn').forEach(item => item.classList.remove('active'));
            btn.classList.add('active');
        });
    });
});
