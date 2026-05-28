<script setup lang="ts">
import { computed, onMounted, reactive } from 'vue';
import { NTabs, NTabPane, NCard, NGrid, NGi, NStatistic, NButton, NInput, NSelect, NSpace, NForm, NFormItem, NDataTable, NTag, NPopconfirm, NSwitch, NSpin, NEmpty, useMessage } from 'naive-ui';
import KpiCard from '../components/KpiCard.vue';
import BarList from '../components/BarList.vue';
import DonutChart from '../components/DonutChart.vue';

const message = useMessage();

type KnowledgeItem = {
  id: string;
  title: string;
  source: string;
  content: string;
  category: string;
  metadata: string;
  updated: string;
};

type AvatarConfig = {
  name: string;
  appearance: string;
  costume: string;
  style: string;
  color: string;
  culture_theme: string;
  voice_type: string;
  voice_tone: string;
  speed: number;
  volume: number;
  greeting: string;
  default_emotion: string;
  emotion_level: number;
};

type VisitorReport = {
  attention_analysis: Array<{ label: string; value: number }>;
  emotion_distribution: Array<{ label: string; icon: string; count: number; percent: number }>;
  emotion_trend: Array<{ date: string; positive_rate: number; negative_rate: number; total: number }>;
  suggestions: Array<{ content: string }>;
  peak_hours: Array<{ hour: string; count: number }>;
  summary: {
    total_interactions: number;
    satisfaction_rate: number;
    negative_rate: number;
    top_concern: string;
    peak_hour: string;
  };
};

const emptyEditor = {
  title: '',
  category: '讲解词',
  source: 'admin',
  content: '',
};

const defaultAvatarConfig: AvatarConfig = {
  name: '小灵',
  appearance: '亲和型国风讲解员',
  costume: '古典汉服',
  style: '古典汉服',
  color: '#D4AF37',
  culture_theme: '灵山佛教文化与江南山水意境',
  voice_type: '温柔自然女声',
  voice_tone: '温暖、端庄、亲切',
  speed: 0.8,
  volume: 80,
  greeting: '欢迎来到灵山胜境，我是您的数字导览员小灵。',
  default_emotion: 'joy',
  emotion_level: 3,
};

type SystemSettings = {
  scenic_name: string;
  scenic_desc: string;
  service_hotline: string;
  enable_login: boolean;
  enable_voice: boolean;
  enable_filter: boolean;
  backup_frequency: string;
  data_retention: string;
};

const defaultSettings: SystemSettings = {
  scenic_name: '灵山胜境',
  scenic_desc: '灵山胜境是著名的佛教文化景区，集自然风光与人文景观于一体。',
  service_hotline: '400-168-0303',
  enable_login: true,
  enable_voice: true,
  enable_filter: true,
  backup_frequency: '每日',
  data_retention: '30',
};

const defaultVisitorReport: VisitorReport = {
  attention_analysis: [],
  emotion_distribution: [],
  emotion_trend: [],
  suggestions: [],
  peak_hours: [],
  summary: {
    total_interactions: 0,
    satisfaction_rate: 0,
    negative_rate: 0,
    top_concern: '暂无数据',
    peak_hour: '暂无数据',
  },
};

const state = reactive({
  tab: 'knowledge',
  search: '',
  loading: false,
  saving: false,
  message: '',
  error: '',
  total: 0,
  page: 1,
  pageSize: 100,
  knowledge: [] as KnowledgeItem[],
  editingID: '',
  editor: { ...emptyEditor },
  uploadCategory: '文史资料',
  selectedFile: null as File | null,
  avatarLoading: false,
  avatarSaving: false,
  avatarMessage: '',
  avatarError: '',
  avatar: { ...defaultAvatarConfig } as AvatarConfig,
  reportLoading: false,
  reportError: '',
  report: { ...defaultVisitorReport } as VisitorReport,
  settingsLoading: false,
  settingsSaving: false,
  settingsMessage: '',
  settingsError: '',
  settings: { ...defaultSettings } as SystemSettings,
});

const tabs = [
  ['knowledge', '知识库管理'],
  ['avatar', '数字人形象'],
  ['reports', '感受度报告'],
  ['settings', '系统设置'],
];

const categoryOptions = ['讲解词', '文史资料', '游客 FAQ', '路线推荐', '服务设施', '票务交通'];
const appearanceOptions = ['亲和型国风讲解员', '端庄礼仪讲解员', '青春活力文旅推荐官', '禅意文化讲述者'];
const costumeOptions = ['古典汉服', '景区文旅制服', '山水讲解员制服', '禅意素雅长衫', '节庆主题服饰'];
const voiceOptions = ['温柔自然女声', '沉稳专业女声', '活力亲切女声', '端庄礼仪女声'];
const toneOptions = ['温暖、端庄、亲切', '专业、清晰、克制', '活泼、轻快、有陪伴感', '舒缓、禅意、适合文化讲解'];
const emotionOptions = [
  ['joy', '亲切微笑'],
  ['neutral', '自然平和'],
  ['surprise', '热情提示'],
  ['sadness', '温和致歉'],
];

const filtered = computed(() => {
  const term = state.search.trim().toLowerCase();
  if (!term) return state.knowledge;
  return state.knowledge.filter(item =>
    `${item.title}${item.category}${item.source}${item.content}`.toLowerCase().includes(term),
  );
});

const currentTitle = computed(() => tabs.find(([key]) => key === state.tab)?.[1] ?? '管理后台');
const avatarUpdatedNote = computed(() => `${state.avatar.costume} / ${state.avatar.voice_type}`);
const attentionBars = computed(() => state.report.attention_analysis.map(item => ({
  label: item.label,
  value: Math.round(item.value),
})));
const emotionDonutItems = computed(() => {
  const colors: Record<string, string> = {
    正面: '#7ef2a0',
    中性: '#52f0ee',
    负面: '#ff8b8b',
  };
  return state.report.emotion_distribution.map(item => ({
    label: item.label,
    value: Math.round(item.percent),
    color: colors[item.label] || '#f4c765',
  }));
});
const peakHourBars = computed(() => {
  const max = Math.max(...state.report.peak_hours.map(item => item.count), 1);
  return state.report.peak_hours.map(item => ({
    label: item.hour,
    value: Math.round(item.count / max * 100),
    suffix: ` / ${item.count}`,
  }));
});

function getField(item: Record<string, unknown>, key: string) {
  return item[key] ?? item[key.charAt(0).toUpperCase() + key.slice(1)] ?? '';
}

function getCategory(metadata: string, source: string) {
  if (!metadata) return source || '未分类';
  try {
    const parsed = JSON.parse(metadata);
    return parsed.category || parsed.filename || source || '未分类';
  } catch {
    return source || '未分类';
  }
}

function normalizeKnowledge(raw: Record<string, unknown>): KnowledgeItem {
  const metadata = String(getField(raw, 'metadata') || '');
  const source = String(getField(raw, 'source') || 'admin');
  const updatedAt = String(getField(raw, 'updatedAt') || getField(raw, 'UpdatedAt') || '');
  return {
    id: String(getField(raw, 'id') || getField(raw, 'ID')),
    title: String(getField(raw, 'title')),
    source,
    content: String(getField(raw, 'content')),
    category: getCategory(metadata, source),
    metadata,
    updated: updatedAt ? new Date(updatedAt).toLocaleDateString('zh-CN') : '-',
  };
}

async function apiFetch<T = unknown>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('authToken');
  const headers: HeadersInit = {
    ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options?.headers || {}),
  };
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers,
  });
  const raw = await response.text();
  let payload: { code?: number; message?: string; msg?: string; data?: unknown } = {};
  if (raw.trim()) {
    try {
      payload = JSON.parse(raw);
    } catch {
      throw new Error(`接口返回非 JSON 响应 (${response.status})`);
    }
  }
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || payload.msg || response.statusText || `请求失败 (${response.status})`);
  }
  return payload.data as T;
}

function normalizeAvatarConfig(raw: Partial<AvatarConfig>): AvatarConfig {
  return {
    ...defaultAvatarConfig,
    ...raw,
    speed: Number(raw.speed ?? defaultAvatarConfig.speed),
    volume: Number(raw.volume ?? defaultAvatarConfig.volume),
    emotion_level: Number(raw.emotion_level ?? defaultAvatarConfig.emotion_level),
  };
}

async function loadKnowledge() {
  state.loading = true;
  state.error = '';
  try {
    const data = await apiFetch<{ list?: Array<Record<string, unknown>>; total?: number }>(`/knowledge/list?page=${state.page}&page_size=${state.pageSize}`);
    state.knowledge = (data.list || []).map((item: Record<string, unknown>) => normalizeKnowledge(item));
    state.total = Number(data.total || state.knowledge.length);
  } catch (error) {
    state.error = error instanceof Error ? error.message : '知识库加载失败';
  } finally {
    state.loading = false;
  }
}

async function loadAvatarConfig() {
  state.avatarLoading = true;
  state.avatarError = '';
  try {
    const data = await apiFetch<Partial<AvatarConfig>>('/admin/digital-human/config');
    state.avatar = normalizeAvatarConfig(data);
  } catch (error) {
    state.avatarError = error instanceof Error ? error.message : '数字人配置加载失败';
  } finally {
    state.avatarLoading = false;
  }
}

async function loadVisitorReport() {
  state.reportLoading = true;
  state.reportError = '';
  try {
    const data = await apiFetch<Partial<VisitorReport> & { summary?: Partial<VisitorReport['summary']> }>('/admin/reports/visitor');
    state.report = {
      ...defaultVisitorReport,
      ...data,
      summary: { ...defaultVisitorReport.summary, ...(data.summary || {}) },
    };
  } catch (error) {
    state.reportError = error instanceof Error ? error.message : '感受度报告加载失败';
  } finally {
    state.reportLoading = false;
  }
}

async function saveAvatarConfig() {
  state.avatarSaving = true;
  state.avatarError = '';
  state.avatarMessage = '';
  try {
    await apiFetch('/admin/digital-human/config', {
      method: 'PUT',
      body: JSON.stringify(state.avatar),
    });
    state.avatarMessage = '数字人形象配置已保存。';
    await loadAvatarConfig();
  } catch (error) {
    state.avatarError = error instanceof Error ? error.message : '保存失败';
  } finally {
    state.avatarSaving = false;
  }
}

async function loadSettings() {
  state.settingsLoading = true;
  state.settingsError = '';
  try {
    const data = await apiFetch<Partial<SystemSettings>>('/admin/settings');
    state.settings = { ...defaultSettings, ...data };
  } catch (error) {
    state.settingsError = error instanceof Error ? error.message : '系统设置加载失败';
  } finally {
    state.settingsLoading = false;
  }
}

async function saveSettings() {
  state.settingsSaving = true;
  state.settingsError = '';
  state.settingsMessage = '';
  try {
    await apiFetch('/admin/settings', {
      method: 'PUT',
      body: JSON.stringify(state.settings),
    });
    state.settingsMessage = '系统设置已保存。';
  } catch (error) {
    state.settingsError = error instanceof Error ? error.message : '保存失败';
  } finally {
    state.settingsSaving = false;
  }
}

function resetEditor() {
  state.editingID = '';
  state.editor = { ...emptyEditor };
}

function editKnowledge(item: KnowledgeItem) {
  state.editingID = item.id;
  state.editor = {
    title: item.title,
    category: item.category,
    source: item.source,
    content: item.content,
  };
}

async function saveKnowledge() {
  state.saving = true;
  state.error = '';
  state.message = '';
  try {
    const body = JSON.stringify(state.editor);
    const path = state.editingID ? `/knowledge/${encodeURIComponent(state.editingID)}` : '/knowledge';
    const method = state.editingID ? 'PUT' : 'POST';
    await apiFetch(path, { method, body });
    state.message = state.editingID ? '知识条目已更新，数字人检索缓存已刷新。' : '知识条目已加入数字人知识库。';
    resetEditor();
    await loadKnowledge();
  } catch (error) {
    state.error = error instanceof Error ? error.message : '保存失败';
  } finally {
    state.saving = false;
  }
}

async function deleteKnowledge(item: KnowledgeItem) {
  if (!window.confirm(`确认删除「${item.title}」吗？`)) return;
  state.error = '';
  state.message = '';
  try {
    await apiFetch(`/knowledge/${encodeURIComponent(item.id)}`, { method: 'DELETE' });
    state.message = '知识条目已删除。';
    await loadKnowledge();
  } catch (error) {
    state.error = error instanceof Error ? error.message : '删除失败';
  }
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  state.selectedFile = input.files?.[0] ?? null;
}

async function uploadKnowledge() {
  if (!state.selectedFile) {
    state.error = '请先选择知识文档。';
    return;
  }
  state.saving = true;
  state.error = '';
  state.message = '';
  try {
    const form = new FormData();
    form.append('file', state.selectedFile);
    form.append('category', state.uploadCategory);
    const data = await apiFetch<{ imported?: number; loaded_count?: number }>('/knowledge/upload', { method: 'POST', body: form });
    state.message = `上传完成，已导入 ${data.loaded_count || 0} 条知识片段。`;
    state.selectedFile = null;
    await loadKnowledge();
  } catch (error) {
    state.error = error instanceof Error ? error.message : '上传失败';
  } finally {
    state.saving = false;
  }
}

onMounted(() => {
  loadKnowledge();
  loadAvatarConfig();
  loadVisitorReport();
  loadSettings();
});
</script>

<template>
  <div class="admin-view">
    <div class="admin-header">
      <h1>管理后台</h1>
      <p>维护景区知识、数字人配置和系统设置</p>
    </div>

    <NTabs v-model:value="state.tab" type="line" animated>
      <NTabPane v-for="[key, label] in tabs" :key="key" :name="key" :tab="label">
      </NTabPane>
    </NTabs>

    <div class="admin-content">

      <section class="kpi-grid four">
        <KpiCard label="知识条目" :value="String(state.total)" note="来自 RAG 知识库" />
        <KpiCard label="当前筛选" :value="String(filtered.length)" note="本页匹配结果" tone="green" />
        <KpiCard label="支持格式" value="JSONL/MD" note="另支持 JSON、TXT" tone="gold" />
        <KpiCard label="缓存状态" value="自动刷新" note="增删改后立即生效" />
      </section>

      <section v-if="state.tab === 'knowledge'" class="knowledge-layout">
        <section class="panel form-panel">
          <h2>{{ state.editingID ? '编辑知识条目' : '新增知识条目' }}</h2>
          <label>标题<input v-model="state.editor.title" placeholder="例如：九龙灌浴讲解词" /></label>
          <label>分类<select v-model="state.editor.category"><option v-for="item in categoryOptions" :key="item">{{ item }}</option></select></label>
          <label>来源<input v-model="state.editor.source" placeholder="admin / 文件名 / 景点名称" /></label>
          <label>知识内容<textarea v-model="state.editor.content" rows="9" placeholder="填写讲解词、历史背景、FAQ 问答或运营说明"></textarea></label>
          <div class="button-row">
            <button class="primary-action" :disabled="state.saving || !state.editor.content.trim()" @click="saveKnowledge">
              {{ state.editingID ? '保存更新' : '加入知识库' }}
            </button>
            <button class="secondary-action" type="button" @click="resetEditor">清空</button>
          </div>
        </section>

        <section class="panel upload-panel">
          <h2>上传知识文档</h2>
          <div class="upload-box">
            <input type="file" accept=".jsonl,.json,.md,.markdown,.txt" @change="onFileChange" />
            <select v-model="state.uploadCategory"><option v-for="item in categoryOptions" :key="item">{{ item }}</option></select>
            <button class="primary-action" :disabled="state.saving || !state.selectedFile" @click="uploadKnowledge">上传并导入</button>
          </div>
          <p class="hint-line">JSONL/JSON 需包含 title、content、source、metadata 字段；Markdown/TXT 会按段落自动切片。</p>
          <div v-if="state.message" class="notice success">{{ state.message }}</div>
          <div v-if="state.error" class="notice error">{{ state.error }}</div>
        </section>

        <section class="panel span-2">
          <div class="toolbar">
            <input v-model="state.search" placeholder="搜索景点、讲解词、FAQ、来源..." />
            <button class="secondary-action" :disabled="state.loading" @click="loadKnowledge">刷新</button>
          </div>
          <div v-if="state.loading" class="muted-center">正在加载知识库...</div>
          <div v-else class="knowledge-grid">
            <article v-for="item in filtered" :key="item.id" class="knowledge-card-vue">
              <div>
                <h3>{{ item.title }}</h3>
                <span>{{ item.category }}</span>
              </div>
              <p>{{ item.content }}</p>
              <small>来源：{{ item.source }} / 更新时间：{{ item.updated }}</small>
              <div class="card-actions">
                <button class="secondary-action" @click="editKnowledge(item)">编辑</button>
                <button class="danger-action" @click="deleteKnowledge(item)">删除</button>
              </div>
            </article>
          </div>
        </section>
      </section>

      <section v-if="state.tab === 'avatar'" class="admin-two">
        <article class="panel avatar-config-preview">
          <div class="avatar-holo" :style="{ background: `radial-gradient(circle, ${state.avatar.color}, var(--cyan))` }">
            {{ state.avatar.name }}
          </div>
          <h2>{{ state.avatar.appearance }}</h2>
          <p>{{ state.avatar.culture_theme }}</p>
          <ul class="avatar-summary">
            <li><span>服装</span><strong>{{ state.avatar.costume }}</strong></li>
            <li><span>声音</span><strong>{{ state.avatar.voice_type }}</strong></li>
            <li><span>语气</span><strong>{{ state.avatar.voice_tone }}</strong></li>
          </ul>
          <small class="hint-line">当前方案：{{ avatarUpdatedNote }}</small>
        </article>
        <article class="panel form-panel avatar-form">
          <h2>形象与文化设定</h2>
          <label>数字人名称<input v-model="state.avatar.name" /></label>
          <label>外观定位<select v-model="state.avatar.appearance"><option v-for="item in appearanceOptions" :key="item">{{ item }}</option></select></label>
          <label>服装风格<select v-model="state.avatar.costume"><option v-for="item in costumeOptions" :key="item">{{ item }}</option></select></label>
          <label>主视觉颜色<input v-model="state.avatar.color" type="color" /></label>
          <label>景区文化主题<textarea v-model="state.avatar.culture_theme" rows="3"></textarea></label>
          <label>欢迎语<textarea v-model="state.avatar.greeting" rows="3"></textarea></label>

          <h2>声音与表达</h2>
          <label>讲解声音<select v-model="state.avatar.voice_type"><option v-for="item in voiceOptions" :key="item">{{ item }}</option></select></label>
          <label>讲解语气<select v-model="state.avatar.voice_tone"><option v-for="item in toneOptions" :key="item">{{ item }}</option></select></label>
          <label>语速 {{ state.avatar.speed.toFixed(1) }}<input v-model.number="state.avatar.speed" type="range" min="0.6" max="1.4" step="0.1" /></label>
          <label>音量 {{ state.avatar.volume }}<input v-model.number="state.avatar.volume" type="range" min="0" max="100" step="1" /></label>
          <label>默认表情<select v-model="state.avatar.default_emotion"><option v-for="[value, label] in emotionOptions" :key="value" :value="value">{{ label }}</option></select></label>
          <label>表情强度 {{ state.avatar.emotion_level }}<input v-model.number="state.avatar.emotion_level" type="range" min="1" max="5" step="1" /></label>

          <div class="button-row">
            <button class="primary-action" :disabled="state.avatarSaving || state.avatarLoading" @click="saveAvatarConfig">保存配置</button>
            <button class="secondary-action" type="button" :disabled="state.avatarLoading" @click="loadAvatarConfig">重新加载</button>
          </div>
          <div v-if="state.avatarMessage" class="notice success">{{ state.avatarMessage }}</div>
          <div v-if="state.avatarError" class="notice error">{{ state.avatarError }}</div>
        </article>
      </section>

      <section v-if="state.tab === 'reports'" class="admin-two">
        <article class="panel span-2 report-summary-panel">
          <div>
            <h2>游客感受度报告</h2>
            <p class="hint-line">基于近 7 天数字人、语音和网页问答交互记录自动生成。</p>
          </div>
          <button class="secondary-action" :disabled="state.reportLoading" @click="loadVisitorReport">刷新报告</button>
          <div v-if="state.reportError" class="notice error">{{ state.reportError }}</div>
        </article>

        <section class="kpi-grid four span-2">
          <KpiCard label="交互记录" :value="String(state.report.summary.total_interactions)" note="近 7 天累计" />
          <KpiCard label="满意倾向" :value="`${Math.round(state.report.summary.satisfaction_rate)}%`" note="正向情绪占比" tone="green" />
          <KpiCard label="负面占比" :value="`${Math.round(state.report.summary.negative_rate)}%`" note="需重点复盘" tone="red" />
          <KpiCard label="高峰时段" :value="state.report.summary.peak_hour" :note="`关注点：${state.report.summary.top_concern}`" tone="gold" />
        </section>

        <article class="panel span-2">
          <h2>游客关注点分析</h2>
          <div v-if="state.reportLoading" class="muted-center">正在生成报告...</div>
          <BarList v-else :items="attentionBars" />
        </article>

        <article class="panel">
          <h2>情绪分布</h2>
          <DonutChart :items="emotionDonutItems" :center="`${Math.round(state.report.summary.satisfaction_rate)}%`" />
        </article>

        <article class="panel">
          <h2>情感趋势</h2>
          <div class="emotion-trend">
            <div v-for="item in state.report.emotion_trend" :key="item.date" class="trend-day">
              <span>{{ item.date.slice(5) }}</span>
              <div class="trend-stack">
                <i class="positive" :style="{ height: `${item.positive_rate}%` }" />
                <i class="negative" :style="{ height: `${item.negative_rate}%` }" />
              </div>
              <small>{{ item.total }}</small>
            </div>
          </div>
        </article>

        <article class="panel">
          <h2>热门时段</h2>
          <BarList :items="peakHourBars" />
        </article>

        <article class="panel">
          <h2>服务建议</h2>
          <ul class="clean-list">
            <li v-for="item in state.report.suggestions" :key="item.content">{{ item.content }}</li>
          </ul>
        </article>
      </section>

      <section v-if="state.tab === 'settings'" class="panel form-panel">
        <p v-if="state.settingsError" class="msg err">{{ state.settingsError }}</p>
        <p v-if="state.settingsMessage" class="msg ok">{{ state.settingsMessage }}</p>
        <label>景区名称<input v-model="state.settings.scenic_name" /></label>
        <label>景区简介<textarea v-model="state.settings.scenic_desc" rows="3"></textarea></label>
        <label>服务热线<input v-model="state.settings.service_hotline" /></label>
        <label>数据保留天数<input v-model="state.settings.data_retention" /></label>
        <label>备份频率<input v-model="state.settings.backup_frequency" /></label>
        <label class="toggle-row"><input type="checkbox" v-model="state.settings.enable_login" /> 启用用户登录</label>
        <label class="toggle-row"><input type="checkbox" v-model="state.settings.enable_voice" /> 启用语音服务</label>
        <label class="toggle-row"><input type="checkbox" v-model="state.settings.enable_filter" /> 启用游客感受度分析</label>
        <div class="action-row">
          <button class="btn primary" :disabled="state.settingsSaving" @click="saveSettings">
            {{ state.settingsSaving ? '保存中...' : '保存设置' }}
          </button>
          <button class="btn" @click="loadSettings">重置</button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.admin-view {
  padding: 24px;
  background: #0a0a0f;
  min-height: 100%;
}
.admin-header {
  margin-bottom: 16px;
}
.admin-header h1 {
  font-size: 20px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.88);
  margin-bottom: 4px;
}
.admin-header p {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.35);
}
.admin-content {
  margin-top: 16px;
}

/* 保留原有子组件样式 */
.panel {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 16px;
}
.panel h2 {
  font-size: 15px;
  color: rgba(255, 255, 255, 0.88);
  margin-bottom: 16px;
}

.form-panel label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.55);
}
.form-panel input,
.form-panel select,
.form-panel textarea {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  padding: 10px 12px;
  color: white;
  font-size: 14px;
  outline: none;
  font-family: inherit;
  transition: border-color 0.2s;
}
.form-panel input:focus,
.form-panel select:focus,
.form-panel textarea:focus {
  border-color: #63e2b7;
}

.button-row {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}
.btn {
  padding: 8px 16px;
  border-radius: 6px;
  border: none;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}
.btn.primary {
  background: #63e2b7;
  color: #0a0a0f;
  font-weight: 600;
}
.btn.primary:hover { background: #4fd1a0; }
.btn.primary:disabled { opacity: 0.5; cursor: not-allowed; }

.primary-action {
  padding: 10px 20px;
  background: linear-gradient(135deg, #63e2b7, #18a058);
  border: none;
  border-radius: 8px;
  color: #0a0a0f;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.primary-action:disabled { opacity: 0.5; cursor: not-allowed; }

.secondary-action {
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.88);
  font-size: 14px;
  cursor: pointer;
}

.msg { padding: 8px 12px; border-radius: 6px; font-size: 13px; margin-bottom: 12px; }
.msg.ok { background: rgba(99, 226, 183, 0.1); color: #63e2b7; border: 1px solid rgba(99, 226, 183, 0.2); }
.msg.err { background: rgba(232, 128, 128, 0.1); color: #e88080; border: 1px solid rgba(232, 128, 128, 0.2); }

.knowledge-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.knowledge-table th {
  text-align: left;
  padding: 10px 12px;
  color: rgba(255, 255, 255, 0.4);
  font-weight: 500;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 12px;
}
.knowledge-table td {
  padding: 10px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.75);
}
.knowledge-table tr:hover td {
  background: rgba(255, 255, 255, 0.02);
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.65);
  margin-bottom: 8px;
}

.action-row {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}
</style>
