const API_BASE = '/api/v1';

function authHeaders() {
    const token = localStorage.getItem('authToken');
    return token ? { Authorization: `Bearer ${token}` } : {};
}

const sectionCopy = {
    knowledge: {
        title: '知识库管理',
        desc: '维护讲解词、文史资料、FAQ 与路线推荐语料，支持 RAG 精准问答。'
    },
    avatar: {
        title: '数字人形象',
        desc: '配置数字人外观、声音、服装和情绪表达，让导览服务更自然。'
    },
    reports: {
        title: '游客感受度报告',
        desc: '分析游客关注点、情绪趋势和服务改进建议，辅助景区运营决策。'
    },
    settings: {
        title: '系统设置',
        desc: '管理景区基础信息、能力开关和复杂场景兜底策略。'
    }
};

const mockKnowledge = [
    {
        id: 'k-001',
        title: '灵山大佛高度与文化寓意',
        category: 'history',
        tag: '历史文化',
        content: '灵山大佛通高88米，依山而建，面向太湖，体现庄严、慈悲与江南山水意境的融合。',
        updated: '2026-05-08'
    },
    {
        id: 'k-002',
        title: '九龙灌浴表演时间与推荐站位',
        category: 'scenic',
        tag: '景点特色',
        content: '九龙灌浴是景区核心动态演艺之一，建议游客提前10分钟到达广场东侧，视野更完整。',
        updated: '2026-05-07'
    },
    {
        id: 'k-003',
        title: '亲子轻松游路线',
        category: 'route',
        tag: '路线讲解',
        content: '推荐游客从游客中心出发，经九龙灌浴、梵宫、五印坛城，再返回商业街休息补给。',
        updated: '2026-05-06'
    },
    {
        id: 'k-004',
        title: '开放时间与入园须知',
        category: 'faq',
        tag: '游客FAQ',
        content: '景区模拟开放时间为08:00-17:30，节假日建议提前预约，并关注实时客流提示。',
        updated: '2026-05-05'
    }
];

const reportAttention = [
    ['开放时间', 92],
    ['灵山大佛', 88],
    ['路线推荐', 81],
    ['演出安排', 74],
    ['餐饮休息', 62],
    ['停车交通', 56]
];

function showSection(sectionId, trigger) {
    document.querySelectorAll('.content-section').forEach(section => section.classList.remove('active'));
    document.querySelectorAll('.rail-nav .nav-item').forEach(item => item.classList.remove('active'));
    const section = document.getElementById(`${sectionId}Section`);
    if (section) section.classList.add('active');
    if (trigger) trigger.classList.add('active');
    const copy = sectionCopy[sectionId];
    if (copy) {
        document.getElementById('pageTitle').textContent = copy.title;
        document.getElementById('pageDesc').textContent = copy.desc;
    }
    if (sectionId === 'reports') renderAttentionAnalysis();
}

function updateCurrentTime() {
    const el = document.getElementById('currentTime');
    if (!el) return;
    el.textContent = new Date().toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function renderKnowledgeList(items = mockKnowledge) {
    const container = document.getElementById('knowledgeList');
    if (!container) return;
    container.innerHTML = items.map(item => `
        <article class="knowledge-card" data-category="${item.category}">
            <div class="card-header">
                <h3>${escapeHtml(item.title)}</h3>
                <span class="category-tag">${escapeHtml(item.tag)}</span>
            </div>
            <p>${escapeHtml(item.content)}</p>
            <div class="card-footer">
                <span>更新：${item.updated}</span>
                <div class="card-actions">
                    <button class="action-btn" onclick="editKnowledge('${item.id}')" aria-label="编辑"><i class="fas fa-edit"></i></button>
                    <button class="action-btn" onclick="deleteKnowledge('${item.id}')" aria-label="删除"><i class="fas fa-trash"></i></button>
                </div>
            </div>
        </article>
    `).join('');
    document.getElementById('metricKnowledge').textContent = (1286 + items.length).toLocaleString('zh-CN');
}

async function loadKnowledgeList() {
    try {
        const resp = await fetch(`${API_BASE}/knowledge/list?page=1&page_size=20`, {
            headers: authHeaders()
        });
        const data = await resp.json();
        const list = data && data.code === 0 && data.data && data.data.list ? data.data.list : [];
        if (!list.length) {
            renderKnowledgeList();
            return;
        }
        renderKnowledgeList(list.map((item, index) => ({
            id: item.id || `api-${index}`,
            title: item.title || `知识片段 ${index + 1}`,
            category: item.source || 'faq',
            tag: sourceLabel(item.source),
            content: item.content || '',
            updated: new Date(item.updated_at || item.created_at || Date.now()).toLocaleDateString('zh-CN')
        })));
    } catch (error) {
        renderKnowledgeList();
    }
}

function sourceLabel(source) {
    return { history: '历史文化', scenic: '景点特色', faq: '游客FAQ', guide: '路线讲解', route: '路线讲解' }[source] || '知识片段';
}

function filterKnowledge() {
    const term = (document.getElementById('knowledgeSearch')?.value || '').trim().toLowerCase();
    const active = document.querySelector('.category-btn.active')?.dataset.category || 'all';
    document.querySelectorAll('.knowledge-card').forEach(card => {
        const categoryOk = active === 'all' || card.dataset.category === active;
        const textOk = !term || card.textContent.toLowerCase().includes(term);
        card.style.display = categoryOk && textOk ? '' : 'none';
    });
}

function showAddKnowledgeModal() {
    document.getElementById('knowledgeTitle').value = '';
    document.getElementById('knowledgeCategory').value = 'history';
    document.getElementById('knowledgeContent').value = '';
    document.getElementById('knowledgeTags').value = '';
    const modal = document.getElementById('addKnowledgeModal');
    modal.classList.remove('hidden');
    modal.classList.add('show');
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.add('hidden');
    modal.classList.remove('show');
}

async function saveKnowledge() {
    const title = document.getElementById('knowledgeTitle').value.trim();
    const category = document.getElementById('knowledgeCategory').value;
    const content = document.getElementById('knowledgeContent').value.trim();
    if (!title || !content) {
        alert('请填写标题和内容');
        return;
    }
    mockKnowledge.unshift({
        id: `local-${Date.now()}`,
        title,
        category,
        tag: sourceLabel(category),
        content,
        updated: new Date().toLocaleDateString('zh-CN')
    });
    closeModal('addKnowledgeModal');
    renderKnowledgeList();
    alert('已保存到演示知识库');
}

function editKnowledge(id) {
    const item = mockKnowledge.find(row => row.id === id);
    if (!item) {
        alert('接口知识片段可在正式版中进入编辑流程，当前演示已保留操作入口。');
        return;
    }
    showAddKnowledgeModal();
    document.getElementById('knowledgeTitle').value = item.title;
    document.getElementById('knowledgeCategory').value = item.category;
    document.getElementById('knowledgeContent').value = item.content;
}

function deleteKnowledge(id) {
    const index = mockKnowledge.findIndex(row => row.id === id);
    if (index >= 0) {
        mockKnowledge.splice(index, 1);
        renderKnowledgeList();
    }
}

function saveAvatarSettings() {
    alert('数字人形象配置已保存到演示状态');
}

function saveSettings() {
    alert('系统设置已保存到演示状态');
}

function renderAttentionAnalysis() {
    const container = document.getElementById('attentionChart');
    if (!container) return;
    container.innerHTML = reportAttention.map(([label, value]) => `
        <div class="bar-item">
            <span class="bar-label">${label}</span>
            <div class="bar-wrapper"><div class="bar" style="width:${value}%"></div></div>
            <strong>${value}%</strong>
        </div>
    `).join('');
}

function escapeHtml(value) {
    const div = document.createElement('div');
    div.textContent = value || '';
    return div.innerHTML;
}

document.addEventListener('DOMContentLoaded', () => {
    updateCurrentTime();
    setInterval(updateCurrentTime, 1000);
    loadKnowledgeList();
    renderAttentionAnalysis();

    document.getElementById('knowledgeSearch')?.addEventListener('input', filterKnowledge);
    document.querySelectorAll('.category-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.category-btn').forEach(item => item.classList.remove('active'));
            btn.classList.add('active');
            filterKnowledge();
        });
    });
});
