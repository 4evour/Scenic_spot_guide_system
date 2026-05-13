let currentUser = null;
let isSpeaking = false;
let recognition = null;
let map = null;
const DIGITAL_HUMAN_URL = '/digital-human#/digital-human';

function escapeHtml(value) {
    return String(value ?? '').replace(/[&<>"']/g, char => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[char]));
}

function escapeJsString(value) {
    return String(value ?? '')
        .replace(/\\/g, '\\\\')
        .replace(/'/g, "\\'")
        .replace(/\r?\n/g, ' ');
}

const digitalHuman = {
    name: '灵灵',
    status: 'idle',
    emotion: 'friendly',
    avatar: null,
    video: null,
    isActive: false
};

function openDigitalHuman() {
    window.location.href = DIGITAL_HUMAN_URL;
}

function goToSection(sectionId) {
    if (sectionId === 'chatBtn') {
        openDigitalHuman();
        return;
    }

    const sectionMap = {
        'homeBtn': 'homeSection',
        'spotsBtn': 'spotsSection',
        'routesBtn': 'routesSection',
        'profileBtn': 'profileSection'
    };

    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.section').forEach(section => section.classList.remove('active'));

    const targetBtn = document.getElementById(sectionId);
    if (targetBtn) {
        targetBtn.classList.add('active');
    }

    const targetSection = document.getElementById(sectionMap[sectionId]);
    if (targetSection) {
        targetSection.classList.add('active');
    }

    if (sectionId === 'spotsBtn') {
        loadSpots();
    } else if (sectionId === 'routesBtn') {
        loadRoutes();
    }
}

function initNavigation() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', function() {
            goToSection(this.id);
        });
    });
}

function initDigitalHuman() {
    const avatarContainer = document.getElementById('digitalHumanContainer');
    if (!avatarContainer) {
        digitalHuman.avatar = null;
        return;
    }

    digitalHuman.avatar = avatarContainer;

    const savedSettings = localStorage.getItem('digitalHumanSettings');
    if (savedSettings) {
        const settings = JSON.parse(savedSettings);
        digitalHuman.name = settings.name || '灵灵';
    }
}

function toggleDigitalHuman() {
    const container = document.getElementById('digitalHumanContainer');
    if (!container) return;

    digitalHuman.isActive = !digitalHuman.isActive;

    if (digitalHuman.isActive) {
        container.classList.add('active');
        showDigitalHumanMessage('您好！我是您的AI数字人导览员，有什么可以帮您的吗？');
    } else {
        container.classList.remove('active');
    }
}

function showDigitalHumanMessage(message) {
    const messageEl = document.getElementById('digitalHumanMessage');
    if (messageEl) {
        messageEl.textContent = message;
        messageEl.classList.add('show');

        if (document.getElementById('autoSpeak').checked) {
            speakText(message);
        }
    }
}

function updateDigitalHumanEmotion(emotion) {
    digitalHuman.emotion = emotion;
    const avatarEl = document.querySelector('.digital-human-avatar');
    if (avatarEl) {
        avatarEl.setAttribute('data-emotion', emotion);
    }
}

function setDigitalHumanName(name) {
    digitalHuman.name = name;
    const nameEl = document.getElementById('digitalHumanName');
    if (nameEl) {
        nameEl.textContent = name;
    }
}

function loadSpots() {
    const spotsGrid = document.getElementById('spotsGrid');
    if (!spotsGrid || spotsGrid.children.length > 0) return;

    const spots = [
        { name: '灵山大佛', desc: '世界最高的佛立像，高达88米', category: '人文', rating: 4.9 },
        { name: '灵山梵宫', desc: '佛教艺术殿堂，建筑精美', category: '人文', rating: 4.8 },
        { name: '九龙灌浴', desc: '大型动态景观表演', category: '娱乐', rating: 4.7 },
        { name: '梵天花海', desc: '四季花海，色彩斑斓', category: '自然', rating: 4.6 },
        { name: '祥符禅寺', desc: '千年古寺，香火鼎盛', category: '人文', rating: 4.8 },
        { name: '五明桥', desc: '佛教五明智慧象征', category: '人文', rating: 4.5 }
    ];

    spotsGrid.innerHTML = spots.map(spot => `
        <div class="spot-card" onclick="showSpotDetail('${escapeJsString(spot.name)}')">
            <div class="spot-image">🗿</div>
            <div class="spot-info">
                <h3>${escapeHtml(spot.name)}</h3>
                <p>${escapeHtml(spot.desc)}</p>
                <div class="spot-meta">
                    <span class="spot-category">${escapeHtml(spot.category)}</span>
                    <span class="spot-rating">⭐ ${escapeHtml(spot.rating)}</span>
                </div>
            </div>
        </div>
    `).join('');
}

function filterSpots(category) {
    const spotsGrid = document.getElementById('spotsGrid');
    if (!spotsGrid) return;

    const spots = [
        { name: '灵山大佛', desc: '世界最高的佛立像，高达88米', category: '人文', rating: 4.9 },
        { name: '灵山梵宫', desc: '佛教艺术殿堂，建筑精美', category: '人文', rating: 4.8 },
        { name: '九龙灌浴', desc: '大型动态景观表演', category: '娱乐', rating: 4.7 },
        { name: '梵天花海', desc: '四季花海，色彩斑斓', category: '自然', rating: 4.6 },
        { name: '祥符禅寺', desc: '千年古寺，香火鼎盛', category: '人文', rating: 4.8 },
        { name: '五明桥', desc: '佛教五明智慧象征', category: '人文', rating: 4.5 }
    ];

    const filteredSpots = category === 'all' ? spots : spots.filter(s => s.category === category);

    spotsGrid.innerHTML = filteredSpots.map(spot => `
        <div class="spot-card" onclick="showSpotDetail('${escapeJsString(spot.name)}')">
            <div class="spot-image">🗿</div>
            <div class="spot-info">
                <h3>${escapeHtml(spot.name)}</h3>
                <p>${escapeHtml(spot.desc)}</p>
                <div class="spot-meta">
                    <span class="spot-category">${escapeHtml(spot.category)}</span>
                    <span class="spot-rating">⭐ ${escapeHtml(spot.rating)}</span>
                </div>
            </div>
        </div>
    `).join('');
}

function showSpotDetail(spotName) {
    openDigitalHuman();
}

function loadRoutes() {
    const routesList = document.getElementById('routesList');
    if (!routesList || routesList.children.length > 0) return;

    const routeData = [
        { name: '经典文化之旅', desc: '从景区入口出发，依次参观灵山大照壁、五明桥等核心景点', duration: '约2.5小时', difficulty: 'medium', spots: 5 },
        { name: '深度历史之旅', desc: '探索灵山胜境千年历史文化，了解佛教传承与艺术精髓', duration: '约1.5小时', difficulty: 'easy', spots: 4 },
        { name: '自然风光之旅', desc: '欣赏灵山胜境自然风光，感受大自然与人文景观的完美融合', duration: '约2小时', difficulty: 'easy', spots: 5 },
        { name: '亲子欢乐之旅', desc: '适合家庭出游，体验互动项目，留下美好回忆', duration: '约2小时', difficulty: 'easy', spots: 5 }
    ];

    routesList.innerHTML = routeData.map(route => `
        <div class="route-card" onclick="selectRoute('${escapeJsString(route.name)}')">
            <div class="route-info">
                <h3>${escapeHtml(route.name)}</h3>
                <p>${escapeHtml(route.desc)}</p>
                <div class="route-details">
                    <div class="route-detail">
                        <i class="fas fa-clock"></i>
                        <span>${escapeHtml(route.duration)}</span>
                    </div>
                    <div class="route-detail">
                        <i class="fas fa-map-marker-alt"></i>
                        <span>${escapeHtml(route.spots)}个景点</span>
                    </div>
                    <span class="route-difficulty ${route.difficulty}">${route.difficulty === 'easy' ? '轻松' : route.difficulty === 'medium' ? '适中' : '挑战'}</span>
                </div>
            </div>
        </div>
    `).join('');
}

function filterRoutes(difficulty) {
    const routesList = document.getElementById('routesList');
    if (!routesList) return;

    const routeData = [
        { name: '经典文化之旅', desc: '从景区入口出发，依次参观灵山大照壁、五明桥等核心景点', duration: '约2.5小时', difficulty: 'medium', spots: 5 },
        { name: '深度历史之旅', desc: '探索灵山胜境千年历史文化，了解佛教传承与艺术精髓', duration: '约1.5小时', difficulty: 'easy', spots: 4 },
        { name: '自然风光之旅', desc: '欣赏灵山胜境自然风光，感受大自然与人文景观的完美融合', duration: '约2小时', difficulty: 'easy', spots: 5 },
        { name: '亲子欢乐之旅', desc: '适合家庭出游，体验互动项目，留下美好回忆', duration: '约2小时', difficulty: 'easy', spots: 5 }
    ];

    const filteredRoutes = difficulty === 'all' ? routeData : routeData.filter(r => r.difficulty === difficulty);

    routesList.innerHTML = filteredRoutes.map(route => `
        <div class="route-card" onclick="selectRoute('${escapeJsString(route.name)}')">
            <div class="route-info">
                <h3>${escapeHtml(route.name)}</h3>
                <p>${escapeHtml(route.desc)}</p>
                <div class="route-details">
                    <div class="route-detail">
                        <i class="fas fa-clock"></i>
                        <span>${escapeHtml(route.duration)}</span>
                    </div>
                    <div class="route-detail">
                        <i class="fas fa-map-marker-alt"></i>
                        <span>${escapeHtml(route.spots)}个景点</span>
                    </div>
                    <span class="route-difficulty ${route.difficulty}">${route.difficulty === 'easy' ? '轻松' : route.difficulty === 'medium' ? '适中' : '挑战'}</span>
                </div>
            </div>
        </div>
    `).join('');
}

function selectRoute(routeName) {
    openDigitalHuman();
}

let autoSpeakEnabled = true;

function toggleSpeak() {
    autoSpeakEnabled = !autoSpeakEnabled;
    const speakBtn = document.querySelector('.dh-speak-btn');
    if (speakBtn) {
        if (autoSpeakEnabled) {
            speakBtn.classList.add('active');
            speakBtn.style.background = 'var(--primary-gradient)';
        } else {
            speakBtn.classList.remove('active');
            speakBtn.style.background = '';
        }
    }
    const autoSpeakCheckbox = document.getElementById('autoSpeak');
    if (autoSpeakCheckbox) {
        autoSpeakCheckbox.checked = autoSpeakEnabled;
    }
}

let emotionIndex = 0;
const emotions = ['friendly', 'happy', 'thinking', 'speaking'];

function cycleEmotion() {
    emotionIndex = (emotionIndex + 1) % emotions.length;
    const emotion = emotions[emotionIndex];
    updateDigitalHumanEmotion(emotion);

    const messages = {
        'friendly': '您好！有什么可以帮助您的吗？',
        'happy': '太棒了！让我们一起探索景区吧！',
        'thinking': '让我想想...这个问题很有意思',
        'speaking': '正在为您讲解，请仔细听哦'
    };

    showDigitalHumanMessage(messages[emotion]);
}

const routes = {
    classic: {
        name: '经典文化之旅',
        duration: '约2.5小时',
        description: '从景区入口出发，依次参观灵山大照壁、五明桥、佛足坛、五智门，最后到达灵山大佛。',
        steps: [
            { name: '灵山胜境入口', desc: '景区入口，领取导览地图', time: '约10分钟', distance: '0.0km' },
            { name: '灵山大照壁', desc: '华夏第一壁，气势恢宏', time: '约15分钟', distance: '0.3km' },
            { name: '五明桥', desc: '佛教智慧象征', time: '约10分钟', distance: '0.2km' },
            { name: '佛足坛', desc: '祈福朝圣之地', time: '约15分钟', distance: '0.4km' },
            { name: '灵山大佛', desc: '世界最高佛立像', time: '约30分钟', distance: '0.5km' }
        ]
    },
    history: {
        name: '深度历史之旅',
        duration: '约1.5小时',
        description: '探索灵山胜境千年历史文化，了解佛教传承与艺术精髓。',
        steps: [
            { name: '无尽意斋', desc: '佛教文化展示', time: '约15分钟', distance: '0.0km' },
            { name: '降魔浮雕', desc: '佛教故事雕刻', time: '约20分钟', distance: '0.3km' },
            { name: '阿育王柱', desc: '历史遗迹', time: '约15分钟', distance: '0.2km' },
            { name: '祥符禅寺', desc: '千年古寺', time: '约30分钟', distance: '0.4km' }
        ]
    },
    nature: {
        name: '自然风光之旅',
        duration: '约2小时',
        description: '欣赏灵山胜境自然风光，感受大自然与人文景观的完美融合。',
        steps: [
            { name: '梵天花海', desc: '四季花海', time: '约20分钟', distance: '0.0km' },
            { name: '曼飞龙塔', desc: '傣式佛塔', time: '约15分钟', distance: '0.3km' },
            { name: '灵山梵宫', desc: '艺术殿堂', time: '约30分钟', distance: '0.5km' },
            { name: '拈花广场', desc: '禅意空间', time: '约20分钟', distance: '0.3km' },
            { name: '香月花街', desc: '文化街区', time: '约15分钟', distance: '0.2km' }
        ]
    },
    family: {
        name: '亲子欢乐之旅',
        duration: '约2小时',
        description: '适合家庭出游，体验互动项目，留下美好回忆。',
        steps: [
            { name: '九龙灌浴', desc: '震撼表演', time: '约20分钟', distance: '0.0km' },
            { name: '百子戏弥勒', desc: '趣味雕塑', time: '约15分钟', distance: '0.2km' },
            { name: '祥符禅寺', desc: '祈福许愿', time: '约20分钟', distance: '0.3km' },
            { name: '佛教文化博览馆', desc: '互动体验', time: '约30分钟', distance: '0.4km' },
            { name: '儿童乐园', desc: '亲子互动', time: '约15分钟', distance: '0.2km' }
        ]
    }
};

function openModal(type) {
    const modal = document.getElementById('loginModal');
    modal.classList.add('show');

    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.form-container').forEach(form => form.classList.add('hidden'));

    if (type === 'login') {
        document.querySelector('.tab-btn:nth-child(1)').classList.add('active');
        document.getElementById('loginForm').classList.remove('hidden');
    } else if (type === 'register') {
        document.querySelector('.tab-btn:nth-child(2)').classList.add('active');
        document.getElementById('registerForm').classList.remove('hidden');
    } else {
        document.querySelector('.tab-btn:nth-child(3)').classList.add('active');
        document.getElementById('adminForm').classList.remove('hidden');
    }
}

function closeModal() {
    const modal = document.getElementById('loginModal');
    modal.classList.remove('show');
}

function showTab(tabName) {
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));

    const tabBtn = document.querySelector(`.tab-btn[onclick="showTab('${tabName}')"]`);
    if (tabBtn) {
        tabBtn.classList.add('active');
    }

    const tabContent = document.getElementById(tabName);
    if (tabContent) {
        tabContent.classList.add('active');
    }
}

function switchTab(tab) {
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.form-container').forEach(form => form.classList.add('hidden'));

    event.target.classList.add('active');
    document.getElementById(`${tab}Form`).classList.remove('hidden');
}

async function login() {
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    if (!username || !password) {
        alert('请输入用户名和密码');
        return;
    }

    try {
        const response = await fetch('/api/v1/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        const result = await response.json();
        if (!response.ok || result.code !== 0) {
            throw new Error(result.message || '登录失败');
        }
        currentUser = result.data;
        localStorage.setItem('authToken', result.data.token);
        localStorage.setItem('currentUser', JSON.stringify(result.data));
        updateUserInterface();
        closeModal();
        alert(`欢迎, ${result.data.username}!`);
    } catch (error) {
        alert(error.message || '登录失败');
    }
}

async function register() {
    const username = document.getElementById('regUsername').value;
    const password = document.getElementById('regPassword').value;

    if (!username || !password) {
        alert('请输入用户名和密码');
        return;
    }

    try {
        const response = await fetch('/api/v1/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        const result = await response.json();
        if (!response.ok || result.code !== 0) {
            throw new Error(result.message || '注册失败');
        }
        closeModal();
        alert(`注册成功，请使用 ${username} 登录`);
    } catch (error) {
        alert(error.message || '注册失败');
    }
}

async function adminLogin() {
    const username = document.getElementById('adminUsername').value;
    const password = document.getElementById('adminPassword').value;

    try {
        const response = await fetch('/api/v1/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        const result = await response.json();
        if (!response.ok || result.code !== 0) {
            throw new Error(result.message || '管理员账号或密码错误');
        }
        if (result.data?.role !== 'admin') {
            throw new Error('当前账号没有管理员权限');
        }
        localStorage.setItem('authToken', result.data.token);
        localStorage.setItem('currentUser', JSON.stringify(result.data));
        window.location.href = '/admin';
    } catch (error) {
        alert(error.message || '管理员账号或密码错误');
    }
}

function logout() {
    currentUser = null;
    localStorage.removeItem('authToken');
    localStorage.removeItem('currentUser');
    updateUserInterface();
    alert('已退出登录');
}

function openDashboard() {
    window.location.href = '/dashboard';
}

function updateUserInterface() {
    const storedUser = currentUser || (() => {
        try {
            return JSON.parse(localStorage.getItem('currentUser') || 'null');
        } catch {
            return null;
        }
    })();

    currentUser = storedUser;

    const userName = document.getElementById('userName');
    const loginBtn = document.getElementById('loginBtn');
    const logoutBtn = document.getElementById('logoutBtn');
    const adminBtn = document.getElementById('adminBtn');
    const profileName = document.getElementById('profileName');
    const profileEmail = document.getElementById('profileEmail');

    if (storedUser) {
        if (userName) userName.textContent = storedUser.username || '已登录用户';
        if (loginBtn) loginBtn.style.display = 'none';
        if (logoutBtn) logoutBtn.style.display = 'inline-block';
        if (profileName) profileName.textContent = storedUser.username || '已登录用户';
        if (profileEmail) profileEmail.textContent = storedUser.role === 'admin' ? '管理员账号' : '游客账号';
        if (adminBtn) adminBtn.style.display = storedUser.role === 'admin' ? 'block' : 'none';
    } else {
        if (userName) userName.textContent = '游客';
        if (loginBtn) loginBtn.style.display = 'inline-block';
        if (logoutBtn) logoutBtn.style.display = 'none';
        if (adminBtn) adminBtn.style.display = 'none';
        if (profileName) profileName.textContent = '未登录用户';
        if (profileEmail) profileEmail.textContent = '请登录以查看个人信息';
    }
}

function openLoginFromHash() {
    if (window.location.hash !== '#login') return;
    const modal = document.getElementById('loginModal');
    if (modal) modal.classList.add('show');
}

function sendMessage(message) {
    if (!message.trim()) return;

    const chatContainer = document.getElementById('chatMessages');
    const userMessage = document.createElement('div');
    userMessage.className = 'chat-message user';
    userMessage.innerHTML = `
        <div class="message-avatar user"></div>
        <div class="message-content">
            <p>${escapeHtml(message)}</p>
            <span class="message-time">${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</span>
        </div>
    `;
    chatContainer.appendChild(userMessage);

    document.getElementById('chatInput').value = '';

    const typingIndicator = document.createElement('div');
    typingIndicator.className = 'typing-indicator';
    typingIndicator.innerHTML = '<span></span><span></span><span></span>';
    chatContainer.appendChild(typingIndicator);

    chatContainer.scrollTop = chatContainer.scrollHeight;

    animateMouth('speaking');

    // Use digital human API which returns emotion
    fetch('/api/v1/dh/chat/text', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: 'home_' + Date.now(), input_text: message })
    })
    .then(response => {
        if (!response.ok) throw new Error('服务器响应错误: ' + response.status);
        return response.json();
    })
    .then(data => {
        typingIndicator.remove();

        if (data.code !== 0) {
            const errorMsg = data.message || '抱歉，服务出现问题';
            const aiMessage = document.createElement('div');
            aiMessage.className = 'chat-message ai';
            aiMessage.innerHTML = `
                <div class="message-avatar"></div>
                <div class="message-content">
                    <p>${escapeHtml(errorMsg)}</p>
                    <span class="message-time">${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</span>
                </div>
            `;
            chatContainer.appendChild(aiMessage);
            chatContainer.scrollTop = chatContainer.scrollHeight;
            animateMouth('idle');
            return;
        }

        const answerText = data.data?.answer_text || '抱歉，我暂时无法回答这个问题。';
        const emotion = data.data?.emotion || 'neutral';
        // Strip emotion tag for display
        const displayText = answerText.replace(/^\[.*?\]\s*/, '');

        const aiMessage = document.createElement('div');
        aiMessage.className = 'chat-message ai';
        aiMessage.innerHTML = `
            <div class="message-avatar"></div>
            <div class="message-content">
                <p>${escapeHtml(displayText)}</p>
                <span class="message-time">${new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</span>
            </div>
        `;
        chatContainer.appendChild(aiMessage);
        chatContainer.scrollTop = chatContainer.scrollHeight;

        // Update digital human emotion
        updateDigitalHumanEmotion(emotion);

        const routeData = data.data?.route_payload || data.route;
        if (routeData) {
            updateRoutePanel(routeData);
        } else {
            updateRoutePanel(matchRoute(message));
        }

        if (document.getElementById('autoSpeak')?.checked && displayText) {
            speakText(displayText);
        }

        animateMouth('idle');
    })
    .catch(error => {
        typingIndicator.remove();
        animateMouth('idle');
        console.error('Error:', error);
        const errMsg = document.createElement('div');
        errMsg.className = 'chat-message ai';
        errMsg.innerHTML = '<div class="message-avatar"></div><div class="message-content"><p>网络错误，请确认后端服务已启动</p></div>';
        chatContainer.appendChild(errMsg);
    });
}

function matchRoute(message) {
    if (message.includes('亲子') || message.includes('儿童') || message.includes('家庭')) {
        return routes.family;
    } else if (message.includes('历史') || message.includes('文化') || message.includes('千年')) {
        return routes.history;
    } else if (message.includes('自然') || message.includes('风景') || message.includes('拈花湾')) {
        return routes.nature;
    }
    return routes.classic;
}

function updateRoutePanel(route) {
    const routeName = document.getElementById('routeName');
    const routeDuration = document.getElementById('routeDuration');
    const routeDesc = document.getElementById('routeDesc');
    const routeStepCount = document.getElementById('routeStepCount');
    const timelineList = document.getElementById('routeTimeline');

    if (!route || !routeName || !routeDuration || !routeDesc || !routeStepCount || !timelineList) {
        return;
    }

    const normalizedRoute = route.steps ? route : {
        name: route.route_id || route.name || '推荐路线',
        duration: route.duration || '按现场游览节奏安排',
        description: route.description || '已根据您的兴趣推荐相关游览点。',
        steps: (route.stops || []).map((name, index) => ({
            name,
            desc: index === 0 ? '路线起点' : '推荐停靠点',
            time: ''
        }))
    };

    routeName.textContent = normalizedRoute.name;
    routeDuration.textContent = normalizedRoute.duration;
    routeDesc.textContent = normalizedRoute.description;
    routeStepCount.textContent = normalizedRoute.steps.length;

    timelineList.innerHTML = '';

    normalizedRoute.steps.forEach((step, index) => {
        const item = document.createElement('div');
        item.className = `timeline-item ${index === 0 ? 'active' : ''}`;
        item.innerHTML = `
            <div class="timeline-dot ${index === 0 ? 'start' : index === normalizedRoute.steps.length - 1 ? 'end' : ''}"></div>
            ${index < normalizedRoute.steps.length - 1 ? '<div class="timeline-line"></div>' : ''}
            <div class="timeline-content">
                <h5>${escapeHtml(step.name)}</h5>
                <p>${escapeHtml(step.desc)}</p>
                <span class="time">${escapeHtml(step.time || '')}</span>
            </div>
        `;
        timelineList.appendChild(item);
    });
}

function switchRoute(type) {
    document.querySelectorAll('.route-tab').forEach(tab => tab.classList.remove('active'));
    event.target.classList.add('active');
    updateRoutePanel(routes[type]);
}

function speakText(text) {
    if ('speechSynthesis' in window) {
        window.speechSynthesis.cancel();

        const utterance = new SpeechSynthesisUtterance(text);
        utterance.lang = 'zh-CN';
        utterance.rate = parseFloat(document.getElementById('speechRate')?.value) || 0.8;
        utterance.onstart = () => {
            isSpeaking = true;
            animateMouth('speaking');
            const stopBtn = document.getElementById('stopBtn');
            if (stopBtn) stopBtn.disabled = false;
        };
        utterance.onend = () => {
            isSpeaking = false;
            animateMouth('idle');
            const stopBtn = document.getElementById('stopBtn');
            if (stopBtn) stopBtn.disabled = true;
        };

        window.speechSynthesis.speak(utterance);
    }
}

function stopSpeaking() {
    if ('speechSynthesis' in window) {
        window.speechSynthesis.cancel();
        isSpeaking = false;
        animateMouth('idle');
        const stopBtn = document.getElementById('stopBtn');
        if (stopBtn) stopBtn.disabled = true;
    }
}

function animateMouth(state) {
    const mouth = document.getElementById('mouth');
    if (!mouth) return;
    if (state === 'speaking') {
        mouth.classList.add('speaking');
    } else {
        mouth.classList.remove('speaking');
    }
}

function startVoiceInput() {
    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
        alert('您的浏览器不支持语音识别');
        return;
    }

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    recognition = new SpeechRecognition();
    recognition.lang = 'zh-CN';
    recognition.interimResults = false;

    recognition.onstart = () => {
        const voiceInputBtn = document.getElementById('voiceInputBtn') || document.getElementById('voiceBtn');
        const stopVoiceBtn = document.getElementById('stopVoiceBtn');
        const emotionBubble = document.getElementById('emotionBubble') || document.getElementById('digitalHumanMessage');
        if (voiceInputBtn) {
            voiceInputBtn.disabled = true;
            voiceInputBtn.classList.add('recording');
        }
        if (stopVoiceBtn) stopVoiceBtn.disabled = false;
        if (emotionBubble) emotionBubble.textContent = '正在听...';
    };

    recognition.onresult = (event) => {
        const result = event.results[0][0].transcript;
        document.getElementById('chatInput').value = result;
        const emotionBubble = document.getElementById('emotionBubble') || document.getElementById('digitalHumanMessage');
        if (emotionBubble) emotionBubble.textContent = '识别完成';
        sendMessage(result);
    };

    recognition.onerror = () => {
        const voiceInputBtn = document.getElementById('voiceInputBtn') || document.getElementById('voiceBtn');
        const stopVoiceBtn = document.getElementById('stopVoiceBtn');
        const emotionBubble = document.getElementById('emotionBubble') || document.getElementById('digitalHumanMessage');
        if (voiceInputBtn) {
            voiceInputBtn.disabled = false;
            voiceInputBtn.classList.remove('recording');
        }
        if (stopVoiceBtn) stopVoiceBtn.disabled = true;
        if (emotionBubble) emotionBubble.textContent = '语音识别失败';
    };

    recognition.onend = () => {
        const voiceInputBtn = document.getElementById('voiceInputBtn') || document.getElementById('voiceBtn');
        const stopVoiceBtn = document.getElementById('stopVoiceBtn');
        const emotionBubble = document.getElementById('emotionBubble') || document.getElementById('digitalHumanMessage');
        if (voiceInputBtn) {
            voiceInputBtn.disabled = false;
            voiceInputBtn.classList.remove('recording');
        }
        if (stopVoiceBtn) stopVoiceBtn.disabled = true;
        if (emotionBubble) emotionBubble.textContent = '您好！我是您的AI导览员灵灵';
    };

    recognition.start();
}

function stopVoiceInput() {
    if (recognition) {
        recognition.stop();
    }
}

function initAMap() {
    const mapContainer = document.getElementById('mapContainer');
    if (!mapContainer) return;

    if (typeof AMap === 'undefined') {
        const retries = initAMap.retries || 0;
        if (retries < 10) {
            initAMap.retries = retries + 1;
            setTimeout(initAMap, 1000);
            return;
        } else {
            document.querySelector('.map-loading').innerHTML = '<i class="fas fa-exclamation-circle"></i><span>地图加载失败</span>';
            return;
        }
    }

    try {
        map = new AMap.Map('mapContainer', {
            center: [120.258, 31.431],
            zoom: 17,
            mapStyle: 'amap://styles/dark',
            features: ['bg', 'road', 'building', 'point']
        });

        map.on('complete', function() {
            const loadingEl = document.querySelector('.map-loading');
            if (loadingEl) loadingEl.style.display = 'none';
            loadRouteMarkers();
        });

        AMap.plugin('AMap.Geolocation', function() {
            try {
                const geolocation = new AMap.Geolocation({
                    enableHighAccuracy: true,
                    timeout: 10000,
                    maximumAge: 0,
                    convert: true
                });
                geolocation.getCurrentPosition();
                geolocation.on('complete', onLocationSuccess);
                geolocation.on('error', onLocationError);
            } catch (e) {
                console.error('定位插件加载失败:', e);
            }
        });
    } catch (e) {
        console.error('高德地图初始化失败:', e);
        document.querySelector('.map-loading').innerHTML = '<i class="fas fa-exclamation-circle"></i><span>地图加载失败</span>';
    }
}

function onLocationSuccess(data) {
    console.log('定位成功:', data);
}

function onLocationError(err) {
    console.error('定位失败:', err);
}

function loadRouteMarkers() {
    if (!map) return;

    const points = [
        { name: '灵山胜境入口', position: [120.255, 31.428] },
        { name: '灵山大照壁', position: [120.256, 31.430] },
        { name: '五明桥', position: [120.257, 31.431] },
        { name: '佛足坛', position: [120.258, 31.432] },
        { name: '灵山大佛', position: [120.260, 31.434] }
    ];

    points.forEach((point, index) => {
        const marker = new AMap.Marker({
            position: point.position,
            title: point.name,
            icon: new AMap.Icon({
                size: new AMap.Size(32, 32),
                image: `https://webapi.amap.com/theme/v1.3/markers/n/${index === 0 ? 'start' : index === points.length - 1 ? 'end' : 'mid'}.png`
            })
        });
        map.add(marker);
    });

    const path = points.map(p => p.position);
    const polyline = new AMap.Polyline({
        path: path,
        strokeColor: '#D4AF37',
        strokeWeight: 4,
        strokeOpacity: 0.8
    });
    map.add(polyline);
}

function getGPSLocation() {
    if (!map) {
        alert('地图尚未加载完成');
        return;
    }

    AMap.plugin('AMap.Geolocation', function() {
        const geolocation = new AMap.Geolocation({
            enableHighAccuracy: true,
            timeout: 10000
        });

        geolocation.getCurrentPosition(function(status, result) {
            if (status === 'complete') {
                map.setCenter(result.position);
                map.setZoom(18);

                new AMap.Marker({
                    position: result.position,
                    icon: new AMap.Icon({
                        size: new AMap.Size(40, 40),
                        image: 'https://webapi.amap.com/theme/v1.3/markers/n/blueA.png'
                    })
                }).addTo(map);

                alert(`定位成功！\n经度: ${result.position.lng}\n纬度: ${result.position.lat}`);
            } else {
                alert('定位失败: ' + result.message);
            }
        });
    });
}

function openAMapNavigation() {
    window.open('https://uri.amap.com/marker?position=120.260,31.434&name=灵山大佛&src=mypage&coordinate=gaode&callnative=0');
}

function startNavigation() {
    alert('已开始导航模式，正在为您播报路线...');
    speakText(`欢迎来到灵山胜境，现在为您开始经典文化之旅导航。首先，请前往灵山胜境入口。`);
}

function shareRoute() {
    const routeName = document.getElementById('routeName').textContent;
    const shareData = {
        title: `灵山胜境 - ${routeName}`,
        text: `推荐游览路线：${routeName}，${document.getElementById('routeDuration').textContent}`,
        url: window.location.href
    };

    if (navigator.share) {
        navigator.share(shareData).catch(() => {
            copyToClipboard(window.location.href);
        });
    } else {
        copyToClipboard(window.location.href);
    }
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        alert('路线链接已复制到剪贴板');
    }).catch(() => {
        alert('复制失败，请手动复制');
    });
}

function clearChat() {
    const chatContainer = document.getElementById('chatMessages');
    chatContainer.innerHTML = `
        <div class="welcome-message">
            <i class="fas fa-robot"></i>
            <p>您好！我是灵灵，您的AI导览员</p>
            <p class="hint">有什么可以帮您的？我可以为您讲解景点、推荐路线...</p>
        </div>
    `;
}

document.addEventListener('DOMContentLoaded', function() {
    initNavigation();
    initDigitalHuman();
    initAMap();

    const chatInput = document.getElementById('chatInput');
    if (chatInput) {
        chatInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage(this.value);
            }
        });
    }

    const voiceBtn = document.getElementById('voiceBtn');
    if (voiceBtn) {
        voiceBtn.addEventListener('click', function() {
            if (!isSpeaking) {
                startVoiceInput();
            }
        });
    }

    const sendBtn = document.getElementById('sendBtn');
    if (sendBtn && chatInput) {
        sendBtn.addEventListener('click', function() {
            const message = chatInput.value;
            if (message.trim()) {
                sendMessage(message);
            }
        });
    }

    const voiceInputBtn = document.getElementById('voiceInputBtn');
    if (voiceInputBtn) {
        voiceInputBtn.addEventListener('click', startVoiceInput);
    }
    const stopVoiceBtn = document.getElementById('stopVoiceBtn');
    if (stopVoiceBtn) {
        stopVoiceBtn.addEventListener('click', stopVoiceInput);
    }

    document.getElementById('gpsBtn')?.addEventListener('click', getGPSLocation);
    document.getElementById('navBtn')?.addEventListener('click', openAMapNavigation);
    document.getElementById('startRouteBtn')?.addEventListener('click', startNavigation);
    document.getElementById('shareRouteBtn')?.addEventListener('click', shareRoute);
    document.getElementById('clearChat')?.addEventListener('click', clearChat);

    const toggleDHBtn = document.getElementById('toggleDHBtn');
    if (toggleDHBtn) {
        toggleDHBtn.addEventListener('click', openDigitalHuman);
    }

    const loginBtn = document.getElementById('loginBtn');
    if (loginBtn) {
        loginBtn.addEventListener('click', function() {
            const modal = document.getElementById('loginModal');
            if (modal) {
                modal.classList.add('show');
            }
        });
    }

    const spotsFilterBtns = document.querySelectorAll('.filter-btn[data-category]');
    spotsFilterBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            document.querySelectorAll('.filter-btn[data-category]').forEach(b => b.classList.remove('active'));
            this.classList.add('active');
            const category = this.getAttribute('data-category');
            filterSpots(category);
        });
    });

    const routesFilterBtns = document.querySelectorAll('.filter-btn[data-difficulty]');
    routesFilterBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            document.querySelectorAll('.filter-btn[data-difficulty]').forEach(b => b.classList.remove('active'));
            this.classList.add('active');
            const difficulty = this.getAttribute('data-difficulty');
            filterRoutes(difficulty);
        });
    });

    const updateProfileBtn = document.getElementById('updateProfileBtn');
    if (updateProfileBtn) {
        updateProfileBtn.addEventListener('click', function() {
            alert('个人资料修改功能开发中...');
        });
    }

    const myQueriesBtn = document.getElementById('myQueriesBtn');
    if (myQueriesBtn) {
        myQueriesBtn.addEventListener('click', function() {
            alert('我的问答记录功能开发中...');
        });
    }

    const adminBtn = document.getElementById('adminBtn');
    if (adminBtn) {
        adminBtn.addEventListener('click', function() {
            window.location.href = '/app#/admin';
        });
    }

    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', function() {
            currentUser = null;
            localStorage.removeItem('authToken');
            localStorage.removeItem('currentUser');
            document.getElementById('userName').textContent = '游客';
            document.getElementById('loginBtn').style.display = 'inline-block';
            document.getElementById('logoutBtn').style.display = 'none';
            document.getElementById('adminBtn').style.display = 'none';
            alert('已退出登录');
        });
    }

    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', function(e) {
            e.preventDefault();
            login();
        });
    }

    const registerForm = document.getElementById('registerForm');
    if (registerForm) {
        registerForm.addEventListener('submit', function(e) {
            e.preventDefault();
            register();
        });
    }

    updateUserInterface();
    updateRoutePanel(routes.classic);

    openLoginFromHash();
});
